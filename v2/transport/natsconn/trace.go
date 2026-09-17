package natsconn

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

// tracer is bus.Trace: correlation propagation, which works on any substrate
// because it is only headers and a context value. Broker-side message tracing
// is a different thing and is not built, so Capabilities.Trace stays false.
type tracer struct{}

type corrKey struct{}

func (tracer) Correlation(ctx context.Context) string {
	if v, ok := ctx.Value(corrKey{}).(string); ok && v != "" {
		return v
	}
	return randomToken()
}

func (tracer) WithCorrelation(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, corrKey{}, id)
}

func (t tracer) Inject(ctx context.Context, h bus.Headers) bus.Headers {
	out := h.Clone()
	if out == nil {
		out = bus.Headers{}
	}
	if out.Get(bus.HeaderCorrelation) == "" {
		out[bus.HeaderCorrelation] = t.Correlation(ctx)
	}
	if tp, ok := ctx.Value(traceparentKey{}).(string); ok && tp != "" && out.Get(bus.HeaderTraceparent) == "" {
		out[bus.HeaderTraceparent] = tp
	}
	return out
}

type traceparentKey struct{}

func (t tracer) Extract(ctx context.Context, h bus.Headers) context.Context {
	if id := h.Get(bus.HeaderCorrelation); id != "" {
		ctx = t.WithCorrelation(ctx, id)
	}
	if tp := h.Get(bus.HeaderTraceparent); tp != "" {
		ctx = context.WithValue(ctx, traceparentKey{}, tp)
	}
	return ctx
}

func randomToken() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail in practice; a token that collides is a
		// muddled trace, never a security decision, so degrade rather than panic.
		return "0000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}
