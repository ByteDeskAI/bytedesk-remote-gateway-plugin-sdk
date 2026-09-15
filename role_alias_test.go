package pluginsdk_test

import (
	"reflect"
	"testing"

	sdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

func TestContributionRoleAliasesMatchCommonSDK(t *testing.T) {
	if reflect.TypeOf(sdk.ContributionRole{}) != reflect.TypeOf(plugin.ContributionRole{}) || reflect.TypeOf(sdk.ContributionEligibility("")) != reflect.TypeOf(plugin.ContributionEligibility("")) {
		t.Fatal("contribution types must be aliases")
	}
	if sdk.ContributionInstalledAllowed != plugin.ContributionInstalledAllowed || sdk.ContributionConsentRequired != plugin.ContributionConsentRequired || sdk.ContributionCompiledOnly != plugin.ContributionCompiledOnly {
		t.Fatal("eligibility constants differ")
	}
	roles := sdk.ContributionRoles()
	canonical := plugin.ContributionRoles()
	if len(roles) == 0 || !reflect.DeepEqual(roles, canonical) {
		t.Fatal("role tables differ or are empty")
	}
	for _, role := range canonical {
		got, ok := sdk.ContributionRoleFor(role.Slot)
		if !ok || got != role {
			t.Fatalf("lookup differs for %s", role.Slot)
		}
		for _, compiled := range []bool{false, true} {
			for _, consent := range []bool{false, true} {
				if sdk.ContributionRoleAllowed(role.Slot, compiled, consent) != plugin.ContributionRoleAllowed(role.Slot, compiled, consent) {
					t.Fatalf("policy differs for %s compiled=%v consent=%v", role.Slot, compiled, consent)
				}
			}
		}
	}
	for _, unknown := range []string{"", "*", "MAIN-NAVIGATION", "unknown"} {
		if _, ok := sdk.ContributionRoleFor(unknown); ok {
			t.Fatalf("unknown role found: %q", unknown)
		}
		for _, compiled := range []bool{false, true} {
			for _, consent := range []bool{false, true} {
				if sdk.ContributionRoleAllowed(unknown, compiled, consent) {
					t.Fatalf("unknown role admitted: %q", unknown)
				}
			}
		}
	}
	for i := range roles {
		roles[i].Eligibility = sdk.ContributionInstalledAllowed
		roles[i].Slot = "mutated"
	}
	if !reflect.DeepEqual(sdk.ContributionRoles(), canonical) || !reflect.DeepEqual(plugin.ContributionRoles(), canonical) {
		t.Fatal("returned role table mutated canonical policy")
	}
}
