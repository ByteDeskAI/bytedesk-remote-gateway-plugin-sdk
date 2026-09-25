package pluginsdk

import (
	"strings"
	"testing"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

func TestHostWorkloadFeatureFailsClosed(t *testing.T) {
	have := HostCapabilities{Major: 2, Generation: "generation", Features: []string{FeatureHostWorkloadAuth}}
	need := ProtocolRequirements{Major: 2, Required: []string{FeatureHostWorkloadAuth}}
	for _, token := range []string{"", "secret", strings.Repeat("A", 64)} {
		have.HostCallToken = token
		if err := CheckProtocol(have, need, nil, bus.Capabilities{}); err == nil {
			t.Fatal("advertised workload authentication accepted malformed token")
		}
	}
	have.HostCallToken = strings.Repeat("a", 64)
	if err := CheckProtocol(have, need, nil, bus.Capabilities{}); err != nil {
		t.Fatal(err)
	}
	have.Features = nil
	if err := CheckProtocol(have, need, nil, bus.Capabilities{}); err == nil {
		t.Fatal("missing workload feature accepted")
	}
	if err := CheckProtocol(have, ProtocolRequirements{Major: 2}, nil, bus.Capabilities{}); err == nil {
		t.Fatal("unnegotiated token accepted")
	}
}

func TestWorkloadIdentityGenerationMustMatch(t *testing.T) {
	have := HostCapabilities{PluginID: "plugin", Generation: "one", Identity: bus.Identity{PluginID: "plugin", Generation: "two"}}
	if err := checkIdentity(have, "plugin"); err == nil {
		t.Fatal("conflicting generation accepted")
	}
	have.Identity.Generation = "one"
	if err := checkIdentity(have, "plugin"); err != nil {
		t.Fatal(err)
	}
}
