package natsconn

import (
	"context"
	"sync"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	nats "github.com/nats-io/nats.go"
)

// subscription is a live bus.Subscription over a nats subscription.
//
// Its whole job beyond delegation is to carry a REASON. nats.go ends a
// subscription by unsubscribing it and reports a permissions violation or a
// slow consumer through a connection-wide callback, so by the time a plugin
// notices its handler stopped firing there is nothing left to ask. Err is
// where the fault the async callback routed here is kept.
type subscription struct {
	c       *Conn
	ns      *nats.Subscription
	subject string
	queue   string

	once sync.Once
	done chan struct{}
	mu   sync.RWMutex
	err  error
}

func (s *subscription) Cancel() {
	s.finish(nil)
}

func (s *subscription) Drain(ctx context.Context) error {
	if s.ns != nil {
		if err := s.ns.Drain(); err != nil {
			s.finish(s.c.fault(bus.FaultUnavailable, s.subject, err))
			return s.Err()
		}
	}
	select {
	case <-s.done:
	case <-ctx.Done():
		// Drain is bounded by the caller's deadline, and a deadline that
		// expires is not a clean drain: the queue was not delivered.
		s.finish(bus.Fault{Code: bus.FaultTimeout, Op: s.subject,
			Message: "principal " + s.c.id + ": drain did not finish within the deadline", Err: ctx.Err()})
		return s.Err()
	}
	return s.Err()
}

func (s *subscription) Done() <-chan struct{} { return s.done }

func (s *subscription) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// finish ends the subscription exactly once, recording why. A nil reason is a
// clean Cancel or Drain and leaves Err nil, which is the v1 contract.
func (s *subscription) finish(reason error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.err = reason
		s.mu.Unlock()
		if s.ns != nil {
			_ = s.ns.Unsubscribe()
		}
		s.c.forget(s)
		close(s.done)
	})
}
