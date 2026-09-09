package pluginsdk_test

import (
	sdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	"testing"
)

func TestRequirementAliasExposesCanonicalVersionEvaluation(t *testing.T) {
	requirement := sdk.Requirement{ID: "peer", Version: "^1.2.0"}
	if ok, err := requirement.MatchesVersion("1.3.0"); err != nil || !ok {
		t.Fatalf("compatible peer: %v %v", ok, err)
	}
	if ok, err := requirement.MatchesVersion("2.0.0"); err != nil || ok {
		t.Fatalf("incompatible peer: %v %v", ok, err)
	}
	requirement.Version = "not-a-range"
	if err := requirement.ValidateVersionConstraint(); err == nil {
		t.Fatal("alias accepted malformed constraint")
	}
}
