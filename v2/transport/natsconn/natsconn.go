// Package natsconn is the NATS transport for a spawned SDK v2 plugin, and the
// ONLY package in either SDK that imports github.com/nats-io/*.
//
// That rule is the whole point of the package (R2). A plugin reaches the
// gateway through bus.Bus and nothing else; if a nats type appeared in an
// exported signature anywhere, every plugin in the fleet would acquire nats.go
// in its dependency graph and the substrate would stop being swappable. The
// boundary is asserted by TestOnlyNatsconnImportsBroker and
// TestNoBrokerTypeInPublicSignatures in the module root, and those tests are
// watched failing before they are trusted.
//
// What crosses the boundary is a *Conn, which implements bus.Bus.
//
// A contained plugin has no network namespace (the host spawns it under
// --unshare-net), so TCP to a loopback broker is impossible by construction.
// The only reachable path is the unix socket in the plugin's own run dir, which
// the host splices to the in-process server. Dial therefore always dials unix
// and there is deliberately no TCP option: adding one would be a hole, not a
// convenience.
package natsconn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	nats "github.com/nats-io/nats.go"
)

// Options configure Dial. Socket and PluginID are required; a Dial without
// CredsFile is refused, because an unauthenticated connection to the bus is
// not a degraded mode, it is a different security model.
type Options struct {
	// Socket is the unix socket the host splices to the broker
	// (GATEWAY_BUS_SOCKET).
	Socket string
	// CredsFile is the standard .creds file the host wrote for this
	// generation (GATEWAY_BUS_CREDS). It is 0400 and belongs to this process.
	CredsFile string
	// PluginID is this plugin's id. It names the custom inbox prefix, so a
	// reply to one plugin can never be delivered to another.
	PluginID string
	// Generation is the host's generation id, used only to name the
	// connection for operator-visible server output.
	Generation string
	// ConnectTimeout bounds the initial CONNECT. Zero means 5s.
	ConnectTimeout time.Duration
	// ReconnectWait is the pause between reconnect attempts. Zero means 250ms.
	// Reconnect attempts themselves are unlimited: a plugin that gave up after
	// N tries would be a plugin an operator has to restart by hand after a
	// broker bounce.
	ReconnectWait time.Duration
}

// Conn is a plugin's bus connection. It implements bus.Bus.
type Conn struct {
	nc    *nats.Conn
	id    string
	inbox string

	mu sync.RWMutex
	// caps are the NEGOTIATED capabilities, installed once by Negotiated.
	// Until then Conn reports only what the transport itself implements, so a
	// caller that skipped the handshake cannot be told the substrate is
	// durable.
	caps  bus.Capabilities
	bound bool
	// deniedPublish records subjects the server refused a publish on. A core
	// publish is asynchronous, so the refusal arrives after the call that
	// caused it returned; recording it here is what turns it into a loud
	// failure on the NEXT call for that subject instead of silence forever.
	deniedPublish map[string]string
	subs          map[*subscription]struct{}

	closeOnce sync.Once
}

// TransportCapabilities are the features THIS transport implements today.
// The negotiated set is intersected with it, so a plugin is never told the
// bus can do something its own connection cannot.
//
// ponytail: streams, KV, objects and schedule are not implemented in rc.1.
// They are reported false rather than stubbed true, which makes a manifest
// that needs them fail closed at the handshake with a named capability
// instead of at first use with a nil dereference. The JetStream surfaces land
// with the durable re-platforms (plan 3); add them here and the intersection
// starts admitting them with no other change.
func TransportCapabilities() bus.Capabilities {
	return bus.Capabilities{
		Services:   true,
		MaxPayload: 64 << 10,
	}
}

// Dial connects to the broker over the host's unix socket and returns a bus.
//
// Every failure comes back as a bus.Fault. A credential the server refuses —
// revoked, expired, or for an account that no longer exists — is
// bus.FaultDenied naming the socket and the plugin, because that is the one
// failure an operator has to be able to read off a log line without a debugger.
func Dial(ctx context.Context, opts Options) (*Conn, error) {
	if strings.TrimSpace(opts.Socket) == "" {
		return nil, bus.Fault{Code: bus.FaultUnavailable, Op: "dial", Message: "bus socket required"}
	}
	if strings.TrimSpace(opts.CredsFile) == "" {
		return nil, bus.Fault{Code: bus.FaultDenied, Op: "dial", Message: "principal " + opts.PluginID + ": bus credentials required"}
	}
	if strings.TrimSpace(opts.PluginID) == "" {
		return nil, bus.Fault{Code: bus.FaultDenied, Op: "dial", Message: "plugin id required"}
	}
	timeout := opts.ConnectTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	wait := opts.ReconnectWait
	if wait <= 0 {
		wait = 250 * time.Millisecond
	}
	c := &Conn{
		id:            opts.PluginID,
		inbox:         "_INBOX." + opts.PluginID,
		caps:          TransportCapabilities(),
		deniedPublish: map[string]string{},
		subs:          map[*subscription]struct{}{},
	}
	name := opts.PluginID
	if opts.Generation != "" {
		name += "@" + opts.Generation
	}
	nc, err := nats.Connect(
		"nats://"+placeholderHost,
		nats.Name(name),
		nats.SetCustomDialer(unixDialer{path: opts.Socket, timeout: timeout}),
		nats.UserCredentials(opts.CredsFile),
		// A reply to this plugin is delivered under its own inbox prefix, so a
		// grant that covers _INBOX.<id>.> covers every reply it can receive and
		// covers no other plugin's.
		nats.CustomInboxPrefix(c.inbox),
		nats.Timeout(timeout),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(wait),
		nats.ErrorHandler(c.onAsyncError),
		nats.NoCallbacksAfterClientClose(),
	)
	if err != nil {
		return nil, dialFault(opts, err)
	}
	c.nc = nc
	if err := ctx.Err(); err != nil {
		nc.Close()
		return nil, err
	}
	return c, nil
}

// placeholderHost is what the URL carries because nats.Connect requires one.
// The custom dialer ignores it and dials the unix socket; nothing resolves it.
const placeholderHost = "gateway-bus"

type unixDialer struct {
	path    string
	timeout time.Duration
}

func (d unixDialer) Dial(_, _ string) (net.Conn, error) {
	return net.DialTimeout("unix", d.path, d.timeout)
}

// dialFault maps a connect failure onto the bus vocabulary. An authorization
// failure is FaultDenied and everything else is FaultUnavailable, so a caller
// can tell "you may not" from "it is not there" without reading error text.
func dialFault(opts Options, err error) bus.Fault {
	switch {
	case errors.Is(err, nats.ErrAuthorization),
		errors.Is(err, nats.ErrAuthExpired),
		errors.Is(err, nats.ErrAuthRevoked),
		errors.Is(err, nats.ErrAccountAuthExpired):
		f := bus.Denied(opts.Socket, opts.PluginID, "bus credential refused at CONNECT: "+err.Error())
		f.Err = err
		return f
	default:
		return bus.Fault{Code: bus.FaultUnavailable, Op: opts.Socket, Message: "bus connect failed: " + err.Error(), Err: err}
	}
}

// Negotiated installs the capabilities the host reported, intersected with what
// this transport implements. It may be called once; a second call is ignored,
// because capabilities are a property of the generation and a generation is
// negotiated exactly once.
func (c *Conn) Negotiated(host bus.Capabilities) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bound {
		return
	}
	c.bound = true
	c.caps = intersect(host, TransportCapabilities())
}

func intersect(a, b bus.Capabilities) bus.Capabilities {
	out := bus.Capabilities{
		Durable:  a.Durable && b.Durable,
		KV:       a.KV && b.KV,
		Objects:  a.Objects && b.Objects,
		Services: a.Services && b.Services,
		Schedule: a.Schedule && b.Schedule,
		Counters: a.Counters && b.Counters,
		Batch:    a.Batch && b.Batch,
		Trace:    a.Trace && b.Trace,
	}
	out.MaxPayload = a.MaxPayload
	if out.MaxPayload <= 0 || (b.MaxPayload > 0 && b.MaxPayload < out.MaxPayload) {
		out.MaxPayload = b.MaxPayload
	}
	return out
}

// Capabilities reports the negotiated set.
func (c *Conn) Capabilities() bus.Capabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.caps
}

// Publish emits to a concrete subject.
//
// A core publish is asynchronous under a broker: the server refuses it with an
// async -ERR, long after this call returned nil. The denied set is what makes
// that refusal reachable — the next publish or request for the same subject
// returns the recorded FaultDenied rather than succeeding into a void.
func (c *Conn) Publish(ctx context.Context, subject bus.Subject, data []byte, opts ...bus.PublishOpt) error {
	if _, err := bus.ParseSubject(string(subject)); err != nil {
		return bus.Fault{Code: bus.FaultDenied, Op: string(subject), Message: "principal " + c.id + ": " + err.Error(), Err: err}
	}
	if err := c.denied(string(subject)); err != nil {
		return err
	}
	o := bus.ResolvePublish(opts)
	if max := c.Capabilities().MaxPayload; max > 0 && len(data) > max {
		return bus.Fault{Code: bus.FaultBudget, Op: string(subject),
			Message: fmt.Sprintf("principal %s: payload %d exceeds the %d byte ceiling", c.id, len(data), max)}
	}
	m := nats.NewMsg(string(subject))
	m.Data = data
	m.Header = toNATSHeader(o.Headers)
	if o.MsgID != "" {
		m.Header.Set(natsMsgID, o.MsgID)
	}
	if err := c.nc.PublishMsg(m); err != nil {
		return c.fault(bus.FaultUnavailable, string(subject), err)
	}
	return nil
}

// Request sends and waits for one reply.
func (c *Conn) Request(ctx context.Context, subject bus.Subject, data []byte, opts ...bus.ReqOpt) (*bus.Msg, error) {
	if _, err := bus.ParseSubject(string(subject)); err != nil {
		return nil, bus.Fault{Code: bus.FaultDenied, Op: string(subject), Message: "principal " + c.id + ": " + err.Error(), Err: err}
	}
	if err := c.denied(string(subject)); err != nil {
		return nil, err
	}
	o := bus.ResolveReq(opts)
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = bus.DefaultTimeout
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	m := nats.NewMsg(string(subject))
	m.Data = data
	m.Header = toNATSHeader(o.Headers)
	// Forwarded raw, exactly as the v1 SDK's rpcHost.Request does: the host
	// alone validates a present value (ContextForRequest/SubjectLeaseFromContext
	// on the receiving end), so this client never duplicates that judgment and
	// the two can never drift apart on what counts as canonical.
	if transport, ok := subjectLeaseTransportFromContext(ctx); ok && transport.present {
		m.Header[HeaderSubjectLease] = append([]string(nil), transport.values...)
	}
	reply, err := c.nc.RequestMsgWithContext(ctx, m)
	if err != nil {
		return nil, c.requestFault(string(subject), err)
	}
	return c.inbound(reply), nil
}

// Subscribe delivers every message matching pattern until cancelled or drained.
func (c *Conn) Subscribe(ctx context.Context, pattern bus.Pattern, h bus.Handler, opts ...bus.SubOpt) (bus.Subscription, error) {
	if _, err := bus.ParsePattern(string(pattern)); err != nil {
		return nil, bus.Fault{Code: bus.FaultDenied, Op: string(pattern), Message: "principal " + c.id + ": " + err.Error(), Err: err}
	}
	if h == nil {
		return nil, bus.Fault{Code: bus.FaultUnhandled, Op: string(pattern), Message: "handler required"}
	}
	o := bus.ResolveSub(opts)
	limit := o.PendingLimit
	if limit <= 0 {
		limit = bus.DefaultPendingLimit
	}
	s := &subscription{c: c, subject: string(pattern), queue: o.QueueGroup, done: make(chan struct{})}
	cb := func(m *nats.Msg) { h(ctx, c.inbound(m)) }
	var (
		ns  *nats.Subscription
		err error
	)
	if o.QueueGroup != "" {
		ns, err = c.nc.QueueSubscribe(string(pattern), o.QueueGroup, cb)
	} else {
		ns, err = c.nc.Subscribe(string(pattern), cb)
	}
	if err != nil {
		return nil, c.fault(bus.FaultUnavailable, string(pattern), err)
	}
	// nats.go defaults a subscription's pending queue to 64 MB, which turns a
	// stalled handler into unbounded memory growth. bus.DefaultPendingLimit
	// makes the overflow a countable FaultSlowConsumer instead.
	if err := ns.SetPendingLimits(limit, -1); err != nil {
		_ = ns.Unsubscribe()
		return nil, c.fault(bus.FaultUnavailable, string(pattern), err)
	}
	s.ns = ns
	c.mu.Lock()
	c.subs[s] = struct{}{}
	c.mu.Unlock()
	return s, nil
}

func (c *Conn) Services() bus.Services  { return &services{c: c} }
func (c *Conn) Trace() bus.Trace        { return tracer{} }
func (c *Conn) Streams() bus.Streams    { return noStreams{} }
func (c *Conn) KV() bus.KV              { return noKV{} }
func (c *Conn) Objects() bus.Objects    { return noObjects{} }
func (c *Conn) Schedule() bus.Scheduler { return noScheduler{} }

// Close releases the connection. It ends every live subscription with
// FaultWithdrawn first, so a plugin waiting on Done is released rather than
// left blocked on a connection that has gone away.
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		subs := make([]*subscription, 0, len(c.subs))
		for s := range c.subs {
			subs = append(subs, s)
		}
		c.mu.Unlock()
		for _, s := range subs {
			s.finish(bus.Fault{Code: bus.FaultWithdrawn, Op: s.subject, Message: "principal " + c.id + ": connection closed"})
		}
		if c.nc != nil {
			c.nc.Close()
		}
	})
	return nil
}

// publishTo publishes with an explicit reply subject. Discovery needs it and
// bus.Publish deliberately has no reply option: a plugin that could set one
// could redirect another principal's replies to itself.
func (c *Conn) publishTo(subject, reply string, data []byte) error {
	m := nats.NewMsg(subject)
	m.Reply = reply
	m.Data = data
	if err := c.nc.PublishMsg(m); err != nil {
		return c.fault(bus.FaultUnavailable, subject, err)
	}
	return nil
}

// InboxPrefix is this connection's reply namespace. The handshake needs it to
// tell the host which inbox family to grant.
func (c *Conn) InboxPrefix() string { return c.inbox }

// onAsyncError is where a broker's out-of-band refusals become Faults.
//
// The server reports a permissions violation with an -ERR that names the
// operation and the subject, and nats.go hands it to this callback with a nil
// subscription in every case. So the subject is parsed out of the text and the
// refusal routed by hand: a subscription violation ends that subscription, and
// a publish violation is recorded against the subject for the next call.
func (c *Conn) onAsyncError(_ *nats.Conn, ns *nats.Subscription, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, nats.ErrSlowConsumer) {
		if s := c.findByNATS(ns); s != nil {
			s.finish(bus.Fault{Code: bus.FaultSlowConsumer, Op: s.subject,
				Message: "principal " + c.id + ": subscription exceeded its pending limit", Err: err})
		}
		return
	}
	if !errors.Is(err, nats.ErrPermissionViolation) {
		return
	}
	op, subject := parseViolation(err.Error())
	if subject == "" {
		return
	}
	switch op {
	case "subscription":
		for _, s := range c.findBySubject(subject) {
			s.finish(bus.Denied(subject, c.id, "grant does not cover this subscription"))
		}
	default:
		c.mu.Lock()
		c.deniedPublish[subject] = op
		c.mu.Unlock()
	}
}

// violationRe reads the operation and subject out of the server's -ERR text,
// e.g. `Permissions Violation for Publish to "cmd.files.v1.list"`.
var violationRe = regexp.MustCompile(`(?i)permissions violation for (\w+) to "([^"]+)"`)

func parseViolation(s string) (op, subject string) {
	m := violationRe.FindStringSubmatch(s)
	if len(m) != 3 {
		return "", ""
	}
	return strings.ToLower(m[1]), m[2]
}

func (c *Conn) denied(subject string) error {
	c.mu.RLock()
	op, ok := c.deniedPublish[subject]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	return bus.Denied(subject, c.id, "the broker refused a previous "+op+" on this subject")
}

func (c *Conn) findByNATS(ns *nats.Subscription) *subscription {
	if ns == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for s := range c.subs {
		if s.ns == ns {
			return s
		}
	}
	return nil
}

func (c *Conn) findBySubject(subject string) []*subscription {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []*subscription
	for s := range c.subs {
		if s.subject == subject {
			out = append(out, s)
		}
	}
	return out
}

func (c *Conn) forget(s *subscription) {
	c.mu.Lock()
	delete(c.subs, s)
	c.mu.Unlock()
}

func (c *Conn) fault(code, op string, err error) bus.Fault {
	return bus.Fault{Code: code, Op: op, Message: "principal " + c.id + ": " + err.Error(), Err: err}
}

func (c *Conn) requestFault(subject string, err error) bus.Fault {
	switch {
	case errors.Is(err, nats.ErrNoResponders):
		return bus.Fault{Code: bus.FaultNoResponders, Op: subject,
			Message: "principal " + c.id + ": nothing is serving this subject", Err: err}
	case errors.Is(err, nats.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		return bus.Fault{Code: bus.FaultTimeout, Op: subject,
			Message: "principal " + c.id + ": no reply within the deadline", Err: err}
	case errors.Is(err, context.Canceled):
		return bus.Fault{Code: bus.FaultWithdrawn, Op: subject, Message: "principal " + c.id + ": request cancelled", Err: err}
	}
	if f := c.denied(subject); f != nil {
		var bf bus.Fault
		if errors.As(f, &bf) {
			bf.Err = err
			return bf
		}
	}
	return c.fault(bus.FaultUnavailable, subject, err)
}
