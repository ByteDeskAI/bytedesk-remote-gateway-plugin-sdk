package pluginsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/transport/natsconn"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

// Env vars the gateway host sets when spawning a v2 plugin. The bind list is
// unchanged from v1: both new keys name paths inside the plugin's own run dir,
// which is already mounted.
const (
	// EnvSocket is where the plugin serves its HTTP routes. Unchanged from v1.
	EnvSocket = "GATEWAY_PLUGIN_SOCKET"
	// EnvID is the identity the host assigned. Unchanged from v1.
	EnvID = "GATEWAY_PLUGIN_ID"
	// EnvBusSocket is the unix socket the host splices to the broker.
	EnvBusSocket = "GATEWAY_BUS_SOCKET"
	// EnvBusCreds is the .creds file for this generation, 0400, written AFTER
	// the containment receipt commits and unlinked on revoke.
	EnvBusCreds = "GATEWAY_BUS_CREDS"
)

// CredsWait is how long ServePlugin waits for the credential file to appear.
//
// It is a wait rather than a requirement because the host writes the file only
// after the containment receipt commits, which can land after the process is
// already running. Five seconds is long enough for that ordering and short
// enough that a plugin whose credential will never arrive fails while an
// operator is still watching the start.
const CredsWait = 5 * time.Second

// PluginConfig is how a spawned plugin is started. Every field has a working
// default read from the environment; the fields exist so a test can construct
// a plugin without a gateway.
type PluginConfig struct {
	// Socket overrides GATEWAY_PLUGIN_SOCKET.
	Socket string
	// BusSocket and BusCreds override GATEWAY_BUS_SOCKET and GATEWAY_BUS_CREDS.
	BusSocket string
	BusCreds  string
	// Bus, when set, is used instead of dialling. This is how a conformance
	// test runs the whole handshake against bus/memory with no broker.
	Bus Bus
	// Logger and Profiler override the defaults.
	Logger   Logger
	Profiler Profiler
	// CredsWait overrides CredsWait.
	CredsWait time.Duration
}

// ServePlugin runs a spawned plugin's entire lifecycle.
//
// The order is load bearing and is the same order the host reasons about:
//
//	identity → manifest → credentials → dial → negotiate → check → bind →
//	validate → start → lifecycle endpoints → HTTP
//
// Lifecycle endpoints mount after Start on purpose, so a host can never call
// Ready, CheckActivation or HealthSections before the plugin's own Start has
// run; that ordering means mounting them is the one step after Start that can
// still refuse (the substrate can deny or reject the subject). Nothing else
// that can refuse happens after Start, and nothing needs the bus before Bind.
// A failure at any step, including that one, stops the steps that ran before
// it: a partial start is always stopped, which is the one v1 behaviour worth
// carrying across unchanged.
func ServePlugin(ctx context.Context, p Plugin, cfg PluginConfig) (result error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p == nil {
		return fmt.Errorf("plugin required")
	}
	manifest := p.Manifest()
	if p.ID() == "" || p.ID() != manifest.ID {
		return fmt.Errorf("plugin and manifest identity must match")
	}
	if id := os.Getenv(EnvID); id != "" && id != p.ID() {
		return fmt.Errorf("plugin identity differs from host assignment")
	}
	if err := plugin.ValidateDiscover(manifest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	b := cfg.Bus
	effective := bus.Capabilities{}
	var conn *natsconn.Conn
	if b == nil {
		var err error
		conn, err = dialBus(ctx, p.ID(), cfg)
		if err != nil {
			return err
		}
		defer conn.Close()
		b = conn
	}

	have, err := negotiate(ctx, b, p.ID(), manifest, conn)
	if err != nil {
		return err
	}
	effective = have.Bus
	if conn != nil {
		conn.Negotiated(have.Bus)
		effective = conn.Capabilities()
	}
	if err := CheckProtocol(have, declaredProtocol(manifest), manifest.Needs, effective); err != nil {
		return err
	}
	if err := checkIdentity(have, p.ID()); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := cfg.Logger
	if logger == nil {
		logger = StderrLogger(p.ID())
	}
	profiler := cfg.Profiler
	if profiler == nil {
		// ponytail: a spawned v2 plugin has no host-driven profiling switch
		// yet — the subject that carries it is the gateway's to define when it
		// adopts v2 (TM-365/TM-368). Until then the switch is off and local,
		// which is what NopProfiler is. Pass cfg.Profiler to drive it yourself.
		profiler = plugin.NopProfiler()
	}
	identity := have.Identity
	if identity.PluginID == "" {
		identity.PluginID = p.ID()
	}
	if identity.Generation == "" {
		identity.Generation = have.Generation
	}
	if err := Bind(p, Binding{
		Bus:      b,
		Logger:   logger,
		Profiler: profiler,
		StateDir: have.StateDir,
		Identity: identity,
		Caps:     effective,
	}); err != nil {
		return err
	}

	// Validate runs after Bind because a precondition check needs the bus, the
	// state dir and the identity, and all three are live by now. It runs before
	// Start because a plugin that cannot run must not acquire anything first.
	if validator, ok := p.(Validator); ok {
		if err := validator.Validate(ctx); err != nil {
			return err
		}
	}
	defer func() {
		cancel()
		stopCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		result = errors.Join(result, p.Stop(stopCtx))
	}()
	if err := p.Start(ctx); err != nil {
		return err
	}

	svc, err := mountLifecycle(ctx, b, p)
	if err != nil {
		return err
	}
	if svc != nil {
		defer func() { _ = svc.Stop(context.Background()) }()
	}

	handler := http.Handler(nil)
	if httpPlugin, ok := p.(HTTPPlugin); ok {
		handler = httpPlugin.Handler()
	}
	return ServeContext(ctx, Config{ID: p.ID(), Socket: cfg.Socket, Handler: handler})
}

// dialBus waits for the credential, then connects.
func dialBus(ctx context.Context, id string, cfg PluginConfig) (*natsconn.Conn, error) {
	socket := firstNonEmpty(cfg.BusSocket, os.Getenv(EnvBusSocket))
	if socket == "" {
		return nil, Fault{Code: FaultUnavailable, Op: EnvBusSocket, Message: EnvBusSocket + " required"}
	}
	creds := firstNonEmpty(cfg.BusCreds, os.Getenv(EnvBusCreds))
	if creds == "" {
		return nil, Fault{Code: FaultDenied, Op: EnvBusCreds, Message: "principal " + id + ": " + EnvBusCreds + " required"}
	}
	wait := cfg.CredsWait
	if wait <= 0 {
		wait = CredsWait
	}
	if err := waitForCreds(ctx, creds, wait); err != nil {
		return nil, err
	}
	return natsconn.Dial(ctx, natsconn.Options{Socket: socket, CredsFile: creds, PluginID: id})
}

// waitForCreds polls until the credential file exists and is non-empty.
//
// Non-empty matters: the host creates the file and writes it, and a plugin that
// raced in between would read a zero-byte creds file and report a parse error
// instead of waiting the extra millisecond it needed.
func waitForCreds(ctx context.Context, path string, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return Fault{Code: FaultDenied, Op: path,
				Message: fmt.Sprintf("bus credentials did not arrive within %s", wait)}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// negotiate asks the host what this generation is.
func negotiate(ctx context.Context, b Bus, id string, m Manifest, conn *natsconn.Conn) (HostCapabilities, error) {
	req := NegotiateRequest{
		PluginID:     id,
		Protocol:     declaredProtocol(m),
		Needs:        m.Needs,
		GrantsDigest: GrantsDigest(m),
		SDK:          Version(),
	}
	if conn != nil {
		req.Inbox = conn.InboxPrefix()
	}
	body, err := json.Marshal(req)
	if err != nil {
		return HostCapabilities{}, err
	}
	reply, err := b.Request(ctx, NegotiateSubject, body)
	if err != nil {
		return HostCapabilities{}, err
	}
	if code := reply.Headers.Get(HeaderFault); code != "" {
		return HostCapabilities{}, Fault{Code: code, Op: string(NegotiateSubject), Message: string(reply.Data)}
	}
	var have HostCapabilities
	if err := json.Unmarshal(reply.Data, &have); err != nil {
		return HostCapabilities{}, Fault{Code: FaultSchema, Op: string(NegotiateSubject),
			Message: "host reply is not a negotiation: " + err.Error(), Err: err}
	}
	return have, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
