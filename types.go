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

	// ManifestConfig describes settings, unlike Config, which configures Serve.
	ManifestConfig          = plugin.Config
	ConfigSection           = plugin.ConfigSection
	ConfigField             = plugin.ConfigField
	ContributionEligibility = plugin.ContributionEligibility
	ContributionRole        = plugin.ContributionRole

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

	DesktopSessionStatus                   = plugin.DesktopSessionStatus
	DesktopApplication                     = plugin.DesktopApplication
	DesktopApplicationWindow               = plugin.DesktopApplicationWindow
	DesktopApplicationSession              = plugin.DesktopApplicationSession
	DesktopApplicationsStatusRequest       = plugin.DesktopApplicationsStatusRequest
	DesktopApplicationsStatusResult        = plugin.DesktopApplicationsStatusResult
	DesktopApplicationsScanRequest         = plugin.DesktopApplicationsScanRequest
	DesktopApplicationsScanResult          = plugin.DesktopApplicationsScanResult
	DesktopApplicationsScanV2Request       = plugin.DesktopApplicationsScanV2Request
	DesktopApplicationsScanV2Result        = plugin.DesktopApplicationsScanV2Result
	DesktopApplicationsRegisterRequest     = plugin.DesktopApplicationsRegisterRequest
	DesktopApplicationsRegisterResult      = plugin.DesktopApplicationsRegisterResult
	DesktopApplicationsOpenRequest         = plugin.DesktopApplicationsOpenRequest
	DesktopApplicationsOpenResult          = plugin.DesktopApplicationsOpenResult
	DesktopApplicationsRefreshRequest      = plugin.DesktopApplicationsRefreshRequest
	DesktopApplicationsRefreshResult       = plugin.DesktopApplicationsRefreshResult
	DesktopApplicationsViewerTicketRequest = plugin.DesktopApplicationsViewerTicketRequest
	DesktopApplicationsViewerTicketResult  = plugin.DesktopApplicationsViewerTicketResult
	DesktopApplicationsQuitRequest         = plugin.DesktopApplicationsQuitRequest
	DesktopApplicationsQuitResult          = plugin.DesktopApplicationsQuitResult
)

const (
	TargetGateway = plugin.TargetGateway
	TargetVault   = plugin.TargetVault
	RoleSystem    = plugin.RoleSystem
	RoleExtension = plugin.RoleExtension

	ConfigKindBool       = plugin.ConfigKindBool
	ConfigKindInt        = plugin.ConfigKindInt
	ConfigKindString     = plugin.ConfigKindString
	ConfigKindStringList = plugin.ConfigKindStringList
	ConfigKindEnum       = plugin.ConfigKindEnum
	ConfigKindSecret     = plugin.ConfigKindSecret
	ConfigKindProvider   = plugin.ConfigKindProvider

	ContributionInstalledAllowed = plugin.ContributionInstalledAllowed
	ContributionConsentRequired  = plugin.ContributionConsentRequired
	ContributionCompiledOnly     = plugin.ContributionCompiledOnly

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
	FeatureLifecycleHooks     = plugin.FeatureLifecycleHooks
	HookActivationCheck       = plugin.HookActivationCheck
	HookReady                 = plugin.HookReady
	HookHealth                = plugin.HookHealth
	LifecycleHookCommand      = plugin.LifecycleHookCommand
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
	SlotMainNavigation        = plugin.SlotMainNavigation
	SlotSubNavigation         = plugin.SlotSubNavigation
	SlotPrimaryAction         = plugin.SlotPrimaryAction
	SlotSecondaryActions      = plugin.SlotSecondaryActions
	SlotStatusIndicator       = plugin.SlotStatusIndicator
	SlotObjectActions         = plugin.SlotObjectActions
	SlotLauncher              = plugin.SlotLauncher
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

	DesktopApplicationsService             = plugin.DesktopApplicationsService
	DesktopApplicationsContractRevision    = plugin.DesktopApplicationsContractRevision
	DesktopApplicationsStatusCommand       = plugin.DesktopApplicationsStatusCommand
	DesktopApplicationsScanCommand         = plugin.DesktopApplicationsScanCommand
	DesktopApplicationsScanV2Command       = plugin.DesktopApplicationsScanV2Command
	DesktopApplicationsRegisterCommand     = plugin.DesktopApplicationsRegisterCommand
	DesktopApplicationsOpenCommand         = plugin.DesktopApplicationsOpenCommand
	DesktopApplicationsRefreshCommand      = plugin.DesktopApplicationsRefreshCommand
	DesktopApplicationsViewerTicketCommand = plugin.DesktopApplicationsViewerTicketCommand
	DesktopApplicationsQuitCommand         = plugin.DesktopApplicationsQuitCommand
	DesktopApplicationsMaxBytes            = plugin.DesktopApplicationsMaxBytes
	DesktopApplicationsScanV2MaxBytes      = plugin.DesktopApplicationsScanV2MaxBytes
	DesktopApplicationsScanV2DefaultLimit  = plugin.DesktopApplicationsScanV2DefaultLimit
	DesktopApplicationsScanV2MaxLimit      = plugin.DesktopApplicationsScanV2MaxLimit
	DesktopApplicationKindDesktop          = plugin.DesktopApplicationKindDesktop
	DesktopApplicationKindBundle           = plugin.DesktopApplicationKindBundle
	DesktopApplicationKindBinary           = plugin.DesktopApplicationKindBinary
	DesktopApplicationReady                = plugin.DesktopApplicationReady
	DesktopApplicationMissing              = plugin.DesktopApplicationMissing
	DesktopApplicationInvalid              = plugin.DesktopApplicationInvalid
	DesktopApplicationsScanV2Scanning      = plugin.DesktopApplicationsScanV2Scanning
	DesktopApplicationsScanV2Complete      = plugin.DesktopApplicationsScanV2Complete
	DesktopApplicationsScanV2Failed        = plugin.DesktopApplicationsScanV2Failed
	DesktopSessionStarting                 = plugin.DesktopSessionStarting
	DesktopSessionReady                    = plugin.DesktopSessionReady
	DesktopSessionChooseWindow             = plugin.DesktopSessionChooseWindow
	DesktopSessionUnavailable              = plugin.DesktopSessionUnavailable
)

var (
	CmdDesktopApplicationsStatus       = plugin.CmdDesktopApplicationsStatus
	CmdDesktopApplicationsScan         = plugin.CmdDesktopApplicationsScan
	CmdDesktopApplicationsScanV2       = plugin.CmdDesktopApplicationsScanV2
	CmdDesktopApplicationsRegister     = plugin.CmdDesktopApplicationsRegister
	CmdDesktopApplicationsOpen         = plugin.CmdDesktopApplicationsOpen
	CmdDesktopApplicationsRefresh      = plugin.CmdDesktopApplicationsRefresh
	CmdDesktopApplicationsViewerTicket = plugin.CmdDesktopApplicationsViewerTicket
	CmdDesktopApplicationsQuit         = plugin.CmdDesktopApplicationsQuit
)

func CheckProtocol(have HostCapabilities, need ProtocolRequirements) error {
	return plugin.CheckProtocol(have, need)
}

// ConfigFieldsFromStruct derives settings fields using the common SDK's tags
// and validation. It does not read or persist configuration values.
func ConfigFieldsFromStruct(v any) ([]ConfigField, error) {
	return plugin.ConfigFieldsFromStruct(v)
}

// ContributionRoles returns the canonical role vocabulary and admission policy
// in a fresh slice that callers may inspect without changing shared policy.
func ContributionRoles() []ContributionRole { return plugin.ContributionRoles() }

// ContributionRoleFor looks up an exact canonical role.
func ContributionRoleFor(slot string) (ContributionRole, bool) {
	return plugin.ContributionRoleFor(slot)
}

// ContributionRoleAllowed checks admission, not execution permissions. Inputs
// must be host-owned: compiledIn is verified build provenance, not Manifest.Role
// or a signature; pointConsent is an explicit grant for this plugin and role,
// not broad module trust. Hosts must re-evaluate on revocation. Unknown roles
// fail closed even for compiled-in code.
func ContributionRoleAllowed(slot string, compiledIn, pointConsent bool) bool {
	return plugin.ContributionRoleAllowed(slot, compiledIn, pointConsent)
}

// DeclaredHooks lists the lifecycle hooks p implements, by local assertion.
func DeclaredHooks(p any) []string { return plugin.DeclaredHooks(p) }

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

// The typed capability layer (common SDK v0.4.0-rc.8). Descriptors carry the
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

// Kit is the admitted Host plus the capabilities already negotiated for it (common
// SDK TM-266). Hold it as a field; Can reads the cached grants and the host still
// enforces every call.
type (
	Kit       = plugin.Kit
	GrantKind = plugin.GrantKind
)

const (
	GrantPublish   = plugin.GrantPublish
	GrantSubscribe = plugin.GrantSubscribe
	GrantRequest   = plugin.GrantRequest
)

// NewKitFrom builds a Kit from capabilities already negotiated. It does not negotiate.
func NewKitFrom(h Host, caps HostCapabilities) *Kit { return plugin.NewKitFrom(h, caps) }

// Optional interfaces that sit beside Host and Plugin (common SDK TM-268).
// Host's method set is pinned, so these arrive alongside it: type-assert for
// what you want, and an older host simply does not implement it.
//
// ObservableRegistrar makes a refused Subscribe or Every visible, Draining runs
// before the host revokes the generation and may not veto, and DataVersioned
// records which version last wrote this plugin's state dir.
type (
	ObservableRegistrar = plugin.ObservableRegistrar
	Draining            = plugin.Draining
	DataVersioned       = plugin.DataVersioned
	DataVersion         = plugin.DataVersion
)

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
