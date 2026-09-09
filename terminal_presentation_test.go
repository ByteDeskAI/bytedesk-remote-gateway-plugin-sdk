package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type presentationProviderFunc func(context.Context, PresentationRequest) (PresentationResult, error)

func (f presentationProviderFunc) Project(ctx context.Context, request PresentationRequest) (PresentationResult, error) {
	return f(ctx, request)
}

func presentationRequest() PresentationRequest {
	return PresentationRequest{
		Lease:     PresentationLease{HostEpoch: "epoch", PluginID: "owner", ProviderID: "provider-b", Generation: "g1", SubjectLease: "subject", RequestID: "request", ViewRevision: "view"},
		Terminals: []PresentationTerminal{{TerminalID: "terminal-1", Context: TerminalBindingContext{Kind: TerminalBindingNone}}},
	}
}

func TestDispatchTerminalPresentationUsesSelectedProvider(t *testing.T) {
	request := presentationRequest()
	wrongCalled := false
	wrong := presentationProviderFunc(func(context.Context, PresentationRequest) (PresentationResult, error) {
		wrongCalled = true
		return PresentationResult{}, nil
	})
	_ = wrong // A second provider can coexist without registering a command name.
	selected := presentationProviderFunc(func(ctx context.Context, got PresentationRequest) (PresentationResult, error) {
		if got.Lease.ProviderID != "provider-b" {
			t.Fatalf("provider id = %q", got.Lease.ProviderID)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("selected provider received no deadline")
		}
		return PresentationResult{Lease: got.Lease, MaxAgeMS: 1000, Items: []PresentationItem{}}, nil
	})
	selection := TerminalPresentationSelection{PluginID: "owner", ProviderID: "provider-b", Generation: "g1", Provider: selected}
	result, err := DispatchTerminalPresentation(context.Background(), selection, request, func(context.Context, PresentationRequest) ([]PresentationTerminal, error) {
		return request.Terminals, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if wrongCalled || result.Lease.ProviderID != "provider-b" {
		t.Fatal("dispatch did not remain scoped to the selected provider")
	}
}

func TestDispatchTerminalPresentationChecksOwnerAndLateAuthority(t *testing.T) {
	request := presentationRequest()
	current := append([]PresentationTerminal(nil), request.Terminals...)
	selection := TerminalPresentationSelection{
		PluginID: "owner", ProviderID: "provider-b", Generation: "g1",
		Provider: presentationProviderFunc(func(context.Context, PresentationRequest) (PresentationResult, error) {
			current = nil // Revoked while the provider was running.
			return PresentationResult{Lease: request.Lease, MaxAgeMS: 1000, Items: []PresentationItem{{TerminalID: "terminal-1", GroupPath: []PresentationGroup{}, Badges: []PresentationBadge{}, Freshness: FreshnessFresh}}}, nil
		}),
	}
	resolver := func(context.Context, PresentationRequest) ([]PresentationTerminal, error) { return current, nil }
	if _, err := DispatchTerminalPresentation(context.Background(), selection, request, resolver); err == nil {
		t.Fatal("result admitted after terminal authority was revoked")
	}
	selection.ProviderID = "provider-a"
	if _, err := DispatchTerminalPresentation(context.Background(), selection, request, resolver); err == nil {
		t.Fatal("lease admitted for a different selected provider")
	}
}

func TestTerminalPresentationHTTPHandlerUsesExternalHTTPTransport(t *testing.T) {
	request := presentationRequest()
	provider := presentationProviderFunc(func(_ context.Context, got PresentationRequest) (PresentationResult, error) {
		return PresentationResult{Lease: got.Lease, MaxAgeMS: 1000, Items: []PresentationItem{}}, nil
	})
	handler := TerminalPresentationHTTPHandler(TerminalPresentationSelection{PluginID: "owner", ProviderID: "provider-b", Generation: "g1", Provider: provider})
	raw, _ := json.Marshal(request)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/"+TerminalPresentationCommand, bytes.NewReader(raw)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got, err := DecodePresentationResult(recorder.Body, request, request.Terminals); err != nil || got.Lease != request.Lease {
		t.Fatalf("response = %#v, %v", got, err)
	}
}

func TestTerminalPresentationHTTPHandlerRejectsUnselectedSurfaceAndBadInput(t *testing.T) {
	handler := TerminalPresentationHTTPHandler(TerminalPresentationSelection{PluginID: "owner", ProviderID: "provider-b", Generation: "g1", Provider: presentationProviderFunc(func(_ context.Context, request PresentationRequest) (PresentationResult, error) {
		return PresentationResult{Lease: request.Lease, MaxAgeMS: 1, Items: []PresentationItem{}}, nil
	})})

	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/" + TerminalPresentationCommand, "", http.StatusMethodNotAllowed},
		{http.MethodPost, "/another-provider", "{}", http.StatusNotFound},
		{http.MethodPost, "/" + TerminalPresentationCommand, `{"lease":{},"terminals":[],"extra":true}`, http.StatusBadRequest},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body)))
		if recorder.Code != tc.status {
			t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, recorder.Code, tc.status)
		}
	}
}
