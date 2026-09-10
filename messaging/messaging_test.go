package messaging_test

import (
	"context"
	"testing"
	"time"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/messaging"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/bus"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/messaging"
)

// host routes Request into the registrar, so one round trip exercises the
// re-exported descriptors, the schema check under them and the wrappers.
type host struct {
	registrar *pluginsdk.Registrar
	subs      map[string][]func(bus.Envelope)
}

func newHost() *host {
	return &host{registrar: pluginsdk.NewRegistrar(), subs: map[string][]func(bus.Envelope){}}
}

func (h *host) Publish(env bus.Envelope) error {
	for _, fn := range h.subs[env.Type] {
		fn(env)
	}
	return nil
}

func (h *host) Subscribe(eventType string, fn func(bus.Envelope)) func() {
	h.subs[eventType] = append(h.subs[eventType], fn)
	return func() { delete(h.subs, eventType) }
}

func (h *host) Request(ctx context.Context, env bus.Envelope) (bus.Envelope, error) {
	return h.registrar.HandleCommand(ctx, env)
}

func (h *host) Logger() pluginsdk.Logger           { return nil }
func (h *host) StateDir(string) string             { return "" }
func (h *host) Every(time.Duration, func()) func() { return func() {} }
func (h *host) BumpContributions()                 {}

var _ pluginsdk.Host = (*host)(nil)

// The re-exported surface is the common contract itself, not a copy: the
// descriptors are the same values and a sequence above 2^53 survives the trip
// because it is carried as a decimal string.
func TestReexportedCommandRoundTripsThroughTheCommonContract(t *testing.T) {
	if messaging.SendMessage != common.SendMessage || messaging.SendMessage.Name() != common.CommandMessageSend {
		t.Fatal("descriptor is a copy, not the common SDK's")
	}
	h := newHost()
	messaging.Handle(h.registrar, messaging.SendMessage, func(_ context.Context, caller pluginsdk.Caller, req messaging.MessageSendRequest) (messaging.MessageSendResult, error) {
		if !caller.Autonomous() {
			t.Errorf("caller with no subject lease must read as autonomous: %+v", caller)
		}
		return messaging.MessageSendResult{
			Status:  messaging.OperationStatus{OK: true},
			Message: messaging.Message{MessageID: "m1", Body: req.Body, Sequence: 1 << 60},
		}, nil
	})
	out, err := messaging.Call(context.Background(), h, messaging.SendMessage, messaging.MessageSendRequest{Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Status.OK || out.Message.Body != "hello" || out.Message.Sequence != 1<<60 {
		t.Fatalf("round trip = %+v", out)
	}
}

func TestReexportedEventRoundTripsAndRefusesAnotherRevision(t *testing.T) {
	h := newHost()
	got := make(chan messaging.ChangedEvent, 2)
	sub, err := messaging.On(h, messaging.Changed, func(v messaging.ChangedEvent) { got <- v })
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()
	if err := messaging.Emit(h, messaging.Changed, messaging.ChangedEvent{Cursor: "c1", Sequence: 7}); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-got:
		if v.Cursor != "c1" || v.Sequence != 7 {
			t.Fatalf("event payload = %+v", v)
		}
	default:
		t.Fatal("no event delivered")
	}
	// A payload stamped with a different schema is dropped, not mis-decoded.
	_ = h.Publish(bus.Envelope{
		Type:    messaging.Changed.Name(),
		Headers: map[string]string{pluginsdk.HeaderSchema: "other"},
		Payload: []byte(`{"cursor":"stale"}`),
	})
	select {
	case v := <-got:
		t.Fatalf("delivered a mismatched schema: %+v", v)
	default:
	}
}
