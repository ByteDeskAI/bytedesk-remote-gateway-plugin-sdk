package pluginsdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

// EnvHostSocket is the socket a spawned plugin dials to reach the host. It is
// the counterpart of EnvSocket: the host dials EnvSocket to send HTTP into the
// plugin, and the plugin dials this to call back (ADR 0024).
const EnvHostSocket = "GATEWAY_HOST_SOCKET"

// rpcHost implements plugin.Host by carrying each call over the host socket. A
// plugin holds this or the host's in-process implementation and cannot tell
// which, which is what lets `spawn` be a manifest field rather than a rewrite.
//
// Subscribe and Every need traffic from host to plugin. Rather than the plugin
// running a second listener for callbacks, it holds one streaming GET open and
// the host writes newline-delimited JSON down it. So a dropped connection ends
// every subscription and timer at once — the right semantics when a plugin dies
// mid-subscription, and no liveness tracking on either side.
type rpcHost struct {
	socket string
	client *http.Client

	mu        sync.Mutex
	subs      map[string]func(bus.Envelope) // subscription id -> handler
	ticks     map[string]func()             // timer id -> callback
	started   bool
	stop      context.CancelFunc
	logger    plugin.Logger
	lastFault error
	seq       atomic.Int64

	// Receive-side counters, so a callback that never reaches its handler can
	// be told apart from one that was never sent.
	received  atomic.Int64
	unmatched atomic.Int64
	streamErr atomic.Value // error
}

// NewHost returns a plugin.Host backed by the host socket. Pass an empty socket
// to read GATEWAY_HOST_SOCKET from the environment. Close the returned host by
// cancelling the context passed to Serve.
func NewHost(socket string) plugin.Host {
	h := &rpcHost{
		socket: socket,
		subs:   map[string]func(bus.Envelope){},
		ticks:  map[string]func(){},
	}
	h.client = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", h.socket)
			},
		},
	}
	h.logger = rpcLogger{h: h}
	return h
}

func (h *rpcHost) url(path string) string { return "http://host" + path }

func (h *rpcHost) post(ctx context.Context, path string, body, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url(path), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("host %s: %s", path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Start opens the callback stream. Serve calls this; a plugin does not.
func (h *rpcHost) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = true
	ctx, cancel := context.WithCancel(ctx)
	h.stop = cancel
	h.mu.Unlock()

	ready := make(chan error, 1)
	go h.readCallbacks(ctx, ready)
	select {
	case err := <-ready:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("host callback stream did not open")
	}
}

func (h *rpcHost) readCallbacks(ctx context.Context, ready chan<- error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url("/callbacks"), nil)
	if err != nil {
		ready <- err
		return
	}
	resp, err := h.client.Do(req)
	if err != nil {
		ready <- err
		return
	}
	defer resp.Body.Close()
	ready <- nil

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	defer func() {
		if err := sc.Err(); err != nil {
			h.streamErr.Store(err)
		}
	}()
	for sc.Scan() {
		var cb struct {
			Kind     string       `json:"kind"`
			ID       string       `json:"id"`
			Envelope bus.Envelope `json:"envelope"`
		}
		if json.Unmarshal(sc.Bytes(), &cb) != nil {
			continue
		}
		h.received.Add(1)
		h.mu.Lock()
		fn := h.subs[cb.ID]
		tick := h.ticks[cb.ID]
		h.mu.Unlock()
		switch cb.Kind {
		case "event":
			if fn != nil {
				fn(cb.Envelope)
			} else {
				h.unmatched.Add(1)
			}
		case "tick":
			if tick != nil {
				tick()
			} else {
				h.unmatched.Add(1)
			}
		}
	}
}

func (h *rpcHost) Publish(env bus.Envelope) error {
	return h.post(context.Background(), "/publish", env, nil)
}

func (h *rpcHost) Request(ctx context.Context, env bus.Envelope) (bus.Envelope, error) {
	var reply bus.Envelope
	err := h.post(ctx, "/request", env, &reply)
	return reply, err
}

func (h *rpcHost) Subscribe(eventType string, fn func(bus.Envelope)) (unsubscribe func()) {
	// The id is generated here and sent to the host, rather than assigned by
	// the host and returned. The host starts delivering as soon as it
	// registers, so if it chose the id there would be a window between the
	// first callback arriving and this side learning which id it belongs to —
	// and a callback for an unknown id is dropped. Registering first closes it.
	id := h.newID("sub")
	h.mu.Lock()
	h.subs[id] = fn
	h.mu.Unlock()

	body := map[string]string{"id": id, "eventType": eventType}
	if err := h.post(context.Background(), "/subscribe", body, nil); err != nil {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
		// A no-op unsubscribe is the only thing the interface allows us to
		// return, so record why: a silently dropped subscription is otherwise
		// indistinguishable from an event that never happened.
		h.recordFault("subscribe "+eventType, err)
		return func() {}
	}
	return func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
		_ = h.post(context.Background(), "/cancel", map[string]string{"id": id}, nil)
	}
}

func (h *rpcHost) Every(interval time.Duration, fn func()) (cancel func()) {
	// Registered before the host is asked, for the same reason as Subscribe:
	// the host's timer starts inside the request handler, and the host's timer
	// wheel is coarse, so a tick can easily arrive before the response does.
	// Losing it means the caller waits a whole interval for the next one — or,
	// when the caller only needs one tick, forever.
	id := h.newID("every")
	h.mu.Lock()
	h.ticks[id] = fn
	h.mu.Unlock()

	body := map[string]any{"id": id, "intervalMs": interval.Milliseconds()}
	if err := h.post(context.Background(), "/every", body, nil); err != nil {
		h.mu.Lock()
		delete(h.ticks, id)
		h.mu.Unlock()
		h.recordFault("every", err)
		return func() {}
	}
	return func() {
		h.mu.Lock()
		delete(h.ticks, id)
		h.mu.Unlock()
		_ = h.post(context.Background(), "/cancel", map[string]string{"id": id}, nil)
	}
}

// newID mints a registration id unique to this plugin process.
func (h *rpcHost) newID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(h.seq.Add(1), 10)
}

func (h *rpcHost) StateDir(pluginID string) string {
	var out struct {
		Dir string `json:"dir"`
	}
	req, err := http.NewRequest(http.MethodGet, h.url("/statedir?plugin="+pluginID), nil)
	if err != nil {
		return ""
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return ""
	}
	return out.Dir
}

func (h *rpcHost) Logger() plugin.Logger { return h.logger }

func (h *rpcHost) BumpContributions() {
	_ = h.post(context.Background(), "/bump", struct{}{}, nil)
}

// rpcLogger forwards to the host's logger so a spawned plugin's lines land in
// the same place an in-process plugin's do, rather than only on its stderr.
type rpcLogger struct{ h *rpcHost }

func (l rpcLogger) log(level, msg string, args ...any) {
	_ = l.h.post(context.Background(), "/log", map[string]any{"level": level, "msg": msg, "args": args}, nil)
}
func (l rpcLogger) Info(msg string, args ...any)  { l.log("info", msg, args...) }
func (l rpcLogger) Warn(msg string, args ...any)  { l.log("warn", msg, args...) }
func (l rpcLogger) Error(msg string, args ...any) { l.log("error", msg, args...) }

// HostStarter is implemented by a Host that needs opening before use.
type HostStarter interface {
	Start(ctx context.Context) error
}

// HostCloser is implemented by a Host holding resources worth releasing.
type HostCloser interface {
	Close()
	LastFault() error
	Counters() (received, unmatched int64, streamErr error)
}

// recordFault stores the most recent registration failure. Publish and Request
// return errors to their caller; Subscribe and Every cannot, because the
// interface returns only a cancel function, so their failures land here rather
// than vanishing — a silently dropped subscription is otherwise
// indistinguishable from an event that never happened.
func (h *rpcHost) recordFault(what string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err == nil {
		err = errors.New("host returned no id")
	}
	h.lastFault = fmt.Errorf("%s: %w", what, err)
}

// Counters reports callbacks received, callbacks whose id matched no handler,
// and any error that ended the callback stream.
func (h *rpcHost) Counters() (received, unmatched int64, streamErr error) {
	if v, ok := h.streamErr.Load().(error); ok {
		streamErr = v
	}
	return h.received.Load(), h.unmatched.Load(), streamErr
}

// LastFault returns the most recent Subscribe or Every failure, or nil.
func (h *rpcHost) LastFault() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastFault
}

// Close stops the callback stream and releases the transport's pooled
// connections. Without it a process that builds several hosts — tests, most
// obviously — leaks a connection pool per host and eventually cannot dial,
// which surfaces as a subscription or timer that silently never fires.
func (h *rpcHost) Close() {
	h.mu.Lock()
	stop := h.stop
	h.stop = nil
	h.started = false
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
	if tr, ok := h.client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

var _ plugin.Host = (*rpcHost)(nil)
var _ HostStarter = (*rpcHost)(nil)
var _ HostCloser = (*rpcHost)(nil)
