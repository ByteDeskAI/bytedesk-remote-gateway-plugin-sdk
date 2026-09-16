package pluginsdk

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// HeaderSubjectLease carries the host-minted subject lease across the private
// host-to-plugin HTTP transport. The host must remove every client-supplied
// copy before stamping this header.
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
// retained as an error so a subsequent Host.Request cannot silently become an
// autonomous call.
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
// HTTP plugin request. A missing lease returns "", nil for compatibility with
// older hosts. Any present non-canonical or duplicate value returns
// ErrInvalidSubjectLease and must be refused by subject-scoped consumers.
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

func canonicalSubjectLease(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func withSubjectLeaseContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(ContextForRequest(r)))
	})
}
