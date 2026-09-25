// Package payloads re-exports the common SDK contract without redefining it.
// The host retains all resource, caller, credential and process authority.
package payloads

import (
	"context"
	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/payloads"
)

const (
	ContractRevision        = common.ContractRevision
	CommandCreate           = common.CommandCreate
	CommandAppend           = common.CommandAppend
	CommandCommit           = common.CommandCommit
	CommandRead             = common.CommandRead
	CommandRevoke           = common.CommandRevoke
	MaxAssembledBytes       = common.MaxAssembledBytes
	MaxChunkBytes           = common.MaxChunkBytes
	MaxInlineTextBytes      = common.MaxInlineTextBytes
	DefaultTTLSeconds       = common.DefaultTTLSeconds
	MaxTTLSeconds           = common.MaxTTLSeconds
	PurposeDecision         = common.PurposeDecision
	PurposeProviderRequest  = common.PurposeProviderRequest
	PurposeProviderResponse = common.PurposeProviderResponse
	PurposeCodingPrompt     = common.PurposeCodingPrompt
	StateUploading          = common.StateUploading
	StateCommitted          = common.StateCommitted
)

type (
	TextInput                  = common.TextInput
	CreateRequest              = common.CreateRequest
	Handle                     = common.Handle
	CreateResult               = common.CreateResult
	AppendRequest              = common.AppendRequest
	AppendResult               = common.AppendResult
	CommitRequest              = common.CommitRequest
	CommitResult               = common.CommitResult
	ReadRequest                = common.ReadRequest
	ReadResult                 = common.ReadResult
	RevokeRequest              = common.RevokeRequest
	RevokeResult               = common.RevokeResult
	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
	Stream[T Payload]          = common.Stream[T]
	Bucket[T Payload]          = common.Bucket[T]
	Service                    = common.Service
)

var (
	Create = common.Create
	Append = common.Append
	Commit = common.Commit
	Read   = common.Read
	Revoke = common.Revoke
)

// Call uses this plugin generation's authenticated host bus.
func Call[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if p == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "payloads: nil plugin"}
	}
	return common.Call(ctx, p.Bus(), command, request)
}

func Serve[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], fn func(context.Context, Req, pluginsdk.Caller) (Resp, error)) (pluginsdk.Service, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "payloads: nil plugin"}
	}
	return common.Serve(ctx, p.Bus(), command, fn)
}

func Emit[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], value T) error {
	if p == nil {
		return pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "payloads: nil plugin"}
	}
	return common.Emit(ctx, p.Bus(), event, value)
}

func On[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], fn func(context.Context, T, pluginsdk.Caller)) (pluginsdk.Subscription, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "payloads: nil plugin"}
	}
	return common.On(ctx, p.Bus(), event, fn)
}
