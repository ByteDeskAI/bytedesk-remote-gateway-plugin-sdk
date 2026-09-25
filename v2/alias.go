package pluginsdk

import (
	"context"
	"net/http"

	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/transport/natsconn"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/semver"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

// The v2 contract, re-exported so a plugin author imports one module.
//
// This list is deliberately SHORT compared with v1's. v1 aliased every typed
// contract in the common SDK — tmux, desktop applications, terminal
// presentation — which meant every contract revision was also a release of this
// module. In v2 those are generated per package by contractgen and imported
// where they are used; what lives here is the contract every plugin touches.
type (
	Bus          = bus.Bus
	Subject      = bus.Subject
	Pattern      = bus.Pattern
	Headers      = bus.Headers
	Msg          = bus.Msg
	Handler      = bus.Handler
	Subscription = bus.Subscription
	Fault        = bus.Fault
	Capabilities = bus.Capabilities
	Identity     = bus.Identity
	Grants       = bus.Grants
	GrantKind    = bus.GrantKind
	Role         = bus.Role
	Lease        = bus.Lease
	Caller       = bus.Caller

	Service      = bus.Service
	ServiceSpec  = bus.ServiceSpec
	EndpointSpec = bus.EndpointSpec
	ServiceInfo  = bus.ServiceInfo
	Streams      = bus.Streams
	KV           = bus.KV
	Bucket       = bus.Bucket
	Objects      = bus.Objects
	Scheduler    = bus.Scheduler
	Trace        = bus.Trace

	Plugin            = plugin.Plugin
	Bound             = plugin.Bound
	Base              = plugin.Base
	Binding           = plugin.Binding
	Manifest          = plugin.Manifest
	ConsentCapability = plugin.ConsentCapability
	Logger            = plugin.Logger
	Profiler          = plugin.Profiler

	HTTPPlugin        = plugin.HTTPPlugin
	HealthContributor = plugin.HealthContributor
	Validator         = plugin.Validator
	Readier           = plugin.Readier
	ActivationChecker = plugin.ActivationChecker
	Draining          = plugin.Draining
	DataVersioned     = plugin.DataVersioned
	DataVersion       = plugin.DataVersion

	ProtocolRequirements = plugin.ProtocolRequirements
	Permissions          = plugin.Permissions
	ServiceDecl          = plugin.ServiceDecl
	EndpointDecl         = plugin.EndpointDecl
	StreamDecl           = plugin.StreamDecl
	KVDecl               = plugin.KVDecl
	ObjectDecl           = plugin.ObjectDecl
	ManifestIdentity     = plugin.ManifestIdentity
	Images               = plugin.Images
	Support              = plugin.Support
	Publisher            = plugin.Publisher
	StaticMount          = plugin.StaticMount
	Point                = plugin.Point
	Registrar            = plugin.Registrar
	Descriptor           = plugin.Descriptor

	ProjectViewContribution                  = plugin.ProjectViewContribution
	DirectoryContextActionContribution       = plugin.DirectoryContextActionContribution
	ProjectDirectoryContext                  = plugin.ProjectDirectoryContext
	DirectoryContextActionEligibilityRequest = plugin.DirectoryContextActionEligibilityRequest
	DirectoryContextActionEligibilityResult  = plugin.DirectoryContextActionEligibilityResult
	DirectoryContextActionWizardContext      = plugin.DirectoryContextActionWizardContext
)

const (
	ProtocolMajor                            = plugin.ProtocolMajor
	DirectoryContextActionEligibilityCommand = plugin.DirectoryContextActionEligibilityCommand

	FaultDenied       = bus.FaultDenied
	FaultBudget       = bus.FaultBudget
	FaultWithdrawn    = bus.FaultWithdrawn
	FaultSchema       = bus.FaultSchema
	FaultUnhandled    = bus.FaultUnhandled
	FaultTimeout      = bus.FaultTimeout
	FaultUnavailable  = bus.FaultUnavailable
	FaultNoResponders = bus.FaultNoResponders
	FaultUnsupported  = bus.FaultUnsupported
	FaultConflict     = bus.FaultConflict
	FaultNotFound     = bus.FaultNotFound
	FaultSlowConsumer = bus.FaultSlowConsumer

	HeaderSchema      = bus.HeaderSchema
	HeaderCaller      = bus.HeaderCaller
	HeaderGeneration  = bus.HeaderGeneration
	HeaderSubject     = bus.HeaderSubject
	HeaderFault       = bus.HeaderFault
	HeaderCorrelation = bus.HeaderCorrelation

	// HeaderSubjectLease is a different thing from HeaderSubject above: that
	// one is the reserved "bd-subject" the substrate itself would stamp from a
	// connection's own credential (not yet built for v2 -- nothing does that
	// today). This is an ordinary, unreserved header a plugin sets itself,
	// carrying an opaque lease id it can only ever echo from its own inbound
	// request, never manufacture. See transport/natsconn.ContextForRequest.
	HeaderSubjectLease = natsconn.HeaderSubjectLease
	// HeaderHostCallToken is transport-managed workload authentication. It does
	// not replace a subject lease or grant permission to perform an operation.
	HeaderHostCallToken = natsconn.HeaderHostCallToken

	GrantPublish   = bus.GrantPublish
	GrantSubscribe = bus.GrantSubscribe
	GrantRequest   = bus.GrantRequest

	TargetGateway = plugin.TargetGateway
	TargetVault   = plugin.TargetVault
	RoleSystem    = plugin.RoleSystem
	RoleExtension = plugin.RoleExtension

	KindBuiltin = plugin.KindBuiltin
	KindProcess = plugin.KindProcess
	KindUI      = plugin.KindUI
	KindFamily  = plugin.KindFamily
)

// Bind installs a generation's binding on a plugin's embedded Base. The host
// calls it in linked mode; ServePlugin calls it in spawned mode.
func Bind(p Bound, b Binding) error { return plugin.Bind(p, b) }

// NewRegistrar returns a registrar that refuses any endpoint outside serves.
func NewRegistrar(serves []Pattern) *Registrar { return plugin.NewRegistrar(serves) }

// OwnNamespace is the grant a plugin has by being that plugin. It is implicit
// and a manifest that lists any of it is refused.
func OwnNamespace(id string) Grants { return plugin.OwnNamespace(id) }

// GrantsDigest is what an operator's consent is keyed by. plugin-sdk pack
// prints it, and every dev-grants entry re-approves once when it changes.
func GrantsDigest(m Manifest) string { return plugin.GrantsDigest(m) }

// ConsentCapabilities is the install-sheet list for this manifest. The
// sentences are the host catalog, the same ids CapabilityEnabled checks.
func ConsentCapabilities(m Manifest) []ConsentCapability { return plugin.ConsentCapabilities(m) }

// CapabilityEnabled reports whether the host grant turns id on.
func CapabilityEnabled(granted []string, id string) bool {
	return plugin.CapabilityEnabled(granted, id)
}

const (
	CapabilityCredentialSecret  = plugin.CapabilityCredentialSecret
	CapabilityEgressProvider    = plugin.CapabilityEgressProvider
	CapabilityProcessSupervised = plugin.CapabilityProcessSupervised
	CapabilityIngressPublish    = plugin.CapabilityIngressPublish
)

// CallerOf reads the caller identity the substrate stamped on a message.
func CallerOf(m *Msg) Caller { return bus.CallerOf(m) }

// ParseSubject and ParsePattern are the one grammar the validator, Grants.Can
// and every substrate share.
func ParseSubject(s string) (Subject, error) { return bus.ParseSubject(s) }
func ParsePattern(s string) (Pattern, error) { return bus.ParsePattern(s) }

// ParseManifest decodes plugin.json using the v2 contract.
func ParseManifest(raw []byte) (Manifest, error) { return plugin.ParseManifest(raw) }

// CoreVersionAtLeast is the shared minCoreVersion compare.
func CoreVersionAtLeast(have, need string) bool { return semver.AtLeast(have, need) }

// ContextForRequest and SubjectLeaseFromContext carry an operator subject
// lease from a plugin's own inbound HTTP request into the context it passes
// to a Bus.Request call, which then attaches it to the outbound message. A
// plugin reads its inbound request's HeaderSubjectLease itself and calls
// ContextForRequest(r) before making the Request call; nothing does this
// automatically, unlike the v1 SDK's HTTP middleware, because v2 does not yet
// give a plugin's own HTTP surface a comparable middleware seam.
func ContextForRequest(r *http.Request) context.Context { return natsconn.ContextForRequest(r) }
func SubjectLeaseFromContext(ctx context.Context) (string, error) {
	return natsconn.SubjectLeaseFromContext(ctx)
}
