package natsconn

import (
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestHostCallTokenBoundAndHostOnly(t *testing.T) {
	c := &Conn{}
	token := strings.Repeat("a", 64)
	if err := c.BindHostCallToken(token); err != nil {
		t.Fatal(err)
	}
	if err := c.BindHostCallToken(token); err != nil {
		t.Fatal(err)
	}
	if err := c.BindHostCallToken(strings.Repeat("b", 64)); err == nil {
		t.Fatal("rebound live identity")
	}
	for _, subject := range []string{"cmd.gateway.ai-decision.v1.start", "svc.other.v1.request", "event.gateway.started", ""} {
		h := nats.Header{strings.ToLower(HeaderHostCallToken): []string{"forged"}, "Other": []string{"keep"}}
		c.bindHostCallHeader(h, subject)
		want := ""
		if strings.HasPrefix(subject, "cmd.gateway.") {
			want = token
		}
		if h.Get(HeaderHostCallToken) != want || h.Get("Other") != "keep" {
			t.Fatalf("unexpected header behavior for %s", subject)
		}
		if _, exists := h[strings.ToLower(HeaderHostCallToken)]; exists {
			t.Fatal("caller-supplied spelling survived")
		}
	}
}

func TestHostCallTokenRejectsMalformedWithoutDisclosure(t *testing.T) {
	for _, token := range []string{"", "secret", strings.Repeat("A", 64), strings.Repeat("a", 65)} {
		c := &Conn{}
		err := c.BindHostCallToken(token)
		if err == nil {
			t.Fatal("malformed token accepted")
		}
		if token != "" && strings.Contains(err.Error(), token) {
			t.Fatal("token disclosed in error")
		}
	}
}
