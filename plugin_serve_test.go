package pluginsdk

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type hookPlugin struct {
	lifecyclePlugin
	activations, readies atomic.Int32
	readyErr             error
}

func (p *hookPlugin) CheckActivation(context.Context) error {
	p.activations.Add(1)
	return p.activationErr
}
func (p *hookPlugin) Ready(context.Context) error { p.readies.Add(1); return p.readyErr }

// hookHost acknowledges every declared hook, or plays an older host that
// rejects the unknown hooks field.
type hookHost struct {
	lifecycleHost
	mu          sync.Mutex
	requests    []ProtocolRequirements
	rejectHooks bool
}

func (h *hookHost) Negotiate(_ context.Context, need ProtocolRequirements) (HostCapabilities, error) {
	h.mu.Lock()
	h.requests = append(h.requests, need)
	h.mu.Unlock()
	if h.rejectHooks && len(need.Hooks) != 0 {
		return HostCapabilities{}, errors.New("host /negotiate: 400 Bad Request")
	}
	return HostCapabilities{Major: ProtocolMajor, PluginID: "sample", Generation: "g1", Hooks: need.Hooks}, nil
}

func TestServePluginLeavesAcknowledgedHooksToTheHostVerb(t *testing.T) {
	t.Setenv(EnvID, "sample")
	sock := shortUnixSocketPath(t)
	p := &hookPlugin{lifecyclePlugin: lifecyclePlugin{activationErr: errors.New("not admitted")}, readyErr: errors.New("cache cold")}
	host := &hookHost{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ServePlugin(ctx, p, PluginConfig{Socket: sock, Host: host}) }()
	for {
		if conn, err := net.Dial("unix", sock); err == nil {
			conn.Close()
			break
		}
		select {
		case err := <-done:
			t.Fatalf("exited before listening: %v", err)
		case <-time.After(5 * time.Millisecond):
		}
	}
	if p.activations.Load() != 0 || p.readies.Load() != 0 {
		t.Fatal("acknowledged hooks also ran locally")
	}
	// A successful socket dial is not a Go memory synchronization boundary.
	// Read the recorder under the same mutex used by Negotiate, and copy its
	// slice before asserting outside the lock.
	host.mu.Lock()
	requestCount := len(host.requests)
	var advertisedHooks []string
	if requestCount > 0 {
		advertisedHooks = slices.Clone(host.requests[0].Hooks)
	}
	host.mu.Unlock()
	if requestCount != 1 {
		t.Fatalf("negotiation request count = %d, want 1", requestCount)
	}
	if got := advertisedHooks; !slices.Equal(got, []string{HookActivationCheck, HookReady}) {
		t.Fatalf("advertised hooks = %v", got)
	}
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	call := func(hook string) (int, string) {
		resp, err := client.Post("http://plugin/"+LifecycleHookCommand, "application/json", strings.NewReader(`{"hook":"`+hook+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var reply struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&reply)
		return resp.StatusCode, reply.Error
	}
	for hook, want := range map[string]string{HookActivationCheck: "not admitted", HookReady: "cache cold"} {
		if code, reason := call(hook); code != http.StatusOK || reason != want {
			t.Fatalf("%s = %d %q", hook, code, reason)
		}
	}
	if code, _ := call("stop"); code != http.StatusNotFound {
		t.Fatalf("undeclared hook answered %d", code)
	}
	if p.activations.Load() != 1 || p.readies.Load() != 1 {
		t.Fatalf("host verb dispatch: activations=%d readies=%d", p.activations.Load(), p.readies.Load())
	}
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !p.stopped {
		t.Fatal("plugin not stopped")
	}
}

func TestServePluginRetriesWithoutHooksForAnOlderHost(t *testing.T) {
	t.Setenv(EnvID, "sample")
	failure := errors.New("cannot activate")
	p := &hookPlugin{lifecyclePlugin: lifecyclePlugin{activationErr: failure, protocol: &ProtocolRequirements{Major: 1}}}
	host := &hookHost{rejectHooks: true}
	err := ServePlugin(context.Background(), p, PluginConfig{Socket: filepath.Join(t.TempDir(), "p.sock"), Host: host})
	if !errors.Is(err, failure) || p.activations.Load() != 1 {
		t.Fatalf("hooks did not run locally after fallback: err=%v activations=%d", err, p.activations.Load())
	}
	if len(host.requests) != 2 || len(host.requests[0].Hooks) != 2 || host.requests[1].Hooks != nil {
		t.Fatalf("negotiation attempts = %+v", host.requests)
	}
}

type lifecyclePlugin struct {
	startErr, activationErr, stopErr error
	started, stopped                 bool
	protocol                         *ProtocolRequirements
}

type subjectHTTPPlugin struct {
	lifecyclePlugin
	handler http.Handler
}

func (p *subjectHTTPPlugin) Handler() http.Handler { return p.handler }

func TestServePluginCarriesHostSubjectLeaseIntoHTTPPluginContext(t *testing.T) {
	t.Setenv(EnvID, "sample")
	sock := shortUnixSocketPath(t)
	seen := make(chan string, 1)
	p := &subjectHTTPPlugin{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lease, err := SubjectLeaseFromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		seen <- lease
		w.WriteHeader(http.StatusNoContent)
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ServePlugin(ctx, p, PluginConfig{Socket: sock, Host: lifecycleHost{}}) }()
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	var resp *http.Response
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		req, err := http.NewRequest(http.MethodGet, "http://plugin/plugin", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(HeaderSubjectLease, testSubjectLease)
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if resp == nil {
		t.Fatal("plugin did not listen")
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := <-seen; got != testSubjectLease {
		t.Fatalf("handler lease = %q", got)
	}
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func (*lifecyclePlugin) ID() string                              { return "sample" }
func (p *lifecyclePlugin) Manifest() Manifest                    { return Manifest{ID: p.ID(), Protocol: p.protocol} }
func (p *lifecyclePlugin) Start(context.Context, Host) error     { p.started = true; return p.startErr }
func (p *lifecyclePlugin) CheckActivation(context.Context) error { return p.activationErr }
func (p *lifecyclePlugin) Stop(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("Stop requires deadline")
	}
	p.stopped = true
	return p.stopErr
}

type lifecycleHost struct{ info HostCapabilities }

func (lifecycleHost) Publish(Envelope) error                              { return nil }
func (lifecycleHost) Subscribe(string, func(Envelope)) func()             { return func() {} }
func (lifecycleHost) Request(context.Context, Envelope) (Envelope, error) { return Envelope{}, nil }
func (lifecycleHost) Logger() Logger                                      { return nil }
func (lifecycleHost) Profiling() Profiler                                 { return NopProfiler() }
func (lifecycleHost) StateDir(string) string                              { return "" }
func (lifecycleHost) Every(time.Duration, func()) func()                  { return func() {} }
func (lifecycleHost) BumpContributions()                                  {}
func (h lifecycleHost) Negotiate(context.Context, ProtocolRequirements) (HostCapabilities, error) {
	return h.info, nil
}

func TestServePluginRollsBackPartialStartAndActivationBeforeListening(t *testing.T) {
	t.Setenv(EnvID, "sample")
	for _, failStart := range []bool{true, false} {
		failure, cleanup := errors.New("cannot activate"), errors.New("cleanup failure")
		p := &lifecyclePlugin{stopErr: cleanup}
		if failStart {
			p.startErr = failure
		} else {
			p.activationErr = failure
		}
		socket := filepath.Join(t.TempDir(), "plugin.sock")
		err := ServePlugin(context.Background(), p, PluginConfig{Socket: socket, Host: lifecycleHost{}})
		if !errors.Is(err, failure) || !errors.Is(err, cleanup) {
			t.Fatalf("lost lifecycle errors: %v", err)
		}
		if !p.started || !p.stopped {
			t.Fatal("partial generation was not stopped")
		}
		if _, err := os.Stat(socket); !os.IsNotExist(err) {
			t.Fatal("failed generation opened a serving socket")
		}
	}
}

func TestServePluginChecksDirectNegotiationBeforeStart(t *testing.T) {
	t.Setenv(EnvID, "sample")
	for _, info := range []HostCapabilities{
		{Major: 1, PluginID: "sample", Generation: "g1"},
		{Major: 1, PluginID: "foreign", Generation: "g1", Features: []string{FeatureScopedHost}},
		{Major: 1, PluginID: "sample", Features: []string{FeatureScopedHost}},
	} {
		p := &lifecyclePlugin{protocol: &ProtocolRequirements{Major: 1, Required: []string{FeatureScopedHost}}}
		err := ServePlugin(context.Background(), p, PluginConfig{Host: lifecycleHost{info: info}})
		if err == nil || p.started || p.stopped {
			t.Fatalf("unsupported host reached lifecycle: err=%v plugin=%+v", err, p)
		}
	}
}
