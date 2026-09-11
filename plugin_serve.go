package pluginsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"time"
)

// PluginConfig hosts the same Plugin implementation used in linked mode.
// Host is normally nil (use GATEWAY_HOST_SOCKET); supplying one supports
// independent construction/conformance tests without a gateway Server.
type PluginConfig struct {
	Socket string
	Host   Host
}

// ServePlugin negotiates, validates and starts a plugin before listening.
// Partial starts are always stopped. Existing Readier semantics are retained:
// degraded health is logged, whereas ActivationChecker failure prevents serving.
//
// The lifecycle hooks the plugin implements are advertised in negotiation. A
// hook the host acknowledges is called by the host over LifecycleHookCommand
// and is not run here; an older host that rejects the advertisement leaves
// every hook running locally, as before.
func ServePlugin(ctx context.Context, p Plugin, cfg PluginConfig) (result error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p == nil {
		return fmt.Errorf("plugin required")
	}
	manifest := p.Manifest()
	if p.ID() == "" || p.ID() != manifest.ID {
		return fmt.Errorf("plugin and manifest identity must match")
	}
	if id := os.Getenv(EnvID); id != "" && id != p.ID() {
		return fmt.Errorf("plugin identity differs from host assignment")
	}
	if err := manifest.ValidateDiscover(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	host := cfg.Host
	if host == nil {
		socket := os.Getenv(EnvHostSocket)
		if socket == "" {
			return fmt.Errorf("%s required", EnvHostSocket)
		}
		host = NewHost(socket)
		defer host.(HostCloser).Close()
	}
	hostHooks, err := negotiate(ctx, host, p.ID(), manifest.Protocol, DeclaredHooks(p))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if starter, ok := host.(HostStarter); ok {
		if err := starter.Start(ctx); err != nil {
			return err
		}
	}
	// Validate stays in the child: it runs before Start and before the plugin
	// socket exists, so there is nothing for the host to call it over.
	if validator, ok := p.(Validator); ok {
		if err := validator.Validate(ctx, host); err != nil {
			return err
		}
	}
	defer func() {
		cancel()
		stopCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		result = errors.Join(result, p.Stop(stopCtx))
	}()
	if err := p.Start(ctx, host); err != nil {
		return err
	}
	if faults, ok := host.(HostCloser); ok && faults.LastFault() != nil {
		return faults.LastFault()
	}
	if checker, ok := p.(ActivationChecker); ok && !slices.Contains(hostHooks, HookActivationCheck) {
		if err := checker.CheckActivation(ctx); err != nil {
			return err
		}
	}
	if readier, ok := p.(Readier); ok && !slices.Contains(hostHooks, HookReady) {
		if err := readier.Ready(ctx); err != nil {
			host.Logger().Warn("plugin health degraded", "reason", err.Error())
		}
	}
	cfgHTTP := Config{ID: p.ID(), Socket: cfg.Socket}
	if httpPlugin, ok := p.(HTTPPlugin); ok {
		cfgHTTP.Handler = httpPlugin.Handler()
	}
	// A spawned plugin is its own process, so the host can only profile it by
	// asking it to profile itself (ProfileCommand).
	cfgHTTP.Handler = withProfileCommand(cfgHTTP.Handler)
	if len(hostHooks) != 0 {
		cfgHTTP.Handler = lifecycleHookHandler(p, hostHooks, cfgHTTP.Handler)
	}
	return ServeContext(ctx, cfgHTTP)
}

// negotiate returns the hooks the host acknowledged. A plugin with no protocol
// declaration used to skip negotiation entirely, so for it any failure only
// means the hooks run locally. A declared protocol retries once without hooks
// (an older host rejects the unknown field) and still fails closed after that.
func negotiate(ctx context.Context, host Host, id string, declared *ProtocolRequirements, hooks []string) ([]string, error) {
	if declared == nil && len(hooks) == 0 {
		return nil, nil
	}
	negotiator, ok := host.(Negotiator)
	if !ok {
		if declared != nil && (declared.Major != 0 || len(declared.Required) != 0) {
			return nil, fmt.Errorf("host does not support protocol negotiation")
		}
		return nil, nil
	}
	need := ProtocolRequirements{Major: ProtocolMajor}
	if declared != nil {
		need = *declared
		if need.Major == 0 && len(hooks) != 0 {
			need.Major = ProtocolMajor
		}
	}
	need.Hooks = hooks
	info, err := negotiateOnce(ctx, negotiator, id, need)
	if err == nil || len(hooks) == 0 {
		return info.Hooks, err
	}
	if declared == nil {
		return nil, nil
	}
	_, err = negotiateOnce(ctx, negotiator, id, *declared)
	return nil, err
}

func negotiateOnce(ctx context.Context, negotiator Negotiator, id string, need ProtocolRequirements) (HostCapabilities, error) {
	info, err := negotiator.Negotiate(ctx, need)
	if err != nil {
		return HostCapabilities{}, err
	}
	if err := CheckProtocol(info, need); err != nil {
		return HostCapabilities{}, err
	}
	if info.PluginID != id {
		return HostCapabilities{}, fmt.Errorf("negotiated plugin identity differs from manifest")
	}
	if info.Major != 0 && info.Generation == "" {
		return HostCapabilities{}, fmt.Errorf("negotiated generation required")
	}
	return info, nil
}

// lifecycleHookHandler serves LifecycleHookCommand for the acknowledged hooks
// and passes every other path to the plugin's own handler. A hook's verdict is
// a 200 with a non-empty error; a transport or unknown-hook failure is not.
func lifecycleHookHandler(p Plugin, hooks []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+LifecycleHookCommand {
			if next == nil {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Hook string `json:"hook"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&req); err != nil || !slices.Contains(hooks, req.Hook) {
			http.Error(w, "unknown lifecycle hook", http.StatusNotFound)
			return
		}
		var reply struct {
			Error  string `json:"error"`
			Result any    `json:"result"`
		}
		var err error
		// CheckProtocol limited hooks to DeclaredHooks(p), so each assertion holds.
		switch req.Hook {
		case HookActivationCheck:
			err = p.(ActivationChecker).CheckActivation(r.Context())
		case HookReady:
			err = p.(Readier).Ready(r.Context())
		case HookHealth:
			reply.Result = p.(HealthContributor).HealthSections(r.Context())
		}
		if err != nil {
			reply.Error = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply)
	})
}
