package natsconn_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/internal/fakebroker"
	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/transport/natsconn"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

func TestRequestCarriesBoundWorkloadOnlyToHost(t *testing.T) {
	b := fakebroker.Start(t)
	c := dial(t, b)
	token := strings.Repeat("a", 64)
	if err := c.BindHostCallToken(token); err != nil {
		t.Fatal(err)
	}
	for _, subject := range []string{"cmd.gateway.ai-decision.v1.start", "svc.provider.v1.evaluate"} {
		seen := make(chan string, 1)
		b.Respond(subject, func(_ string, h map[string]string, _ []byte) []byte {
			seen <- h[strings.ToLower(natsconn.HeaderHostCallToken)]
			return []byte(`{}`)
		})
		if _, err := c.Request(context.Background(), bus.Subject(subject), []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
		want := ""
		if strings.HasPrefix(subject, "cmd.gateway.") {
			want = token
		}
		if got := <-seen; got != want {
			t.Fatalf("workload header not correctly scoped on %s", subject)
		}
	}
}
