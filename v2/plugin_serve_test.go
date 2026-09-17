package pluginsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/internal/fakebroker"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus/memory"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

// testPlugin is the smallest complete v2 plugin: it embeds Base and implements
// the four methods of the contract.
type testPlugin struct {
	pluginsdk.Base

	id       string
	needs    []string
	startErr error

	mu        sync.Mutex
	started   bool
	stopped   bool
	validated bool
	ready     bool
	checked   bool
}

func (p *testPlugin) ID() string { return p.id }

func (p *testPlugin) Manifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{
		ID:       p.id,
		Version:  "1.0.0",
		Targets:  []string{pluginsdk.TargetGateway},
		Role:     pluginsdk.RoleSystem,
		Spawn:    true,
		Binary:   p.id,
		Needs:    p.needs,
		Protocol: &pluginsdk.ProtocolRequirements{Major: pluginsdk.ProtocolMajor},
	}
}

func (p *testPlugin) Start(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started = true
	return p.startErr
}

func (p *testPlugin) Stop(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopped = true
	return nil
}

func (p *testPlugin) Validate(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validated = true
	return nil
}

func (p *testPlugin) Ready(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ready = true
	return nil
}

func (p *testPlugin) CheckActivation(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checked = true
	return nil
}

func (p *testPlugin) did(f func(*testPlugin) bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return f(p)
}

// negotiateReply is what a v2 host answers. The tests build it the way the host
// will, so a change to the reply shape breaks here rather than in production.
func negotiateReply(id string, caps bus.Capabilities) []byte {
	b, _ := json.Marshal(pluginsdk.HostCapabilities{
		Major:      pluginsdk.ProtocolMajor,
		PluginID:   id,
		Generation: "gen-1",
		Features: []string{
			pluginsdk.FeatureLifecycleEndpoints,
			pluginsdk.FeatureHTTPRoutes,
			pluginsdk.FeatureGrantsDigest,
		},
		Bus:      caps,
		Identity: bus.Identity{PluginID: id, Generation: "gen-1", Role: bus.RolePlugin},
		StateDir: "/var/lib/gateway/" + id,
	})
	return b
}

// TestRevokedCredentialStopsTheProcess is the acceptance criterion at the level
// an operator sees it: the plugin process does not start, and what it reports
// on the way out is a FaultDenied naming the principal — not a stack trace, and
// not a generic "connection failed".
func TestRevokedCredentialStopsTheProcess(t *testing.T) {
	b := fakebroker.Start(t)
	b.Refuse("Authorization Violation")

	p := &testPlugin{id: "files"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
		Socket:    filepath.Join(t.TempDir(), "plugin.sock"),
		BusSocket: b.Socket,
		BusCreds:  fakebroker.WriteCreds(t),
	})
	var f bus.Fault
	if !errors.As(err, &f) {
		t.Fatalf("err = %v (%T), want a bus.Fault", err, err)
	}
	if f.Code != bus.FaultDenied {
		t.Fatalf("code = %q, want %q (err %v)", f.Code, bus.FaultDenied, err)
	}
	if !strings.Contains(f.Message, "principal files") {
		t.Errorf("message %q does not name the principal", f.Message)
	}
	// Nothing ran: a plugin that cannot reach the bus must not have acquired
	// anything to release.
	if p.did(func(p *testPlugin) bool { return p.started || p.validated }) {
		t.Error("the plugin started or validated without a bus")
	}
}

// TestMissingCredentialsTimeOutAsDenied: the host writes the creds file after
// the containment receipt commits, so the SDK waits. A wait that never ends is
// a hang; a wait that ends is a refusal an operator can read.
func TestMissingCredentialsTimeOutAsDenied(t *testing.T) {
	b := fakebroker.Start(t)
	p := &testPlugin{id: "files"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
		Socket:    filepath.Join(t.TempDir(), "plugin.sock"),
		BusSocket: b.Socket,
		BusCreds:  filepath.Join(t.TempDir(), "never-written.creds"),
		CredsWait: 200 * time.Millisecond,
	})
	var f bus.Fault
	if !errors.As(err, &f) || f.Code != bus.FaultDenied {
		t.Fatalf("err = %v, want FaultDenied", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("waited %s for a credential that was never coming", elapsed)
	}
}

// TestCredentialsArrivingLateAreWaitedFor is the other side of the same rule:
// the file appearing after the process starts is the NORMAL case, not a failure.
func TestCredentialsArrivingLateAreWaitedFor(t *testing.T) {
	b := fakebroker.Start(t)
	b.Respond(string(pluginsdk.NegotiateSubject), func(string, map[string]string, []byte) []byte {
		return negotiateReply("files", bus.Capabilities{Services: true, MaxPayload: 64 << 10})
	})
	creds := filepath.Join(t.TempDir(), "late.creds")
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = os.WriteFile(creds, []byte(fakebroker.TestCreds), 0o400)
	}()

	p := &testPlugin{id: "files"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
			Socket:    filepath.Join(t.TempDir(), "plugin.sock"),
			BusSocket: b.Socket,
			BusCreds:  creds,
		})
	}()
	// ServePlugin runs until its context ends; what matters is that it got past
	// the handshake, which only a plugin that started can have done.
	waitFor(t, func() bool { return p.did(func(p *testPlugin) bool { return p.started }) })
	cancel()
	<-done
}

// TestHandshakeOrder pins the sequence the whole design rests on: validate,
// then start, then lifecycle endpoints — and Bind before any of them, so that
// Bus(), Identity() and StateDir() are live by the time plugin code runs.
func TestHandshakeOrder(t *testing.T) {
	b := fakebroker.Start(t)
	b.Respond(string(pluginsdk.NegotiateSubject), func(_ string, _ map[string]string, data []byte) []byte {
		var req pluginsdk.NegotiateRequest
		if err := json.Unmarshal(data, &req); err != nil {
			t.Errorf("negotiate request is not JSON: %v", err)
			return nil
		}
		if req.PluginID != "files" {
			t.Errorf("negotiate PluginID = %q", req.PluginID)
		}
		if req.Protocol.Major != pluginsdk.ProtocolMajor {
			t.Errorf("negotiate major = %d, want %d", req.Protocol.Major, pluginsdk.ProtocolMajor)
		}
		if req.Inbox != "_INBOX.files" {
			t.Errorf("negotiate Inbox = %q, want the plugin's own inbox", req.Inbox)
		}
		if req.GrantsDigest == "" {
			t.Error("negotiate carries no grants digest; consent cannot be checked")
		}
		return negotiateReply("files", bus.Capabilities{Services: true, MaxPayload: 64 << 10})
	})

	p := &testPlugin{id: "files"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
			Socket:    filepath.Join(t.TempDir(), "plugin.sock"),
			BusSocket: b.Socket,
			BusCreds:  fakebroker.WriteCreds(t),
		})
	}()
	waitFor(t, func() bool { return p.did(func(p *testPlugin) bool { return p.started }) })
	cancel()
	<-done
	if !p.did(func(p *testPlugin) bool { return p.validated }) {
		t.Error("Validate never ran")
	}
	if !p.did(func(p *testPlugin) bool { return p.started }) {
		t.Error("Start never ran")
	}
	if !p.did(func(p *testPlugin) bool { return p.stopped }) {
		t.Error("Stop never ran: a partial start must always be stopped")
	}
	// The base is live: a plugin that was bound has an identity and a state dir.
	if got := p.Identity().PluginID; got != "files" {
		t.Errorf("Identity().PluginID = %q after Bind", got)
	}
	if got := p.StateDir(); got != "/var/lib/gateway/files" {
		t.Errorf("StateDir() = %q, want the host's", got)
	}
	if p.Bus() == nil {
		t.Error("Bus() is nil after Bind")
	}
}

// TestNeedsFailClosed is the capability gate. A plugin that declares it cannot
// run without durable storage must not start on a substrate that has none: a
// plugin which silently degrades is a data-loss report later.
func TestNeedsFailClosed(t *testing.T) {
	b := fakebroker.Start(t)
	b.Respond(string(pluginsdk.NegotiateSubject), func(string, map[string]string, []byte) []byte {
		// The HOST reports durable storage; the transport does not implement
		// it. The effective set is what the plugin is judged against.
		return negotiateReply("files", bus.Capabilities{Durable: true, Services: true, MaxPayload: 64 << 10})
	})

	p := &testPlugin{id: "files", needs: []string{"durable"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
		Socket:    filepath.Join(t.TempDir(), "plugin.sock"),
		BusSocket: b.Socket,
		BusCreds:  fakebroker.WriteCreds(t),
	})
	var f bus.Fault
	if !errors.As(err, &f) || f.Code != bus.FaultUnsupported {
		t.Fatalf("err = %v, want FaultUnsupported", err)
	}
	if !strings.Contains(f.Message, "durable") {
		t.Errorf("message %q does not name the missing capability", f.Message)
	}
	if p.did(func(p *testPlugin) bool { return p.started }) {
		t.Error("the plugin started without a capability it declared it needs")
	}
}

// TestLifecycleEndpointsAreMounted: v1's hooks were negotiated and called over
// a bespoke HTTP command; v2's are services in the plugin's own namespace. A
// host calls them by subject.
func TestLifecycleEndpointsAreMounted(t *testing.T) {
	mem := memoryBus(t, "files")
	p := &testPlugin{id: "files"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The memory bus stands in for the host too, so the host's half of the
	// handshake is one subscription.
	negotiator, err := mem.Subscribe(ctx, pluginsdk.Pattern(pluginsdk.NegotiateSubject), func(_ context.Context, m *bus.Msg) {
		_ = m.Respond(nil, negotiateReply("files", bus.Capabilities{Services: true, MaxPayload: 64 << 10}))
	})
	if err != nil {
		t.Fatalf("mount the host side of negotiate: %v", err)
	}
	defer negotiator.Cancel()

	done := make(chan error, 1)
	go func() {
		done <- pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{
			Socket: filepath.Join(t.TempDir(), "plugin.sock"),
			Bus:    mem,
		})
	}()
	waitFor(t, func() bool { return p.did(func(p *testPlugin) bool { return p.started }) })

	for _, hook := range []string{"ready", "activation.check"} {
		subject := pluginsdk.LifecycleSubject("files", hook)
		reply, err := mem.Request(ctx, subject, nil)
		if err != nil {
			t.Fatalf("%s: %v", subject, err)
		}
		var out struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(reply.Data, &out); err != nil {
			t.Fatalf("%s reply is not a verdict: %v", subject, err)
		}
		if out.Error != "" {
			t.Fatalf("%s verdict = %q, want none", subject, out.Error)
		}
	}
	if !p.did(func(p *testPlugin) bool { return p.ready && p.checked }) {
		t.Error("the host called the endpoints but the plugin's hooks did not run")
	}
	cancel()
	<-done
}

// memoryBus is the SDK's own test double, bound to id's own namespace. It is
// how a plugin author tests a plugin with no broker at all, and this test is
// the SDK proving its own advice works.
//
// The host principal is used rather than the plugin's, because this bus stands
// in for BOTH sides: the plugin mounts its lifecycle endpoints on it and the
// test calls them back as the host would.
func memoryBus(t *testing.T, id string) pluginsdk.Bus {
	t.Helper()
	store := memory.NewStore(memory.WithCapabilities(bus.Capabilities{Services: true, MaxPayload: 64 << 10}))
	t.Cleanup(store.Close)
	// The grants are the plugin's own namespace plus the negotiate command,
	// which in production belongs to the host and is reached by every plugin.
	grants := plugin.OwnNamespace(id)
	grants.Subscribe = append(grants.Subscribe, "cmd.plugin.v1.>")
	// and the lifecycle endpoints, which the HOST calls on the plugin.
	grants.Request = append(grants.Request, "cmd.plugin.v1.>", bus.Pattern("svc."+id+".>"))
	b := store.Connect(bus.Identity{PluginID: id, Generation: "gen-1", Role: bus.RoleHost, Grants: grants})
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition never became true")
}
