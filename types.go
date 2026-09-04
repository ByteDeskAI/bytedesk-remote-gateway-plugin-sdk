package pluginsdk

import (
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/semver"
)

// Common types inherited from sdk-dependencies. Plugin authors should
// import this module only; do not redefine Manifest.
type (
	Manifest     = plugin.Manifest
	NavItem      = plugin.NavItem
	PanelSpec    = plugin.PanelSpec
	LauncherSpec = plugin.LauncherSpec
	Pricing      = plugin.Pricing
	Publisher    = plugin.Publisher
	Requirement  = plugin.Requirement
	Envelope     = bus.Envelope

	// Composition and extension points. The host resolves all of these: it
	// expands a family and starts the matching member, and it mediates every
	// provider registration. A plugin never loads, execs or proxies another.
	When           = plugin.When
	Family         = plugin.Family
	FamilyMember   = plugin.FamilyMember
	ExtensionPoint = plugin.ExtensionPoint
	Provider       = plugin.Provider
)

const (
	TargetGateway = plugin.TargetGateway
	TargetVault   = plugin.TargetVault
	RoleSystem    = plugin.RoleSystem
	RoleExtension = plugin.RoleExtension

	// Lifecycle states a host announces on the bus as a plugin moves through
	// them. Subscribe to learn that you are ready, that a peer arrived, or that
	// a dependency went away.
	StateDiscovered  = plugin.StateDiscovered
	StateValidated   = plugin.StateValidated
	StateStarting    = plugin.StateStarting
	StateRunning     = plugin.StateRunning
	StateEnabled     = plugin.StateEnabled
	StateDegraded    = plugin.StateDegraded
	StateStopping    = plugin.StateStopping
	StateDisabled    = plugin.StateDisabled
	StateQuarantined = plugin.StateQuarantined
	StateFailed      = plugin.StateFailed
	StateExited      = plugin.StateExited

	EventExtensionRegistered = plugin.EventExtensionRegistered
	EventExtensionRevoked    = plugin.EventExtensionRevoked
)

// LifecycleEvent is the bus type published when a plugin enters state, e.g.
// LifecycleEvent(StateRunning) == "event.plugin.running".
func LifecycleEvent(state string) string { return plugin.LifecycleEvent(state) }

// ParseManifest decodes plugin.json using the common contract.
func ParseManifest(raw []byte) (Manifest, error) {
	return plugin.ParseManifest(raw)
}

// CoreVersionAtLeast is the shared minCoreVersion compare.
func CoreVersionAtLeast(have, need string) bool {
	return semver.AtLeast(have, need)
}
