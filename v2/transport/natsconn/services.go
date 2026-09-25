package natsconn

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

// services is bus.Services over core NATS.
//
// It speaks the NATS micro discovery protocol by hand ($SRV.PING / $SRV.INFO /
// $SRV.STATS) rather than importing nats.go's micro package, because micro's
// handler type is a nats type and wrapping it costs more than the three
// request/reply shapes below. Endpoints are ordinary queue subscriptions, which
// is exactly what micro's are.
type services struct{ c *Conn }

const (
	srvPing  = "$SRV.PING"
	srvInfo  = "$SRV.INFO"
	srvStats = "$SRV.STATS"
	// discoverWindow bounds a scatter-gather. Discovery is best effort by
	// nature: there is no roster to compare against, so the answer is whoever
	// replied in time.
	discoverWindow = 500 * time.Millisecond
)

func (s services) Serve(ctx context.Context, spec bus.ServiceSpec) (bus.Service, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return nil, bus.Fault{Code: bus.FaultUnhandled, Op: "services.serve", Message: "service name required"}
	}
	if len(spec.Endpoints) == 0 {
		return nil, bus.Fault{Code: bus.FaultUnhandled, Op: "services.serve", Message: "service " + spec.Name + " declares no endpoints"}
	}
	svc := &service{
		c:       s.c,
		spec:    spec,
		id:      s.c.id + "-" + spec.Name,
		started: time.Now(),
		done:    make(chan struct{}),
	}
	for _, ep := range spec.Endpoints {
		if err := svc.mount(ctx, ep); err != nil {
			_ = svc.Stop(ctx)
			return nil, err
		}
	}
	if err := svc.mountDiscovery(ctx); err != nil {
		_ = svc.Stop(ctx)
		return nil, err
	}
	return svc, nil
}

// Call invokes an endpoint. A service error reply carries bd-fault, and that
// comes back as a Fault rather than as a payload the caller has to inspect.
func (s services) Call(ctx context.Context, subject bus.Subject, data []byte, opts ...bus.ReqOpt) (*bus.Msg, error) {
	m, err := s.c.Request(ctx, subject, data, opts...)
	if err != nil {
		return nil, err
	}
	if code := m.Headers.Get(bus.HeaderFault); code != "" {
		return nil, bus.Fault{Code: code, Op: string(subject), Message: string(m.Data)}
	}
	return m, nil
}

func (s services) Discover(ctx context.Context, name string) ([]bus.ServiceInfo, error) {
	subject := srvInfo
	if name != "" {
		subject += "." + name
	}
	var out []bus.ServiceInfo
	err := s.gather(ctx, bus.Subject(subject), func(b []byte) {
		var info bus.ServiceInfo
		if json.Unmarshal(b, &info) == nil && info.Name != "" {
			out = append(out, info)
		}
	})
	return out, err
}

func (s services) Stats(ctx context.Context, name string) ([]bus.ServiceStats, error) {
	subject := srvStats
	if name != "" {
		subject += "." + name
	}
	var out []bus.ServiceStats
	err := s.gather(ctx, bus.Subject(subject), func(b []byte) {
		var st bus.ServiceStats
		if json.Unmarshal(b, &st) == nil && st.Name != "" {
			out = append(out, st)
		}
	})
	return out, err
}

// gather is the scatter-gather every discovery call is: subscribe an inbox,
// publish the probe to it, collect until the window closes.
func (s services) gather(ctx context.Context, subject bus.Subject, collect func([]byte)) error {
	inbox := bus.Pattern(s.c.inbox + ".discover." + randomToken())
	var mu sync.Mutex
	sub, err := s.c.Subscribe(ctx, inbox, func(_ context.Context, m *bus.Msg) {
		mu.Lock()
		defer mu.Unlock()
		collect(m.Data)
	})
	if err != nil {
		return err
	}
	defer sub.Cancel()
	if err := s.c.publishTo(string(subject), string(inbox), nil); err != nil {
		return err
	}
	window := discoverWindow
	if dl, ok := ctx.Deadline(); ok {
		if left := time.Until(dl); left < window {
			window = left
		}
	}
	select {
	case <-time.After(window):
	case <-ctx.Done():
	}
	mu.Lock()
	defer mu.Unlock()
	return nil
}

// service is one mounted bus.Service.
type service struct {
	c       *Conn
	spec    bus.ServiceSpec
	id      string
	started time.Time

	mu    sync.Mutex
	subs  []bus.Subscription
	reqs  atomic.Uint64
	errs  atomic.Uint64
	nanos atomic.Int64

	once sync.Once
	done chan struct{}
	err  error
}

func (s *service) mount(ctx context.Context, ep bus.EndpointSpec) error {
	if ep.Handler == nil {
		return bus.Fault{Code: bus.FaultUnhandled, Op: string(ep.Subject), Message: "endpoint " + ep.Name + " has no handler"}
	}
	group := ep.QueueGroup
	if group == "" {
		group = s.spec.QueueGroup
	}
	if group == "" {
		// Two generations of the same plugin must not both answer one call.
		// The service name is the default group for exactly that reason.
		group = s.spec.Name
	}
	h := ep.Handler
	wrapped := func(ctx context.Context, m *bus.Msg) {
		start := time.Now()
		s.reqs.Add(1)
		defer func() { s.nanos.Add(int64(time.Since(start))) }()
		h(ctx, m)
	}
	sub, err := s.c.Subscribe(ctx, bus.Pattern(ep.Subject), wrapped, bus.QueueGroup(group))
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()
	return nil
}

// mountDiscovery answers $SRV.PING/INFO/STATS so the host can find this service
// the same way it finds any other. Without it a service is reachable but
// invisible, and the host resolves extension points by discovery.
func (s *service) mountDiscovery(ctx context.Context) error {
	for _, probe := range []struct {
		root string
		body func() any
	}{
		{srvPing, func() any {
			return map[string]string{"type": "io.nats.micro.v1.ping_response", "name": s.spec.Name, "id": s.id, "version": s.spec.Version}
		}},
		{srvInfo, func() any { return s.Info() }},
		{srvStats, func() any { return s.Stats() }},
	} {
		body := probe.body
		for _, pattern := range []bus.Pattern{
			bus.Pattern(probe.root),
			bus.Pattern(probe.root + "." + s.spec.Name),
			bus.Pattern(probe.root + "." + s.spec.Name + "." + s.id),
		} {
			sub, err := s.c.Subscribe(ctx, pattern, func(_ context.Context, m *bus.Msg) {
				b, err := json.Marshal(body())
				if err != nil {
					return
				}
				_ = m.Respond(nil, b)
			})
			if err != nil {
				return err
			}
			s.mu.Lock()
			s.subs = append(s.subs, sub)
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *service) Stop(ctx context.Context) error {
	s.once.Do(func() {
		s.mu.Lock()
		subs := s.subs
		s.subs = nil
		s.mu.Unlock()
		for _, sub := range subs {
			sub.Cancel()
		}
		close(s.done)
	})
	return nil
}

func (s *service) Info() bus.ServiceInfo {
	eps := make([]bus.EndpointInfo, 0, len(s.spec.Endpoints))
	for _, ep := range s.spec.Endpoints {
		group := ep.QueueGroup
		if group == "" {
			group = s.spec.QueueGroup
		}
		eps = append(eps, bus.EndpointInfo{Name: ep.Name, Subject: ep.Subject, QueueGroup: group, Point: ep.Point})
	}
	return bus.ServiceInfo{
		ID:        s.id,
		Name:      s.spec.Name,
		Version:   s.spec.Version,
		Endpoints: eps,
		Metadata:  s.spec.Metadata,
	}
}

func (s *service) Stats() bus.ServiceStats {
	return bus.ServiceStats{
		ID:         s.id,
		Name:       s.spec.Name,
		Requests:   s.reqs.Load(),
		Errors:     s.errs.Load(),
		Processing: time.Duration(s.nanos.Load()),
		Started:    s.started,
	}
}

func (s *service) Done() <-chan struct{} { return s.done }
func (s *service) Err() error            { return s.err }
