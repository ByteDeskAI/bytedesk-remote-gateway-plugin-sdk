package pluginsdk

import (
	"context"
	"encoding/json"
	"io"

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

	// Terminal presentation contracts are canonical common-SDK aliases. Hosts
	// select a concrete owner/provider before dispatching; the command identifier
	// is not a process-global CommandHandler registration key.
	PresentationLease            = plugin.PresentationLease
	TerminalBindingContext       = plugin.TerminalBindingContext
	TmuxPresentationContext      = plugin.TmuxPresentationContext
	PresentationTerminal         = plugin.PresentationTerminal
	PresentationRequest          = plugin.PresentationRequest
	PresentationGroup            = plugin.PresentationGroup
	PresentationBadge            = plugin.PresentationBadge
	PresentationItem             = plugin.PresentationItem
	PresentationResult           = plugin.PresentationResult
	TerminalPresentationProvider = plugin.TerminalPresentationProvider
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

	TerminalPresentationPoint        = plugin.TerminalPresentationPoint
	TerminalPresentationInterface    = plugin.TerminalPresentationInterface
	TerminalPresentationCommand      = plugin.TerminalPresentationCommand
	TerminalPresentationBindingRead  = plugin.TerminalPresentationBindingRead
	TerminalPresentationMaxBytes     = plugin.TerminalPresentationMaxBytes
	TerminalPresentationMaxTerminals = plugin.TerminalPresentationMaxTerminals
	TerminalPresentationMaxGroups    = plugin.TerminalPresentationMaxGroups
	TerminalPresentationMaxBadges    = plugin.TerminalPresentationMaxBadges
	TerminalPresentationDeadlineMS   = plugin.TerminalPresentationDeadlineMS
	TerminalBindingNone              = plugin.TerminalBindingNone
	TerminalBindingTmux              = plugin.TerminalBindingTmux
	FreshnessFresh                   = plugin.FreshnessFresh
	FreshnessStale                   = plugin.FreshnessStale
	FreshnessUnknown                 = plugin.FreshnessUnknown
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

func DecodePresentationRequest(r io.Reader) (PresentationRequest, error) {
	return plugin.DecodePresentationRequest(r)
}

func DecodePresentationResult(r io.Reader, request PresentationRequest, current []PresentationTerminal) (PresentationResult, error) {
	return plugin.DecodePresentationResult(r, request, current)
}

func ValidatePresentationRequest(request PresentationRequest) error {
	return plugin.ValidatePresentationRequest(request)
}

func ValidatePresentationResult(request PresentationRequest, current []PresentationTerminal, result PresentationResult) error {
	return plugin.ValidatePresentationResult(request, current, result)
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

// The typed capability layer (common SDK v0.4.0-rc.7). Descriptors carry the
// operation name, contract revision and schema hash, so a mismatched peer is
// rejected before anything decodes. Plugin authors reach it through this
// module; do not import sdk-dependencies directly.
type (
	Descriptor       = plugin.Descriptor
	Registrar        = plugin.Registrar
	Caller           = plugin.Caller
	Fault            = plugin.Fault
	Subscription     = plugin.Subscription
	StatusSubscriber = plugin.StatusSubscriber

	// Payload is the closed set of plugin-owned payload types. A union type
	// set, not a marker method: embedding promotes methods, so a marker is
	// satisfied by a wrapper that adds an unclassified field.
	Payload = plugin.Payload

	Command[Req, Resp Payload] = plugin.Command[Req, Resp]
	Event[T Payload]           = plugin.Event[T]
)

const (
	HeaderSchema     = plugin.HeaderSchema
	HeaderCaller     = plugin.HeaderCaller
	HeaderGeneration = plugin.HeaderGeneration
	HeaderSubject    = plugin.HeaderSubject
	HeaderFault      = plugin.HeaderFault

	FaultDenied      = plugin.FaultDenied
	FaultBudget      = plugin.FaultBudget
	FaultWithdrawn   = plugin.FaultWithdrawn
	FaultSchema      = plugin.FaultSchema
	FaultUnhandled   = plugin.FaultUnhandled
	FaultTimeout     = plugin.FaultTimeout
	FaultUnavailable = plugin.FaultUnavailable
)

// NewDescriptor names one operation at one contract revision and schema hash.
func NewDescriptor(name string, rev uint32, schemaHash string) Descriptor {
	return plugin.NewDescriptor(name, rev, schemaHash)
}

// NewRegistrar returns an empty command registrar.
func NewRegistrar() *Registrar { return plugin.NewRegistrar() }

// NewCommand and NewEvent build descriptors for plugin-owned payload types.
func NewCommand[Req, Resp Payload](name string, rev uint32, schemaHash string) Command[Req, Resp] {
	return plugin.NewCommand[Req, Resp](name, rev, schemaHash)
}

func NewEvent[T Payload](name string, rev uint32, schemaHash string) Event[T] {
	return plugin.NewEvent[T](name, rev, schemaHash)
}

// Invoke, Publish, Observe and HandleRaw are the untyped seam the generated
// per-package wrappers sit on. Prefer Call/Emit/On/Handle.
func Invoke(ctx context.Context, h Host, d Descriptor, req, resp any) error {
	return plugin.Invoke(ctx, h, d, req, resp)
}

func Publish(h Host, d Descriptor, v any) error { return plugin.Publish(h, d, v) }

func Observe(h Host, d Descriptor, fn func(json.RawMessage)) (Subscription, error) {
	return plugin.Observe(h, d, fn)
}

func HandleRaw(r *Registrar, d Descriptor, fn func(context.Context, Caller, json.RawMessage) (json.RawMessage, error)) {
	plugin.HandleRaw(r, d, fn)
}

// Call, Emit, On and Handle are the typed round trips. Generic functions
// cannot be aliased, so these forward rather than re-export.
func Call[Req, Resp Payload](ctx context.Context, h Host, c Command[Req, Resp], req Req) (Resp, error) {
	return plugin.Call(ctx, h, c, req)
}

func Emit[T Payload](h Host, e Event[T], v T) error { return plugin.Emit(h, e, v) }

func On[T Payload](h Host, e Event[T], fn func(T)) (Subscription, error) {
	return plugin.On(h, e, fn)
}

func Handle[Req, Resp Payload](r *Registrar, c Command[Req, Resp], fn func(context.Context, Caller, Req) (Resp, error)) {
	plugin.Handle(r, c, fn)
}

// CallerOf reads the caller identity a host stamped on an envelope.
func CallerOf(env Envelope) Caller { return plugin.CallerOf(env) }
