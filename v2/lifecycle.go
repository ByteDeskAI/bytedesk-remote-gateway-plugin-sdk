package pluginsdk

import (
	"context"
	"encoding/json"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

// Lifecycle hook endpoints. In v1 these were hooks the host acknowledged
// during negotiation and then called over a bespoke HTTP command on the plugin
// socket. In v2 they are ordinary service endpoints in the plugin's own
// namespace, discoverable like anything else, and the negotiation no longer
// carries a hook list at all.
const (
	LifecycleService = "lifecycle"

	lifecycleActivationCheck = "activation.check"
	lifecycleReady           = "ready"
	lifecycleHealth          = "health"
)

// LifecycleSubject is where a host calls one of this plugin's lifecycle hooks.
func LifecycleSubject(id, hook string) Subject {
	return Subject("svc." + id + "." + LifecycleService + ".v1." + hook)
}

// lifecycleReply is what every hook answers.
//
// A hook VERDICT is a successful reply with a non-empty error: "I ran, and the
// answer is no". A transport failure is not a verdict, and conflating the two
// would let a broker hiccup read as a plugin refusing to activate.
type lifecycleReply struct {
	Error  string `json:"error,omitempty"`
	Result any    `json:"result,omitempty"`
}

// mountLifecycle publishes the hooks this plugin implements as endpoints.
//
// It returns nil when the plugin implements none, because mounting an empty
// service would advertise a lifecycle surface with nothing behind it.
func mountLifecycle(ctx context.Context, b Bus, p Plugin) (Service, error) {
	id := p.ID()
	// The registrar is the local preflight: it refuses an endpoint outside the
	// serves list here, with the endpoint still in hand, rather than letting
	// the substrate refuse it later by subject alone.
	reg := NewRegistrar([]Pattern{Pattern("svc." + id + "." + LifecycleService + ".v1.>")})

	if checker, ok := p.(ActivationChecker); ok {
		if err := reg.Mount(mountHook(id, lifecycleActivationCheck, func(ctx context.Context) lifecycleReply {
			return verdict(checker.CheckActivation(ctx))
		})); err != nil {
			return nil, err
		}
	}
	if readier, ok := p.(Readier); ok {
		if err := reg.Mount(mountHook(id, lifecycleReady, func(ctx context.Context) lifecycleReply {
			return verdict(readier.Ready(ctx))
		})); err != nil {
			return nil, err
		}
	}
	if health, ok := p.(HealthContributor); ok {
		if err := reg.Mount(mountHook(id, lifecycleHealth, func(ctx context.Context) lifecycleReply {
			return lifecycleReply{Result: health.HealthSections(ctx)}
		})); err != nil {
			return nil, err
		}
	}
	eps := reg.Endpoints()
	if len(eps) == 0 {
		return nil, nil
	}
	return b.Services().Serve(ctx, ServiceSpec{
		Name:        id + "-" + LifecycleService,
		Version:     "1.0.0",
		Description: "lifecycle hooks for " + id,
		// The queue group is the plugin id so two generations never both
		// answer one activation check.
		QueueGroup: id,
		Endpoints:  eps,
	})
}

func mountHook(id, hook string, run func(context.Context) lifecycleReply) EndpointSpec {
	return EndpointSpec{
		Name:    hook,
		Subject: LifecycleSubject(id, hook),
		Handler: func(ctx context.Context, m *Msg) {
			body, err := json.Marshal(run(ctx))
			if err != nil {
				_ = m.RespondFault(bus.Fault{Code: FaultSchema, Op: string(m.Subject), Message: err.Error()})
				return
			}
			_ = m.Respond(nil, body)
		},
	}
}

func verdict(err error) lifecycleReply {
	if err != nil {
		return lifecycleReply{Error: err.Error()}
	}
	return lifecycleReply{}
}
