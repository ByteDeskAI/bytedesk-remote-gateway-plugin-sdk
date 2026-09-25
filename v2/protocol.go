package pluginsdk

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

// NegotiateSubject is where a plugin asks the host what it is running under.
// It is a command in the host's own namespace, so a plugin needs no manifest
// permission to reach it: cmd.plugin.v1.> is granted by being a plugin.
const NegotiateSubject Subject = "cmd.plugin.v1.negotiate"

// Protocol features a v2 host may advertise. A plugin that cannot run without
// one lists it in manifest protocol.required and the handshake fails closed.
const (
	FeatureLifecycleEndpoints = "lifecycle.endpoints.v1"
	FeatureHTTPRoutes         = "http.routes.v1"
	FeatureUIModuleMount      = plugin.FeatureUIModuleMount
	FeatureGrantsDigest       = "grants.digest.v2"
	// FeatureHostWorkloadAuth binds native host calls to the negotiated plugin
	// generation. It is separate from a human subject lease and grants no verbs.
	FeatureHostWorkloadAuth = "host.workload-auth.v1"
)

// NegotiateRequest is what a plugin sends. It is an assertion of identity and
// a statement of requirements; it grants nothing.
type NegotiateRequest struct {
	PluginID string               `json:"pluginId"`
	Protocol ProtocolRequirements `json:"protocol"`
	// Needs are the substrate capabilities from the manifest, repeated here so
	// the host can refuse a plugin it cannot serve before it starts.
	Needs []string `json:"needs,omitempty"`
	// Inbox is this connection's reply prefix, so the host can confirm the
	// credential it issued matches the connection that is using it.
	Inbox string `json:"inbox"`
	// GrantsDigest is what the operator consented to. A host that sees a digest
	// it did not approve refuses rather than running on stale consent.
	GrantsDigest string `json:"grantsDigest,omitempty"`
	// SDK is this module's version, for operator-visible diagnostics only.
	SDK string `json:"sdk,omitempty"`
}

// HostCapabilities is what the host answers.
//
// Everything a generation needs arrives in this one reply, because a handshake
// that took several round trips would have several places to be half-complete.
type HostCapabilities struct {
	Major      uint32   `json:"major"`
	PluginID   string   `json:"pluginId"`
	Generation string   `json:"generation"`
	Features   []string `json:"features,omitempty"`
	// Bus is what the SUBSTRATE implements. The effective set a plugin sees is
	// this intersected with what its transport implements.
	Bus bus.Capabilities `json:"bus"`
	// Identity is who this generation belongs to, with its effective grants.
	Identity bus.Identity `json:"identity"`
	// StateDir is where this plugin may persist state.
	StateDir string `json:"stateDir,omitempty"`
	// HostCallToken is a private, generation-bound bearer capability. The SDK
	// forwards it only to cmd.gateway.* and must never log or persist it. The
	// host delivers negotiation replies only to this plugin's admitted inbox.
	HostCallToken string `json:"hostCallToken,omitempty"`
	// Error is the host's refusal. A refusal arrives as a populated reply
	// rather than a transport error so the reason survives.
	Error string `json:"error,omitempty"`
}

// CheckProtocol is the fail-closed gate between negotiate and Bind.
//
// It checks four things, and every one of them refuses rather than degrades:
// the major version must match exactly, every required feature must be
// advertised, every needed capability must be present in the EFFECTIVE set,
// and the identity the host answered with must be the plugin that asked. A
// plugin that runs anyway on a partial match is a plugin that fails later, in
// production, with a symptom that does not name this handshake.
//
// effective is the negotiated capabilities intersected with the transport's —
// not have.Bus. The distinction matters: a host substrate may be durable while
// the connection this plugin holds cannot reach streams, and it is the
// connection the plugin will actually use.
func CheckProtocol(have HostCapabilities, need ProtocolRequirements, needs []string, effective bus.Capabilities) error {
	if have.Error != "" {
		return Fault{Code: FaultDenied, Op: string(NegotiateSubject), Message: "host refused the handshake: " + have.Error}
	}
	if need.Major == 0 {
		need.Major = ProtocolMajor
	}
	if need.Major != have.Major {
		return Fault{Code: FaultUnsupported, Op: string(NegotiateSubject),
			Message: fmt.Sprintf("plugin protocol major %d is unsupported (host %d)", need.Major, have.Major)}
	}
	for _, feature := range need.Required {
		if strings.TrimSpace(feature) == "" || !slices.Contains(have.Features, feature) {
			return Fault{Code: FaultUnsupported, Op: string(NegotiateSubject),
				Message: fmt.Sprintf("required host feature %q is unsupported", feature)}
		}
	}
	if missing := effective.Missing(needs); len(missing) != 0 {
		return Fault{Code: FaultUnsupported, Op: string(NegotiateSubject),
			Message: "substrate does not provide " + strings.Join(missing, ", ")}
	}
	if have.Generation == "" {
		return Fault{Code: FaultSchema, Op: string(NegotiateSubject), Message: "host returned no generation"}
	}
	if slices.Contains(have.Features, FeatureHostWorkloadAuth) && !validHostCallToken(have.HostCallToken) {
		return Fault{Code: FaultSchema, Op: string(NegotiateSubject), Message: "host workload authentication token is missing or malformed"}
	}
	if !slices.Contains(have.Features, FeatureHostWorkloadAuth) && have.HostCallToken != "" {
		return Fault{Code: FaultSchema, Op: string(NegotiateSubject), Message: "host supplied an unnegotiated workload authentication token"}
	}
	return nil
}

func validHostCallToken(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// checkIdentity is the half of the handshake that is about WHO, kept separate
// because it is checked against the plugin value rather than the manifest.
func checkIdentity(have HostCapabilities, id string) error {
	if have.PluginID != id {
		return Fault{Code: FaultDenied, Op: string(NegotiateSubject),
			Message: fmt.Sprintf("host negotiated identity %q for plugin %q", have.PluginID, id)}
	}
	if have.Identity.PluginID != "" && have.Identity.PluginID != id {
		return Fault{Code: FaultDenied, Op: string(NegotiateSubject),
			Message: fmt.Sprintf("host bound identity %q to plugin %q", have.Identity.PluginID, id)}
	}
	if have.Identity.Generation != "" && have.Identity.Generation != have.Generation {
		return Fault{Code: FaultDenied, Op: string(NegotiateSubject), Message: "host returned conflicting workload generations"}
	}
	return nil
}

// DeclaredNeeds reads the capability list a manifest cannot run without.
func DeclaredNeeds(m Manifest) []string { return m.Needs }

// declaredProtocol is the manifest's protocol block, defaulted to this major.
func declaredProtocol(m plugin.Manifest) ProtocolRequirements {
	if m.Protocol == nil {
		return ProtocolRequirements{Major: ProtocolMajor}
	}
	out := *m.Protocol
	if out.Major == 0 {
		out.Major = ProtocolMajor
	}
	return out
}
