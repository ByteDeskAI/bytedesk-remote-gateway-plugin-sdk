package natsconn

import (
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/nats-io/nats.go"
)

// HeaderHostCallToken authenticates a host call's workload, not its human actor.
// It grants no operation and must be stripped before dispatching to providers.
const HeaderHostCallToken = "X-Bytedesk-Host-Call"

// BindHostCallToken installs the generation token returned by negotiation. It
// cannot be replaced while this connection is alive, including during reconnect.
func (c *Conn) BindHostCallToken(token string) error {
	if !canonicalSubjectLease(token) {
		return bus.Fault{Code: bus.FaultSchema, Op: "workload-bind", Message: "invalid host workload token"}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hostCallToken != "" && c.hostCallToken != token {
		return bus.Fault{Code: bus.FaultDenied, Op: "workload-bind", Message: "host workload identity is already bound"}
	}
	c.hostCallToken = token
	return nil
}

func (c *Conn) bindHostCallHeader(headers nats.Header, subject string) {
	// Remove caller-supplied spellings, even on non-host requests. Otherwise a
	// reusable options object could accidentally disclose the token to a peer.
	for key := range headers {
		if strings.EqualFold(key, HeaderHostCallToken) {
			delete(headers, key)
		}
	}
	if !strings.HasPrefix(subject, "cmd.gateway.") {
		return
	}
	c.mu.RLock()
	token := c.hostCallToken
	c.mu.RUnlock()
	if token != "" {
		headers.Set(HeaderHostCallToken, token)
	}
}
