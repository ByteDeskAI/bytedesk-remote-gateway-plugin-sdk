package pluginsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TerminalPresentationSelection binds the host-selected implementation to the
// owner identity that must appear in the request lease.
type TerminalPresentationSelection struct {
	PluginID   string
	ProviderID string
	Generation string
	Provider   TerminalPresentationProvider
}

// TerminalPresentationCurrent resolves principal-authorized terminal
// incarnations after provider execution and immediately before result admission.
type TerminalPresentationCurrent func(context.Context, PresentationRequest) ([]PresentationTerminal, error)

// DispatchTerminalPresentation invokes the owner/provider already selected by
// the host. Selection is deliberately an input to this function: multiple
// providers may implement TerminalPresentationCommand without competing for a
// process-global CommandHandler command name.
func DispatchTerminalPresentation(ctx context.Context, selected TerminalPresentationSelection, request PresentationRequest, current TerminalPresentationCurrent) (PresentationResult, error) {
	if selected.Provider == nil {
		return PresentationResult{}, fmt.Errorf("terminal presentation provider required")
	}
	if err := ValidatePresentationRequest(request); err != nil {
		return PresentationResult{}, fmt.Errorf("invalid terminal presentation request: %w", err)
	}
	if request.Lease.PluginID != selected.PluginID || request.Lease.ProviderID != selected.ProviderID || request.Lease.Generation != selected.Generation {
		return PresentationResult{}, fmt.Errorf("terminal presentation lease does not match selected owner")
	}
	if current == nil {
		return PresentationResult{}, fmt.Errorf("terminal presentation authority resolver required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(TerminalPresentationDeadlineMS)*time.Millisecond)
	defer cancel()
	result, err := selected.Provider.Project(ctx, request)
	if err != nil {
		return PresentationResult{}, err
	}
	authorized, err := current(ctx, request)
	if err != nil {
		return PresentationResult{}, err
	}
	if err := ValidatePresentationResult(request, authorized, result); err != nil {
		return PresentationResult{}, fmt.Errorf("invalid terminal presentation result: %w", err)
	}
	return result, nil
}

// TerminalPresentationHTTPHandler adapts one selected spawned-plugin provider
// to the external HTTPPlugin transport at /terminal.presentation.project.v1.
// It does not implement CommandHandler or perform provider selection; the host
// selects the plugin socket/owner before sending this request.
func TerminalPresentationHTTPHandler(selected TerminalPresentationSelection) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+TerminalPresentationCommand {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		request, err := DecodePresentationRequest(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// A spawned process cannot authoritatively resolve the host's current
		// principal set. This is a local conformance check; the host must validate
		// the response again against its current authorized incarnations.
		result, err := DispatchTerminalPresentation(r.Context(), selected, request, func(context.Context, PresentationRequest) ([]PresentationTerminal, error) {
			return request.Terminals, nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
