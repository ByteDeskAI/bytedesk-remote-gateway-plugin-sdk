package pluginsdk_test

import (
	"reflect"
	"testing"

	sdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
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
			t.Fatalf("Gateway type %v must alias common type %v", pair[0], pair[1])
		}
	}
	if sdk.DirectoryContextActionEligibilityCommand != plugin.DirectoryContextActionEligibilityCommand {
		t.Fatal("eligibility command differs from the common SDK")
	}
}

func TestGatewayManifestValidatesProjectsContributionOwnership(t *testing.T) {
	m := sdk.Manifest{
		ID: "web-apps", Version: "1.0.0",
		Panels:                  []sdk.PanelSpec{{ID: "main", Kind: "page", URL: "/main"}, {ID: "create", Kind: "page", URL: "/create"}},
		ProjectViews:            []sdk.ProjectViewContribution{{ID: "web-apps", Label: "Web Apps", Icon: "globe", PanelID: "main"}},
		DirectoryContextActions: []sdk.DirectoryContextActionContribution{{ID: "create", Label: "Create Web App", WizardPanelID: "create"}},
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	m.DirectoryContextActions[0].WizardPanelID = "foreign"
	if err := m.Validate(); err == nil {
		t.Fatal("foreign wizard panel accepted")
	}
}
