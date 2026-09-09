package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	if manifest.Protocol != nil {
		negotiator, ok := host.(Negotiator)
		if !ok {
			if manifest.Protocol.Major != 0 || len(manifest.Protocol.Required) != 0 {
				return fmt.Errorf("host does not support protocol negotiation")
			}
		} else {
			info, err := negotiator.Negotiate(ctx, *manifest.Protocol)
			if err != nil {
				return err
			}
			if err := CheckProtocol(info, *manifest.Protocol); err != nil {
				return err
			}
			if info.PluginID != p.ID() {
				return fmt.Errorf("negotiated plugin identity differs from manifest")
			}
			if info.Major != 0 && info.Generation == "" {
				return fmt.Errorf("negotiated generation required")
			}
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if starter, ok := host.(HostStarter); ok {
		if err := starter.Start(ctx); err != nil {
			return err
		}
	}
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
	if checker, ok := p.(ActivationChecker); ok {
		if err := checker.CheckActivation(ctx); err != nil {
			return err
		}
	}
	if readier, ok := p.(Readier); ok {
		if err := readier.Ready(ctx); err != nil {
			host.Logger().Warn("plugin health degraded", "reason", err.Error())
		}
	}
	cfgHTTP := Config{ID: p.ID(), Socket: cfg.Socket}
	if httpPlugin, ok := p.(HTTPPlugin); ok {
		cfgHTTP.Handler = httpPlugin.Handler()
	}
	return ServeContext(ctx, cfgHTTP)
}
