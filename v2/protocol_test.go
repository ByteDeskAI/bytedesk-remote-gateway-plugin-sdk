package pluginsdk_test

import (
	"strings"
	"testing"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

func pluginsdkVersion() string { return pluginsdk.Version() }

// TestCheckProtocolFailsClosed is the handshake gate's table. Every row is a
// way the host and the plugin can disagree, and every one of them must REFUSE:
// a plugin that runs on a partial match fails later, in production, with a
// symptom that does not name this handshake.
func TestCheckProtocolFailsClosed(t *testing.T) {
	ok := pluginsdk.HostCapabilities{
		Major:      pluginsdk.ProtocolMajor,
		PluginID:   "files",
		Generation: "gen-1",
		Features:   []string{pluginsdk.FeatureLifecycleEndpoints},
	}
	full := bus.Capabilities{Durable: true, KV: true, Services: true, MaxPayload: 64 << 10}

	for _, tc := range []struct {
		name      string
		have      pluginsdk.HostCapabilities
		need      pluginsdk.ProtocolRequirements
		needs     []string
		effective bus.Capabilities
		wantCode  string
		wantIn    string
	}{
		{
			name: "matching", have: ok, need: pluginsdk.ProtocolRequirements{Major: 2},
			effective: full,
		},
		{
			name: "an unset major defaults to this generation's",
			have: ok, need: pluginsdk.ProtocolRequirements{}, effective: full,
		},
		{
			name: "a v1 plugin against a v2 host",
			have: ok, need: pluginsdk.ProtocolRequirements{Major: 1}, effective: full,
			wantCode: pluginsdk.FaultUnsupported, wantIn: "major 1 is unsupported",
		},
		{
			name: "a v2 plugin against a v1 host",
			have: pluginsdk.HostCapabilities{Major: 1, Generation: "gen-1"},
			need: pluginsdk.ProtocolRequirements{Major: 2}, effective: full,
			wantCode: pluginsdk.FaultUnsupported, wantIn: "host 1",
		},
		{
			name: "a required feature the host does not advertise",
			have: ok, need: pluginsdk.ProtocolRequirements{Major: 2, Required: []string{"time.travel.v1"}},
			effective: full,
			wantCode:  pluginsdk.FaultUnsupported, wantIn: "time.travel.v1",
		},
		{
			name: "an empty required feature is not a free pass",
			have: ok, need: pluginsdk.ProtocolRequirements{Major: 2, Required: []string{""}},
			effective: full,
			wantCode:  pluginsdk.FaultUnsupported,
		},
		{
			name: "a need the effective substrate lacks",
			have: ok, need: pluginsdk.ProtocolRequirements{Major: 2},
			needs:     []string{"durable", "objects"},
			effective: bus.Capabilities{Durable: true, Services: true},
			wantCode:  pluginsdk.FaultUnsupported, wantIn: "objects",
		},
		{
			name: "the host's own refusal is carried, not discarded",
			have: pluginsdk.HostCapabilities{Major: 2, Generation: "gen-1", Error: "consent withdrawn"},
			need: pluginsdk.ProtocolRequirements{Major: 2}, effective: full,
			wantCode: pluginsdk.FaultDenied, wantIn: "consent withdrawn",
		},
		{
			name: "a host that answers with no generation",
			have: pluginsdk.HostCapabilities{Major: 2, PluginID: "files"},
			need: pluginsdk.ProtocolRequirements{Major: 2}, effective: full,
			wantCode: pluginsdk.FaultSchema,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.CheckProtocol(tc.have, tc.need, tc.needs, tc.effective)
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want a refusal, got nil")
			}
			f, ok := err.(bus.Fault)
			if !ok {
				t.Fatalf("err = %v (%T), want a bus.Fault", err, err)
			}
			if f.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q (%v)", f.Code, tc.wantCode, err)
			}
			if tc.wantIn != "" && !strings.Contains(f.Message, tc.wantIn) {
				t.Errorf("message %q does not mention %q", f.Message, tc.wantIn)
			}
		})
	}
}

// TestNegotiateSubjectIsInTheHostNamespace: a plugin must be able to reach the
// handshake with no manifest permission at all, which is only true while the
// subject is a host command.
func TestNegotiateSubjectIsInTheHostNamespace(t *testing.T) {
	if _, err := pluginsdk.ParseSubject(string(pluginsdk.NegotiateSubject)); err != nil {
		t.Fatalf("the negotiate subject is not a valid subject: %v", err)
	}
	if !strings.HasPrefix(string(pluginsdk.NegotiateSubject), "cmd.plugin.v1.") {
		t.Fatalf("negotiate subject %q left the host command namespace", pluginsdk.NegotiateSubject)
	}
}
