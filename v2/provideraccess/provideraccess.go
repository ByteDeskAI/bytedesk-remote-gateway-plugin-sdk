// Package provideraccess re-exports the common SDK contract without redefining it.
// The host retains all resource, caller, credential and process authority.
package provideraccess

import (
	"context"
	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/provideraccess"
)

const (
	ContractRevision    = common.ContractRevision
	CommandCredential   = common.CommandCredential
	CommandRevoke       = common.CommandRevoke
	CommandEgressStart  = common.CommandEgressStart
	CommandEgressRead   = common.CommandEgressRead
	CommandEgressCancel = common.CommandEgressCancel
	OperationEvaluate   = common.OperationEvaluate
	OperationModels     = common.OperationModels
	MaxTimeoutSeconds   = common.MaxTimeoutSeconds
)

type (
	CredentialRequest          = common.CredentialRequest
	CredentialHandle           = common.CredentialHandle
	CredentialResult           = common.CredentialResult
	RevokeRequest              = common.RevokeRequest
	RevokeResult               = common.RevokeResult
	EgressRequest              = common.EgressRequest
	EgressResult               = common.EgressResult
	EgressStartResult          = common.EgressStartResult
	EgressReadRequest          = common.EgressReadRequest
	EgressReadResult           = common.EgressReadResult
	EgressCancelRequest        = common.EgressCancelRequest
	EgressCancelResult         = common.EgressCancelResult
	EgressFailure              = common.EgressFailure
	EgressJob                  = common.EgressJob
	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
	Stream[T Payload]          = common.Stream[T]
	Bucket[T Payload]          = common.Bucket[T]
	Service                    = common.Service
)

var (
	Credential   = common.Credential
	Revoke       = common.Revoke
	EgressStart  = common.EgressStart
	EgressRead   = common.EgressRead
	EgressCancel = common.EgressCancel
)

// Call uses this plugin generation's authenticated host bus.
func Call[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if p == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "provideraccess: nil plugin"}
	}
	return common.Call(ctx, p.Bus(), command, request)
}

func Serve[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], fn func(context.Context, Req, pluginsdk.Caller) (Resp, error)) (pluginsdk.Service, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "provideraccess: nil plugin"}
	}
	return common.Serve(ctx, p.Bus(), command, fn)
}

func Emit[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], value T) error {
	if p == nil {
		return pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "provideraccess: nil plugin"}
	}
	return common.Emit(ctx, p.Bus(), event, value)
}

func On[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], fn func(context.Context, T, pluginsdk.Caller)) (pluginsdk.Subscription, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "provideraccess: nil plugin"}
	}
	return common.On(ctx, p.Bus(), event, fn)
}
