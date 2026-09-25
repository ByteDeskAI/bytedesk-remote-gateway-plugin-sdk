package pluginsdk

import "testing"

func TestCapabilityExportsAreTheCatalog(t *testing.T) {
	m := Manifest{Capabilities: []string{CapabilityIngressPublish, CapabilityCredentialSecret}}
	lines := ConsentCapabilities(m)
	if len(lines) != 2 || lines[0].ID != CapabilityCredentialSecret || lines[0].Sentence == "" {
		t.Fatalf("ConsentCapabilities = %+v", lines)
	}
	if CapabilityEnabled(nil, lines[0].ID) {
		t.Fatal("empty grant enabled a catalog id")
	}
	if !CapabilityEnabled([]string{lines[0].ID}, lines[0].ID) {
		t.Fatal("granted catalog id was not enabled")
	}
}
