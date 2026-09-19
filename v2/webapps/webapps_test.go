package webapps

import (
	"reflect"
	"testing"

	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/webapps"
)

func TestCommonContractIsReExportedExactly(t *testing.T) {
	pairs := [][2]reflect.Type{
		{reflect.TypeOf(Target{}), reflect.TypeOf(common.Target{})},
		{reflect.TypeOf(App{}), reflect.TypeOf(common.App{})},
		{reflect.TypeOf(RuntimeEvent{}), reflect.TypeOf(common.RuntimeEvent{})},
		{reflect.TypeOf(ConversationSendRequest{}), reflect.TypeOf(common.ConversationSendRequest{})},
		{reflect.TypeOf(ServicesLogsResult{}), reflect.TypeOf(common.ServicesLogsResult{})},
		{reflect.TypeOf(PreviewResolveResult{}), reflect.TypeOf(common.PreviewResolveResult{})},
	}
	for _, pair := range pairs {
		if pair[0] != pair[1] {
			t.Fatalf("Gateway type %v differs from common type %v", pair[0], pair[1])
		}
	}
	if List.Name() != CommandList || Create.Name() != CommandCreate || Changed.Name() != EventChanged || Events.Name() != "WEB_APPS_EVENTS" {
		t.Fatalf("descriptor aliases drifted: %q %q %q %q", List.Name(), Create.Name(), Changed.Name(), Events.Name())
	}
}

func TestNilPluginCallFailsClosed(t *testing.T) {
	_, err := Call(nil, nil, List, ListRequest{})
	if err == nil {
		t.Fatal("nil plugin accepted")
	}
}
