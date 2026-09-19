package pluginsdk_test

import (
	"reflect"
	"testing"

	sdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

func TestProjectsContributionAliasesMatchCommonSDK(t *testing.T) {
	pairs := [][2]reflect.Type{
		{reflect.TypeOf(sdk.ProjectViewContribution{}), reflect.TypeOf(plugin.ProjectViewContribution{})},
		{reflect.TypeOf(sdk.DirectoryContextActionContribution{}), reflect.TypeOf(plugin.DirectoryContextActionContribution{})},
		{reflect.TypeOf(sdk.ProjectDirectoryContext{}), reflect.TypeOf(plugin.ProjectDirectoryContext{})},
		{reflect.TypeOf(sdk.DirectoryContextActionEligibilityRequest{}), reflect.TypeOf(plugin.DirectoryContextActionEligibilityRequest{})},
		{reflect.TypeOf(sdk.DirectoryContextActionEligibilityResult{}), reflect.TypeOf(plugin.DirectoryContextActionEligibilityResult{})},
		{reflect.TypeOf(sdk.DirectoryContextActionWizardContext{}), reflect.TypeOf(plugin.DirectoryContextActionWizardContext{})},
	}
	for _, pair := range pairs {
		if pair[0] != pair[1] {
			t.Fatalf("Gateway v2 type %v must alias common v2 type %v", pair[0], pair[1])
		}
	}
	if sdk.DirectoryContextActionEligibilityCommand != plugin.DirectoryContextActionEligibilityCommand {
		t.Fatal("eligibility command differs from the common v2 SDK")
	}
}
