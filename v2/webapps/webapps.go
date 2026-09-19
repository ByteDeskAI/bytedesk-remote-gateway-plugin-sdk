// Package webapps re-exports the common SDK's host-owned Web Apps runtime
// contract. Gateway plugins import this module only; the Gateway retains
// filesystem, coding-session, process, port, preview, and credential authority.
package webapps

import (
	"context"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	commonplugin "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/webapps"
)

const (
	ContractRevision = common.ContractRevision

	CommandList                = common.CommandList
	CommandCreate              = common.CommandCreate
	CommandCreationEligibility = common.CommandCreationEligibility
	CommandConversationSend    = common.CommandConversationSend
	CommandConversationApprove = common.CommandConversationApprove
	CommandConversationAnswer  = common.CommandConversationAnswer
	CommandRunStop             = common.CommandRunStop
	CommandServicesStart       = common.CommandServicesStart
	CommandServicesStop        = common.CommandServicesStop
	CommandServicesLogs        = common.CommandServicesLogs
	CommandPreviewResolve      = common.CommandPreviewResolve
	CommandPreviewNavigate     = common.CommandPreviewNavigate
	CommandPreviewOpenExternal = common.CommandPreviewOpenExternal
	EventChanged               = common.EventChanged

	MaxPayloadBytes = common.MaxPayloadBytes
	MaxItems        = common.MaxItems
	MaxTextBytes    = common.MaxTextBytes
	MaxAttachments  = common.MaxAttachments
)

type (
	ProjectDirectoryContext = common.ProjectDirectoryContext
	Target                  = common.Target
	ProjectTarget           = common.ProjectTarget
	Attachment              = common.Attachment
	Provider                = common.Provider
	Message                 = common.Message
	Conversation            = common.Conversation
	ToolActivity            = common.ToolActivity
	Run                     = common.Run
	ServiceStatus           = common.ServiceStatus
	PreviewCapabilities     = common.PreviewCapabilities
	Preview                 = common.Preview
	App                     = common.App
	RuntimeEvent            = common.RuntimeEvent
	ListRequest             = common.ListRequest
	ListResult              = common.ListResult

	CreationEligibilityRequest = common.CreationEligibilityRequest
	CreationEligibilityResult  = common.CreationEligibilityResult
	CreateRequest              = common.CreateRequest
	CreateResult               = common.CreateResult

	ConversationSendRequest    = common.ConversationSendRequest
	ConversationSendResult     = common.ConversationSendResult
	ConversationApproveRequest = common.ConversationApproveRequest
	ConversationApproveResult  = common.ConversationApproveResult
	ConversationAnswerRequest  = common.ConversationAnswerRequest
	ConversationAnswerResult   = common.ConversationAnswerResult
	RunStopRequest             = common.RunStopRequest
	RunStopResult              = common.RunStopResult
	ServicesStartRequest       = common.ServicesStartRequest
	ServicesStartResult        = common.ServicesStartResult
	ServicesStopRequest        = common.ServicesStopRequest
	ServicesStopResult         = common.ServicesStopResult
	ServicesLogsRequest        = common.ServicesLogsRequest
	LogEntry                   = common.LogEntry
	ServicesLogsResult         = common.ServicesLogsResult
	PreviewResolveRequest      = common.PreviewResolveRequest
	PreviewResolveResult       = common.PreviewResolveResult
	PreviewNavigateRequest     = common.PreviewNavigateRequest
	PreviewNavigateResult      = common.PreviewNavigateResult
	PreviewOpenExternalRequest = common.PreviewOpenExternalRequest
	PreviewOpenExternalResult  = common.PreviewOpenExternalResult

	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
	Stream[T Payload]          = common.Stream[T]
)

var (
	List                       = common.List
	Create                     = common.Create
	CheckCreationEligibility   = common.CheckCreationEligibility
	SendConversationMessage    = common.SendConversationMessage
	ApproveConversationRequest = common.ApproveConversationRequest
	AnswerConversationRequest  = common.AnswerConversationRequest
	StopRun                    = common.StopRun
	StartServices              = common.StartServices
	StopServices               = common.StopServices
	ReadServiceLogs            = common.ReadServiceLogs
	ResolvePreview             = common.ResolvePreview
	NavigatePreview            = common.NavigatePreview
	OpenPreviewExternal        = common.OpenPreviewExternal
	Changed                    = common.Changed
	Events                     = common.Events
)

// Call invokes a Web Apps command through this plugin generation's bound bus.
func Call[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if p == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "webapps: nil plugin"}
	}
	return common.Call(ctx, p.Bus(), command, request)
}

func Emit[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], value T) error {
	if p == nil {
		return pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "webapps: nil plugin"}
	}
	return common.Emit(ctx, p.Bus(), event, value)
}

func On[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], fn func(context.Context, T, pluginsdk.Caller)) (pluginsdk.Subscription, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "webapps: nil plugin"}
	}
	return common.On(ctx, p.Bus(), event, fn)
}

func OpenStream[T Payload](p *pluginsdk.Base, stream Stream[T]) commonplugin.TypedStream[T] {
	if p == nil {
		var zero commonplugin.TypedStream[T]
		return zero
	}
	return common.OpenStream(p.Bus(), stream)
}
