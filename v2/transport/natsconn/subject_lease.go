package natsconn

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// HeaderSubjectLease carries the host-minted subject lease across the private
// host-to-plugin transport, the v2 (real NATS wire) equivalent of the v1 SDK's
// header of the same name and shape. The host must remove every
// client-supplied copy before stamping this header on a request it proxies to
// a plugin's own HTTP surface; a plugin echoes it back verbatim on any Conn
// Request made while handling that inbound call.
//
// It deliberately does not live under the bus package's "bd-" prefix: that
// namespace is the identity triple the substrate itself stamps from a
// connection's credential and strips from anything client-supplied
// (bus.StripReserved) -- a lease forwarded this way is not an identity claim,
// it is an opaque, host-minted, unguessable id a plugin can only ever echo,
// never manufacture, because it is never told what audience or role the id
// resolves to. The host resolves and checks it, exactly as it does for a v1
// spawned plugin's /request RPC.
const HeaderSubjectLease = "X-Bytedesk-Subject-Lease"

// ErrInvalidSubjectLease means the private transport contained a subject
// header, but it was malformed or ambiguous. This is different from a missing
// header, which preserves the legacy autonomous-call behavior.
var ErrInvalidSubjectLease = errors.New("invalid host subject lease")

type subjectLeaseContextKey struct{}

type subjectLeaseTransport struct {
	present bool
	values  []string
	lease   string
	err     error
}

// ContextForRequest returns the request context with the private subject
// transport parsed and retained. A malformed or duplicate present value is
// retained as an error so a subsequent Conn.Request cannot silently become an
// autonomous call. Mirrors the v1 SDK's http_subject.go exactly -- the shape
// of the problem (an opaque lease riding an HTTP header into a plugin's own
// request context) is identical; only the outbound transport differs.
func ContextForRequest(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	transport := subjectLeaseTransport{}
	for key, values := range r.Header {
		if !strings.EqualFold(key, HeaderSubjectLease) {
			continue
		}
		transport.present = true
		transport.values = append(transport.values, values...)
	}
	if !transport.present {
		return r.Context()
	}
	if len(transport.values) != 1 || !canonicalSubjectLease(transport.values[0]) {
		transport.err = ErrInvalidSubjectLease
	} else {
		transport.lease = transport.values[0]
	}
	return context.WithValue(r.Context(), subjectLeaseContextKey{}, transport)
}

// SubjectLeaseFromContext returns the canonical opaque lease carried by an
// inbound request, or "" when the request carried none. err is non-nil only
// when a subject header was present and malformed -- a caller must treat that
// as a hard denial, never as an absent lease, so a garbled transport can never
// silently downgrade to autonomous authority.
func SubjectLeaseFromContext(ctx context.Context) (string, error) {
	transport, ok := subjectLeaseTransportFromContext(ctx)
	if !ok {
		return "", nil
	}
	return transport.lease, transport.err
}

func subjectLeaseTransportFromContext(ctx context.Context) (subjectLeaseTransport, bool) {
	if ctx == nil {
		return subjectLeaseTransport{}, false
	}
	transport, ok := ctx.Value(subjectLeaseContextKey{}).(subjectLeaseTransport)
	return transport, ok
}

// canonicalSubjectLease matches exactly what hostSubjectLeases.mint produces
// on the gateway: 32 random bytes, lowercase hex. Anything else -- wrong
// length, uppercase, non-hex -- is rejected before it ever reaches a lookup,
// so a malformed value fails fast and by shape, not by a failed host resolve.
func canonicalSubjectLease(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
