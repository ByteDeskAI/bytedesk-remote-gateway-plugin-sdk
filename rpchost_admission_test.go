package pluginsdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCallbackAdmissionRejectsHTTPFailureAndAllowsRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	h := NewHost("").(*rpcHost)
	h.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer h.Close()
	if err := h.Start(ctx); err == nil {
		t.Fatal("denied callback connection admitted")
	}
	if err := h.Start(ctx); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

type rewriteHostTransport struct {
	base     string
	delegate http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = t.base[len("http://"):]
	return t.delegate.RoundTrip(clone)
}
func TestCallbackAdmissionHonorsCallerCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	h := NewHost("").(*rpcHost)
	h.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}
	defer h.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := h.Start(ctx); err == nil {
		t.Fatal("canceled callback admission succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("caller cancellation ignored")
	}
}
func TestNewHostUsesEnvironmentSocket(t *testing.T) {
	t.Setenv(EnvHostSocket, "/fixture/host.sock")
	if got := NewHost("").(*rpcHost).socket; got != "/fixture/host.sock" {
		t.Fatalf("socket %q", got)
	}
	if got := NewHost("/explicit.sock").(*rpcHost).socket; got != "/explicit.sock" {
		t.Fatalf("explicit socket %q", got)
	}
}
