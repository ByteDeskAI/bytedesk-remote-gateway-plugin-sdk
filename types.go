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

	// The plugin contract (ADR 0024). A plugin implements these once; whether
	// the host links it in or spawns it is a deployment decision read from the
	// manifest's spawn field, and the Host handed to Start differs only in
	// whether its calls cross a socket. Serve/ServeContext below is the
	// entry point for the spawned mode.
	Plugin = plugin.Plugin
	Host   = plugin.Host
	Logger = plugin.Logger

	// Optional capabilities, interface-segregated: implement only what you
	// need and the host type-asserts for each.
	HTTPPlugin           = plugin.HTTPPlugin
	HealthContributor    = plugin.HealthContributor
	CommandHandler       = plugin.CommandHandler
	Validator            = plugin.Validator
	Readier              = plugin.Readier
	ActivationChecker    = plugin.ActivationChecker
	Negotiator           = plugin.Negotiator
	RuntimeStatus        = plugin.RuntimeStatus
	RuntimeSnapshot      = plugin.RuntimeSnapshot
	LifecycleOperation   = plugin.LifecycleOperation
	ProtocolRequirements = plugin.ProtocolRequirements
	HostCapabilities     = plugin.HostCapabilities
	Permissions          = plugin.Permissions
	UIContribution       = plugin.UIContribution
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

	ProtocolMajor             = plugin.ProtocolMajor
	FeatureRuntimeSnapshot    = plugin.FeatureRuntimeSnapshot
	FeatureScopedHost         = plugin.FeatureScopedHost
	FeatureActivationCheck    = plugin.FeatureActivationCheck
	FeatureShellContributions = plugin.FeatureShellContributions
	FeatureDocumentPaths      = plugin.FeatureDocumentPaths
	FeatureUIModuleMount      = plugin.FeatureUIModuleMount
	DesiredUnknown            = plugin.DesiredUnknown
	DesiredAbsent             = plugin.DesiredAbsent
	DesiredDisabled           = plugin.DesiredDisabled
	DesiredEnabled            = plugin.DesiredEnabled
	OperationPending          = plugin.OperationPending
	OperationCompleted        = plugin.OperationCompleted
	OperationFailed           = plugin.OperationFailed
	SlotDefaultView           = plugin.SlotDefaultView
	SlotToolbar               = plugin.SlotToolbar
	SlotOverlay               = plugin.SlotOverlay
	SlotBadge                 = plugin.SlotBadge
	SlotSettings              = plugin.SlotSettings
	SlotCommand               = plugin.SlotCommand
)

func CheckProtocol(have HostCapabilities, need ProtocolRequirements) error {
	return plugin.CheckProtocol(have, need)
}

func ValidateDocumentPath(pattern string) error { return plugin.ValidateDocumentPath(pattern) }

func MatchDocumentPath(pattern, escapedPath string) (map[string]string, bool) {
	return plugin.MatchDocumentPath(pattern, escapedPath)
}

func DocumentPathsOverlap(a, b string) (bool, error) {
	return plugin.DocumentPathsOverlap(a, b)
}

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
