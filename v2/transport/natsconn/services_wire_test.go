package natsconn_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/internal/fakebroker"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/aidecision"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

func TestServiceDiscoveryWirePreservesPointAndContractMetadata(t *testing.T) {
	b := fakebroker.Start(t)
	c := dial(t, b)
	ctx := context.Background()
	const name = "svc.files.ai.decision.v1.start"
	const point = plugin.PointAIDecision
	command := plugin.NewValidatedCommand[aidecision.ProviderStartRequest, aidecision.StartResult](name, 1, strings.Repeat("a", 64), name)
	svc, err := plugin.ServeAtPoint(ctx, c, command, point, func(context.Context, aidecision.ProviderStartRequest, bus.Caller) (aidecision.StartResult, error) {
		t.Error("discovery must not invoke the provider")
		return aidecision.StartResult{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Stop(ctx) })
	reply, err := c.Request(ctx, "$SRV.INFO."+name, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bus.ServiceInfo
	if err := json.Unmarshal(reply.Data, &found); err != nil {
		t.Fatal(err)
	}
	if len(found.Endpoints) != 1 || found.Endpoints[0].Point != string(point) || found.Endpoints[0].Subject != name {
		t.Fatalf("discovery lost endpoint identity: %+v", found.Endpoints)
	}
	if !reflect.DeepEqual(found.Metadata, svc.Info().Metadata) || found.Metadata[plugin.MetadataContractSchema] != command.Descriptor().SchemaHash() {
		t.Fatalf("discovery lost contract metadata: %+v", found.Metadata)
	}
	bound, err := plugin.BindDiscoveredCommand[aidecision.ProviderStartRequest, aidecision.StartResult](found, "files", point, "start", 1)
	if err != nil || bound.Descriptor().SchemaHash() != command.Descriptor().SchemaHash() {
		t.Fatalf("wire service cannot bind its declared extension point: %v", err)
	}
}
