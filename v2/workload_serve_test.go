package pluginsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

func TestServeRefusesWorkloadAuthOnUnsupportedInjectedBus(t *testing.T) {
	mem := memoryBus(t, "files")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sub, err := mem.Subscribe(ctx, pluginsdk.Pattern(pluginsdk.NegotiateSubject), func(_ context.Context, msg *bus.Msg) {
		var response pluginsdk.HostCapabilities
		_ = json.Unmarshal(negotiateReply("files", bus.Capabilities{Services: true, MaxPayload: 64 << 10}), &response)
		response.Features = append(response.Features, pluginsdk.FeatureHostWorkloadAuth)
		response.HostCallToken = strings.Repeat("a", 64)
		raw, _ := json.Marshal(response)
		_ = msg.Respond(nil, raw)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()
	p := &testPlugin{id: "files"}
	err = pluginsdk.ServePlugin(ctx, p, pluginsdk.PluginConfig{Bus: mem})
	var fault bus.Fault
	if !errors.As(err, &fault) || fault.Code != bus.FaultUnsupported {
		t.Fatalf("got %v", err)
	}
	if p.did(func(p *testPlugin) bool { return p.started || p.validated }) {
		t.Fatal("plugin ran with an unbound workload")
	}
}
