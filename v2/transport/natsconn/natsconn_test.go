package natsconn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/internal/fakebroker"
	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/transport/natsconn"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

func dial(t *testing.T, b *fakebroker.Broker) *natsconn.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := natsconn.Dial(ctx, natsconn.Options{
		Socket:    b.Socket,
		CredsFile: fakebroker.WriteCreds(t),
		PluginID:  "files",
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func fault(t *testing.T, err error) bus.Fault {
	t.Helper()
	if err == nil {
		t.Fatal("expected a fault, got nil")
	}
	var f bus.Fault
	if !errors.As(err, &f) {
		t.Fatalf("error %v (%T) is not a bus.Fault", err, err)
	}
	return f
}

// TestRevokedCredentialFailsConnect is the acceptance criterion in one test:
// a credential the server refuses must fail the CONNECT, and the failure must
// arrive as FaultDenied naming the socket and the principal — not as a generic
// transport error an operator has to grep the broker's log to understand.
//
// Every refusal a real server produces for a withdrawn credential is covered,
// because they are different -ERR strings and nats.go maps them to different
// sentinels: an operator revoking an account sees one, an expired JWT another.
func TestRevokedCredentialFailsConnect(t *testing.T) {
	for _, tc := range []struct{ name, errText string }{
		{"revoked account", "Authorization Violation"},
		{"revoked credential", "Authentication Revoked"},
		{"expired credential", "Authentication Expired"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := fakebroker.Start(t)
			b.Refuse(tc.errText)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := natsconn.Dial(ctx, natsconn.Options{
				Socket:    b.Socket,
				CredsFile: fakebroker.WriteCreds(t),
				PluginID:  "files",
			})
			f := fault(t, err)
			if f.Code != bus.FaultDenied {
				t.Fatalf("code = %q, want %q (err %v)", f.Code, bus.FaultDenied, err)
			}
			if f.Op != b.Socket {
				t.Errorf("Op = %q, want the socket %q", f.Op, b.Socket)
			}
			if want := "principal files"; !contains(f.Message, want) {
				t.Errorf("Message %q does not name the principal", f.Message)
			}
		})
	}
}

// TestConnectFailureThatIsNotAuthIsNotDenied is the other half: a broker that
// is simply absent must NOT report FaultDenied. Mapping every connect failure
// to "denied" would make the grant system look broken every time the socket
// was missing.
func TestConnectFailureThatIsNotAuthIsNotDenied(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := natsconn.Dial(ctx, natsconn.Options{
		Socket:    "/nonexistent/bus.sock",
		CredsFile: fakebroker.WriteCreds(t),
		PluginID:  "files",
	})
	if f := fault(t, err); f.Code != bus.FaultUnavailable {
		t.Fatalf("code = %q, want %q (err %v)", f.Code, bus.FaultUnavailable, err)
	}
}

// TestDialRefusesWithoutCredentials: an unauthenticated bus connection is not a
// degraded mode, so the SDK refuses before it dials.
func TestDialRefusesWithoutCredentials(t *testing.T) {
	b := fakebroker.Start(t)
	ctx := context.Background()
	if f := fault(t, mustErr(natsconn.Dial(ctx, natsconn.Options{Socket: b.Socket, PluginID: "files"}))); f.Code != bus.FaultDenied {
		t.Fatalf("code = %q, want %q", f.Code, bus.FaultDenied)
	}
	if f := fault(t, mustErr(natsconn.Dial(ctx, natsconn.Options{CredsFile: "x", PluginID: "files"}))); f.Code != bus.FaultUnavailable {
		t.Fatalf("missing socket: code = %q, want %q", f.Code, bus.FaultUnavailable)
	}
}

// TestRequestReply proves the transport round trip, headers included: a
// request reaches the broker's responder and the reply's headers come back
// lowercased, which is the contract bus.Headers documents and NATS breaks.
func TestRequestReply(t *testing.T) {
	b := fakebroker.Start(t)
	b.Respond("cmd.files.v1.list", func(_ string, h map[string]string, data []byte) []byte {
		if h["bd-schema"] != "abc123" {
			t.Errorf("headers = %v, want bd-schema to survive egress", h)
		}
		return []byte(`{"ok":true}` + string(data))
	})
	c := dial(t, b)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reply, err := c.Request(ctx, "cmd.files.v1.list", []byte("!"), bus.WithSchema("abc123"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if got, want := string(reply.Data), `{"ok":true}!`; got != want {
		t.Fatalf("reply = %q, want %q", got, want)
	}
}

// TestRequestWithNothingServingTimesOut: the fake broker has no no-responders
// support, so an unserved subject exhausts the deadline. What matters is that
// the SDK reports it as FaultTimeout rather than as an opaque error.
func TestRequestWithNothingServingTimesOut(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := c.Request(ctx, "cmd.nobody.v1.home", nil)
	if f := fault(t, err); f.Code != bus.FaultTimeout {
		t.Fatalf("code = %q, want %q (err %v)", f.Code, bus.FaultTimeout, err)
	}
}

// TestSubscribeDelivers is the core pub/sub path, and it also pins the header
// lowercasing on ingress.
func TestSubscribeDelivers(t *testing.T) {
	b := fakebroker.Start(t)
	c := dial(t, b)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got := make(chan *bus.Msg, 1)
	sub, err := c.Subscribe(ctx, "event.files.v1.changed", func(_ context.Context, m *bus.Msg) { got <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Cancel()
	if err := c.Publish(ctx, "event.files.v1.changed", []byte("hello")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	select {
	case m := <-got:
		if string(m.Data) != "hello" {
			t.Fatalf("data = %q", m.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no delivery")
	}
}

// TestSubscriptionViolationEndsTheSubscription is the async-refusal half of the
// error handler: the broker refuses the subscription out of band, with a nil
// subscription attached, and the SDK has to route it to the right one.
//
// Without this the plugin's handler simply stops firing, with nothing anywhere
// to say why — which is the silent refusal v2 exists to remove.
func TestSubscriptionViolationEndsTheSubscription(t *testing.T) {
	b := fakebroker.Start(t)
	b.ViolateOn("event.secrets.v1.read")
	c := dial(t, b)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub, err := c.Subscribe(ctx, "event.secrets.v1.read", func(context.Context, *bus.Msg) {})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	select {
	case <-sub.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("the subscription is still live after a permissions violation")
	}
	f := fault(t, sub.Err())
	if f.Code != bus.FaultDenied {
		t.Fatalf("code = %q, want %q", f.Code, bus.FaultDenied)
	}
	if f.Op != "event.secrets.v1.read" {
		t.Errorf("Op = %q, want the refused subject", f.Op)
	}
}

// TestPublishViolationSurfacesOnTheNextCall pins the behaviour the plan calls
// for: a core publish is asynchronous, so the refusal cannot be returned by the
// call that caused it. It must be returned by the NEXT call for that subject,
// rather than being dropped.
func TestPublishViolationSurfacesOnTheNextCall(t *testing.T) {
	b := fakebroker.Start(t)
	b.ViolateOn("event.secrets.v1.written")
	c := dial(t, b)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The first publish is accepted locally: the server has not answered yet.
	if err := c.Publish(ctx, "event.secrets.v1.written", []byte("x")); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var err error
	for time.Now().Before(deadline) {
		if err = c.Publish(ctx, "event.secrets.v1.written", []byte("x")); err != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	f := fault(t, err)
	if f.Code != bus.FaultDenied {
		t.Fatalf("code = %q, want %q", f.Code, bus.FaultDenied)
	}
	// And a DIFFERENT subject is unaffected: the refusal is recorded against
	// the subject, not against the connection.
	if err := c.Publish(ctx, "event.files.v1.changed", []byte("x")); err != nil {
		t.Fatalf("an unrelated subject was refused too: %v", err)
	}
}

// TestCapabilitiesAreIntersected: the plugin is told what its own connection
// can do, never what the host substrate can do. A host that reports durable
// storage does not make this transport durable.
func TestCapabilitiesAreIntersected(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	c.Negotiated(bus.Capabilities{Durable: true, KV: true, Services: true, Trace: true, MaxPayload: 1 << 20})
	caps := c.Capabilities()
	if caps.Durable || caps.KV || caps.Trace {
		t.Errorf("caps = %+v; the transport implements none of these yet", caps)
	}
	if !caps.Services {
		t.Error("services must survive the intersection")
	}
	if caps.MaxPayload != 64<<10 {
		t.Errorf("MaxPayload = %d, want the narrower of the two", caps.MaxPayload)
	}

	// A second Negotiated is ignored: capabilities belong to the generation.
	c.Negotiated(bus.Capabilities{Durable: true})
	if c.Capabilities().Services != true {
		t.Error("a second Negotiated replaced the generation's capabilities")
	}
}

// TestUnsupportedSurfacesRefuseLoudly: the durable surfaces are not nil, and
// they do not pretend. A plugin that reaches one gets an attributed
// FaultUnsupported naming the capability it needed.
func TestUnsupportedSurfacesRefuseLoudly(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	ctx := context.Background()
	for name, err := range map[string]error{
		"streams.declare": c.Streams().Declare(ctx, bus.StreamSpec{Name: "X"}),
		"kv.declare":      c.KV().Declare(ctx, bus.BucketSpec{Name: "X"}),
		"objects.declare": c.Objects().Declare(ctx, bus.BucketSpec{Name: "X"}),
		"schedule.every":  c.Schedule().Every(ctx, "tick", time.Minute, "tick.files.v1.beat", nil),
	} {
		if f := fault(t, err); f.Code != bus.FaultUnsupported {
			t.Errorf("%s: code = %q, want %q", name, f.Code, bus.FaultUnsupported)
		}
	}
	// Cancel is the exception and it is deliberate: Stop calls it, and Stop
	// must be safe to call twice.
	if err := c.Schedule().Cancel(ctx, "tick"); err != nil {
		t.Errorf("schedule.cancel must not fail a clean teardown: %v", err)
	}
}

// TestPayloadCeilingIsRefusedBeforeItIsSent: the server enforces the per
// principal limit, but a refusal that arrives asynchronously is a refusal a
// publisher cannot act on. The ceiling is checked here too.
func TestPayloadCeilingIsRefusedBeforeItIsSent(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	c.Negotiated(bus.Capabilities{Services: true, MaxPayload: 64 << 10})
	err := c.Publish(context.Background(), "event.files.v1.changed", make([]byte, (64<<10)+1))
	if f := fault(t, err); f.Code != bus.FaultBudget {
		t.Fatalf("code = %q, want %q (err %v)", f.Code, bus.FaultBudget, err)
	}
}

// TestInboxPrefixIsPerPlugin: a reply to one plugin can never be delivered to
// another, because each connection's inbox is under its own id.
func TestInboxPrefixIsPerPlugin(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	if got, want := c.InboxPrefix(), "_INBOX.files"; got != want {
		t.Fatalf("InboxPrefix = %q, want %q", got, want)
	}
}

// TestCloseEndsLiveSubscriptions: a plugin waiting on Done must be released
// when the connection goes away, with a reason.
func TestCloseEndsLiveSubscriptions(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	sub, err := c.Subscribe(context.Background(), "event.files.v1.>", func(context.Context, *bus.Msg) {})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	_ = c.Close()
	select {
	case <-sub.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("subscription still live after Close")
	}
	if f := fault(t, sub.Err()); f.Code != bus.FaultWithdrawn {
		t.Fatalf("code = %q, want %q", f.Code, bus.FaultWithdrawn)
	}
}

// TestMalformedSubjectIsRefused: one grammar, checked before anything is sent.
func TestMalformedSubjectIsRefused(t *testing.T) {
	c := dial(t, fakebroker.Start(t))
	ctx := context.Background()
	if f := fault(t, c.Publish(ctx, "Event.Files", nil)); f.Code != bus.FaultDenied {
		t.Errorf("publish: code = %q", f.Code)
	}
	if _, err := c.Request(ctx, "event.files.*", nil); fault(t, err).Code != bus.FaultDenied {
		t.Error("a wildcard is not a subject and must be refused")
	}
}

func mustErr[T any](_ T, err error) error { return err }

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
