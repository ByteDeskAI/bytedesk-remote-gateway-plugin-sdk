package pluginsdk

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime/pprof"
	"testing"
)

// TestProfileCommandCapturesThePluginProcess proves a spawned plugin can profile
// itself on the host's request, and that every other path reaches the plugin's
// own handler untouched.
func TestProfileCommandCapturesThePluginProcess(t *testing.T) {
	var reached string
	h := withProfileCommand(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = r.URL.Path
		w.WriteHeader(http.StatusTeapot)
	}))
	serve := func(method, target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
		return rec
	}

	if rec := serve(http.MethodGet, "/tmux/api/tree"); rec.Code != http.StatusTeapot || reached != "/tmux/api/tree" {
		t.Fatalf("plugin route = %d reaching %q, want the plugin's own handler", rec.Code, reached)
	}
	if rec := serve(http.MethodGet, ProfileCommand+"?seconds=0"); rec.Code != http.StatusBadRequest {
		t.Fatalf("seconds=0 = %d, want 400", rec.Code)
	}
	if rec := serve(http.MethodPost, ProfileCommand); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d, want 405", rec.Code)
	}
	rec := serve(http.MethodGet, ProfileCommand+"?seconds=1")
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("capture = %d with %d bytes, want a profile", rec.Code, rec.Body.Len())
	}

	var busy bytes.Buffer
	if err := pprof.StartCPUProfile(&busy); err != nil {
		t.Skipf("cannot hold the profiler for the conflict case: %v", err)
	}
	defer pprof.StopCPUProfile()
	if rec := serve(http.MethodGet, ProfileCommand+"?seconds=1"); rec.Code != http.StatusConflict {
		t.Fatalf("capture while another runs = %d, want 409", rec.Code)
	}
}

// TestProfileCommandKeepsHealthzWithoutAHandler covers a plugin that serves no
// HTTP of its own: wrapping must not remove the socket server's /healthz.
func TestProfileCommandKeepsHealthzWithoutAHandler(t *testing.T) {
	h := withProfileCommand(nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("/healthz = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/elsewhere", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown path = %d, want 404", rec.Code)
	}
}
