// Package codingsessions re-exports the common SDK contract without redefining it.
// The host retains all resource, caller, credential and process authority.
package codingsessions

import (
	"context"
	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/codingsessions"
)

const (
	CommandPreview        = common.CommandPreview
	CommandPreviewRead    = common.CommandPreviewRead
	CommandPreviewCancel  = common.CommandPreviewCancel
	ContractRevision      = common.ContractRevision
	CommandCatalog        = common.CommandCatalog
	CommandCreate         = common.CommandCreate
	CommandRead           = common.CommandRead
	CommandRecover        = common.CommandRecover
	CommandList           = common.CommandList
	CommandPrompt         = common.CommandPrompt
	CommandStop           = common.CommandStop
	CommandEnd            = common.CommandEnd
	CommandComplete       = common.CommandComplete
	CommandNewTask        = common.CommandNewTask
	CommandPreferences    = common.CommandPreferences
	CommandApprove        = common.CommandApprove
	CommandEvents         = common.CommandEvents
	CommandOpenSurface    = common.CommandOpenSurface
	EventChanged          = common.EventChanged
	PolicyBalanced        = common.PolicyBalanced
	PolicyEconomy         = common.PolicyEconomy
	PolicyFastest         = common.PolicyFastest
	PolicyMaximumQuality  = common.PolicyMaximumQuality
	PermissionAsk         = common.PermissionAsk
	PermissionAutoEdit    = common.PermissionAutoEdit
	PermissionFullAccess  = common.PermissionFullAccess
	StatePending          = common.StatePending
	StateRouting          = common.StateRouting
	StateQueued           = common.StateQueued
	StateActive           = common.StateActive
	StateRecovering       = common.StateRecovering
	StateRecoveryRequired = common.StateRecoveryRequired
	StateCompleted        = common.StateCompleted
	StateEnded            = common.StateEnded
	StateFailed           = common.StateFailed
)

type (
	PreviewRequest             = common.PreviewRequest
	PreviewResult              = common.PreviewResult
	PreviewJob                 = common.PreviewJob
	PreviewJobResult           = common.PreviewJobResult
	PreviewJobRequest          = common.PreviewJobRequest
	TextInput                  = common.TextInput
	ConfigChoice               = common.ConfigChoice
	ConfigOption               = common.ConfigOption
	ConfigValue                = common.ConfigValue
	Model                      = common.Model
	Capabilities               = common.Capabilities
	Provider                   = common.Provider
	CatalogRequest             = common.CatalogRequest
	CatalogResult              = common.CatalogResult
	Overrides                  = common.Overrides
	Preferences                = common.Preferences
	Route                      = common.Route
	Session                    = common.Session
	CreateRequest              = common.CreateRequest
	SessionResult              = common.SessionResult
	SessionRequest             = common.SessionRequest
	ListRequest                = common.ListRequest
	ListResult                 = common.ListResult
	PromptRequest              = common.PromptRequest
	PromptResult               = common.PromptResult
	StopRequest                = common.StopRequest
	NewTaskRequest             = common.NewTaskRequest
	PreferencesRequest         = common.PreferencesRequest
	ApprovalOption             = common.ApprovalOption
	Approval                   = common.Approval
	ApproveRequest             = common.ApproveRequest
	Message                    = common.Message
	ToolUpdate                 = common.ToolUpdate
	StateUpdate                = common.StateUpdate
	ConfigUpdate               = common.ConfigUpdate
	Failure                    = common.Failure
	SessionEvent               = common.SessionEvent
	ChangedEvent               = common.ChangedEvent
	EventsRequest              = common.EventsRequest
	EventsResult               = common.EventsResult
	OpenSurfaceRequest         = common.OpenSurfaceRequest
	OpenSurfaceResult          = common.OpenSurfaceResult
	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
	Stream[T Payload]          = common.Stream[T]
	Bucket[T Payload]          = common.Bucket[T]
	Service                    = common.Service
)

var (
	Catalog           = common.Catalog
	Preview           = common.Preview
	PreviewRead       = common.PreviewRead
	PreviewCancel     = common.PreviewCancel
	Create            = common.Create
	Read              = common.Read
	Recover           = common.Recover
	List              = common.List
	Prompt            = common.Prompt
	Stop              = common.Stop
	End               = common.End
	Complete          = common.Complete
	NewTask           = common.NewTask
	UpdatePreferences = common.UpdatePreferences
	Approve           = common.Approve
	Events            = common.Events
	OpenSurface       = common.OpenSurface
	Changed           = common.Changed
)

// Call uses this plugin generation's authenticated host bus.
func Call[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if p == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "codingsessions: nil plugin"}
	}
	return common.Call(ctx, p.Bus(), command, request)
}

func Serve[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], fn func(context.Context, Req, pluginsdk.Caller) (Resp, error)) (pluginsdk.Service, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "codingsessions: nil plugin"}
	}
	return common.Serve(ctx, p.Bus(), command, fn)
}

func Emit[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], value T) error {
	if p == nil {
		return pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "codingsessions: nil plugin"}
	}
	return common.Emit(ctx, p.Bus(), event, value)
}

func On[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], fn func(context.Context, T, pluginsdk.Caller)) (pluginsdk.Subscription, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "codingsessions: nil plugin"}
	}
	return common.On(ctx, p.Bus(), event, fn)
}
