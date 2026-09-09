package pluginsdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type lifecyclePlugin struct {
	startErr, activationErr, stopErr error
	started, stopped                 bool
	protocol                         *ProtocolRequirements
}

func (*lifecyclePlugin) ID() string                              { return "sample" }
func (p *lifecyclePlugin) Manifest() Manifest                    { return Manifest{ID: p.ID(), Protocol: p.protocol} }
func (p *lifecyclePlugin) Start(context.Context, Host) error     { p.started = true; return p.startErr }
func (p *lifecyclePlugin) CheckActivation(context.Context) error { return p.activationErr }
func (p *lifecyclePlugin) Stop(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("Stop requires deadline")
	}
	p.stopped = true
	return p.stopErr
}

type lifecycleHost struct{ info HostCapabilities }

func (lifecycleHost) Publish(Envelope) error                              { return nil }
func (lifecycleHost) Subscribe(string, func(Envelope)) func()             { return func() {} }
func (lifecycleHost) Request(context.Context, Envelope) (Envelope, error) { return Envelope{}, nil }
func (lifecycleHost) Logger() Logger                                      { return nil }
func (lifecycleHost) StateDir(string) string                              { return "" }
func (lifecycleHost) Every(time.Duration, func()) func()                  { return func() {} }
func (lifecycleHost) BumpContributions()                                  {}
func (h lifecycleHost) Negotiate(context.Context, ProtocolRequirements) (HostCapabilities, error) {
	return h.info, nil
}

func TestServePluginRollsBackPartialStartAndActivationBeforeListening(t *testing.T) {
	t.Setenv(EnvID, "sample")
	for _, failStart := range []bool{true, false} {
		failure, cleanup := errors.New("cannot activate"), errors.New("cleanup failure")
		p := &lifecyclePlugin{stopErr: cleanup}
		if failStart {
			p.startErr = failure
		} else {
			p.activationErr = failure
		}
		socket := filepath.Join(t.TempDir(), "plugin.sock")
		err := ServePlugin(context.Background(), p, PluginConfig{Socket: socket, Host: lifecycleHost{}})
		if !errors.Is(err, failure) || !errors.Is(err, cleanup) {
			t.Fatalf("lost lifecycle errors: %v", err)
		}
		if !p.started || !p.stopped {
			t.Fatal("partial generation was not stopped")
		}
		if _, err := os.Stat(socket); !os.IsNotExist(err) {
			t.Fatal("failed generation opened a serving socket")
		}
	}
}

func TestServePluginChecksDirectNegotiationBeforeStart(t *testing.T) {
	t.Setenv(EnvID, "sample")
	for _, info := range []HostCapabilities{
		{Major: 1, PluginID: "sample", Generation: "g1"},
		{Major: 1, PluginID: "foreign", Generation: "g1", Features: []string{FeatureScopedHost}},
		{Major: 1, PluginID: "sample", Features: []string{FeatureScopedHost}},
	} {
		p := &lifecyclePlugin{protocol: &ProtocolRequirements{Major: 1, Required: []string{FeatureScopedHost}}}
		err := ServePlugin(context.Background(), p, PluginConfig{Host: lifecycleHost{info: info}})
		if err == nil || p.started || p.stopped {
			t.Fatalf("unsupported host reached lifecycle: err=%v plugin=%+v", err, p)
		}
	}
}
