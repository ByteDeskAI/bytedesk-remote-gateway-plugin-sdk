package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRPCRequestAloneForwardsSubjectLeaseOutsideJSON(t *testing.T) {
	type observation struct {
		path    string
		headers []string
		body    []byte
	}
	var seen []observation
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		seen = append(seen, observation{path: r.URL.Path, headers: append([]string(nil), r.Header.Values(HeaderSubjectLease)...), body: body.Bytes()})
		if r.URL.Path == "/request" {
			_ = json.NewEncoder(w).Encode(Envelope{Type: "reply"})
			return
		}
		if r.URL.Path == "/negotiate" {
			_ = json.NewEncoder(w).Encode(HostCapabilities{})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	h := NewHost("").(*rpcHost)
	h.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}

	req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil)
	req.Header.Set(HeaderSubjectLease, testSubjectLease)
	ctx := ContextForRequest(req)
	if _, err := h.Request(ctx, Envelope{Type: "query"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Publish(Envelope{Type: "event"}); err != nil {
		t.Fatal(err)
	}
	_, _ = h.Negotiate(ctx, ProtocolRequirements{})
	h.Subscribe("event", func(Envelope) {})
	h.Every(time.Second, func() {})

	if len(seen) != 5 {
		t.Fatalf("requests = %d", len(seen))
	}
	for _, got := range seen {
		if bytes.Contains(got.body, []byte(testSubjectLease)) {
			t.Fatalf("%s JSON leaked subject lease: %s", got.path, got.body)
		}
		if got.path == "/request" {
			if len(got.headers) != 1 || got.headers[0] != testSubjectLease {
				t.Fatalf("request subject header = %v", got.headers)
			}
		} else if len(got.headers) != 0 {
			t.Fatalf("%s forwarded subject header %v", got.path, got.headers)
		}
	}
}

func TestRPCRequestWithoutSubjectLeaseStaysAutonomous(t *testing.T) {
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append([]string(nil), r.Header.Values(HeaderSubjectLease)...)
		_ = json.NewEncoder(w).Encode(Envelope{Type: "reply"})
	}))
	defer srv.Close()
	h := NewHost("").(*rpcHost)
	h.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}
	if _, err := h.Request(context.Background(), Envelope{Type: "query"}); err != nil {
		t.Fatal(err)
	}
	if len(headers) != 0 {
		t.Fatalf("autonomous request carried subject headers %v", headers)
	}
}

func TestRPCRequestPreservesMalformedOrDuplicateSubjectForHostRefusal(t *testing.T) {
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append([]string(nil), r.Header.Values(HeaderSubjectLease)...)
		_ = json.NewEncoder(w).Encode(Envelope{Type: "reply"})
	}))
	defer srv.Close()
	h := NewHost("").(*rpcHost)
	h.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}
	req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil)
	req.Header[HeaderSubjectLease] = []string{"invalid", testSubjectLease}
	ctx := ContextForRequest(req)
	if _, err := SubjectLeaseFromContext(ctx); !errors.Is(err, ErrInvalidSubjectLease) {
		t.Fatalf("context error = %v", err)
	}
	if _, err := h.Request(ctx, Envelope{Type: "query"}); err != nil {
		t.Fatal(err)
	}
	if len(headers) != 2 || headers[0] != "invalid" || headers[1] != testSubjectLease {
		t.Fatalf("forwarded subject headers = %v", headers)
	}
}

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
