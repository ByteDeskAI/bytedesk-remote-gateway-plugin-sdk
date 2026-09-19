// Package sessioncontext re-exports the common SDK's lease-scoped,
// host-owned interaction contract. Gateway plugins import this module only;
// the Gateway implements the commands and retains terminal, project, process,
// route, and proxy authority.
package sessioncontext

import (
	"context"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/sessioncontext"
)

const (
	ContractRevision = common.ContractRevision

	CommandOpen    = common.CommandOpen
	CommandRefresh = common.CommandRefresh
	CommandAction  = common.CommandAction

	PurposeTaskDashboard = common.PurposeTaskDashboard
	StateReady           = common.StateReady
	StateStarting        = common.StateStarting
	StateUnavailable     = common.StateUnavailable

	MaxBytes        = common.MaxBytes
	MaxActions      = common.MaxActions
	MaxActionInputs = common.MaxActionInputs
)

type (
	OpenRequest   = common.OpenRequest
	Context       = common.Context
	AllowedAction = common.AllowedAction
	OpenResult    = common.OpenResult

	RefreshRequest = common.RefreshRequest
	RefreshResult  = common.RefreshResult

	ActionInput   = common.ActionInput
	ActionRequest = common.ActionRequest
	ActionResult  = common.ActionResult

	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
)

var (
	Open    = common.Open
	Refresh = common.Refresh
	Action  = common.Action
)

// Call invokes a host-owned session-context command through the generation's
// bound bus. The transport stamps the workload identity and subject lease;
// callers cannot manufacture either from this API.
func Call[Req, Resp Payload](ctx context.Context, plugin *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if plugin == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "sessioncontext: nil plugin"}
	}
	return common.Call(ctx, plugin.Bus(), command, request)
}
