package natsconn

import (
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	nats "github.com/nats-io/nats.go"
)

// natsMsgID is the broker's own idempotency header. bus.WithMsgID sets
// bus.HeaderMsgID; the transport translates so a plugin never has to know the
// broker's spelling.
const natsMsgID = "Nats-Msg-Id"

// toNATSHeader writes bus headers onto the wire.
//
// The identity triple is stripped rather than forwarded: the substrate stamps
// bd-caller, bd-generation and bd-subject from the connection's own credential,
// and a client that could write them could act as someone else (R7). Everything
// else under bd- is metadata and travels as the caller set it.
func toNATSHeader(h bus.Headers) nats.Header {
	out := nats.Header{}
	for k, v := range bus.StripReserved(h) {
		out.Set(k, v)
	}
	return out
}

// fromNATSHeader reads wire headers back, lowercasing every key.
//
// NATS canonicalises header keys on the wire (bd-schema goes out and
// Bd-Schema comes back), so a contract that compared keys by case would break
// on the first real broker. bus.Headers is documented lowercase; this is where
// that becomes true.
func fromNATSHeader(h nats.Header) bus.Headers {
	if len(h) == 0 {
		return nil
	}
	out := make(bus.Headers, len(h))
	for k, vs := range h {
		if len(vs) == 0 {
			continue
		}
		out[strings.ToLower(k)] = vs[0]
	}
	return out
}

// inbound converts a delivered nats message into the bus message a handler
// sees, installing the responder so Respond answers the real reply subject.
func (c *Conn) inbound(m *nats.Msg) *bus.Msg {
	if m == nil {
		return nil
	}
	var respond func(bus.Headers, []byte) error
	if m.Reply != "" {
		reply := m.Reply
		respond = func(h bus.Headers, data []byte) error {
			out := nats.NewMsg(reply)
			out.Data = data
			out.Header = toNATSHeader(h)
			if err := c.nc.PublishMsg(out); err != nil {
				return c.fault(bus.FaultUnavailable, reply, err)
			}
			return nil
		}
	}
	return bus.NewMsg(bus.Subject(m.Subject), bus.Subject(m.Reply), fromNATSHeader(m.Header), m.Data, respond)
}
