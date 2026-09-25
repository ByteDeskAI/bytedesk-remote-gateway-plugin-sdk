// Package aidecision re-exports the common SDK contract without redefining it.
// The host retains all resource, caller, credential and process authority.
package aidecision

import (
	"context"
	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/aidecision"
)

const (
	ContractRevision         = common.ContractRevision
	CommandStart             = common.CommandStart
	CommandRead              = common.CommandRead
	CommandCancel            = common.CommandCancel
	CommandModels            = common.CommandModels
	KindChoice               = common.KindChoice
	KindScore                = common.KindScore
	KindNoul                 = common.KindNoul
	FormatText               = common.FormatText
	FormatJSON               = common.FormatJSON
	MaxQuestions             = common.MaxQuestions
	MaxOptions               = common.MaxOptions
	MaxScoreLevels           = common.MaxScoreLevels
	ProbabilityTolerance     = common.ProbabilityTolerance
	StateQueued              = common.StateQueued
	StateRunning             = common.StateRunning
	StateCompleted           = common.StateCompleted
	StateFailed              = common.StateFailed
	StateCancelled           = common.StateCancelled
	StateExpired             = common.StateExpired
	PurposeEvaluation        = common.PurposeEvaluation
	PurposeRouting           = common.PurposeRouting
	EvaluationTimeoutSeconds = common.EvaluationTimeoutSeconds
	RoutingTimeoutSeconds    = common.RoutingTimeoutSeconds
)

type (
	TextInput                  = common.TextInput
	StructuredValue            = common.StructuredValue
	Option                     = common.Option
	ChoiceQuestion             = common.ChoiceQuestion
	ScoreQuestion              = common.ScoreQuestion
	NoulCriteria               = common.NoulCriteria
	NoulQuestion               = common.NoulQuestion
	Question                   = common.Question
	BatchRequest               = common.BatchRequest
	Probability                = common.Probability
	LegendEntry                = common.LegendEntry
	ChoiceAnswer               = common.ChoiceAnswer
	ScoreAnswer                = common.ScoreAnswer
	NoulAnswer                 = common.NoulAnswer
	Answer                     = common.Answer
	Usage                      = common.Usage
	BatchResult                = common.BatchResult
	ProviderStartRequest       = common.ProviderStartRequest
	ProviderReadRequest        = common.ProviderReadRequest
	ProviderCancelRequest      = common.ProviderCancelRequest
	ProviderModelsRequest      = common.ProviderModelsRequest
	StartResult                = common.StartResult
	ReadRequest                = common.ReadRequest
	ReadResult                 = common.ReadResult
	CancelRequest              = common.CancelRequest
	CancelResult               = common.CancelResult
	Failure                    = common.Failure
	Job                        = common.Job
	ModelsRequest              = common.ModelsRequest
	DecisionModel              = common.DecisionModel
	ModelsResult               = common.ModelsResult
	Payload                    = common.Payload
	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
	Stream[T Payload]          = common.Stream[T]
	Bucket[T Payload]          = common.Bucket[T]
	Service                    = common.Service
)

var (
	Start                 = common.Start
	Read                  = common.Read
	Cancel                = common.Cancel
	Models                = common.Models
	ParseDecimal          = common.ParseDecimal
	DecimalFromJSONNumber = common.DecimalFromJSONNumber
	DecimalFromFloat      = common.DecimalFromFloat
	DeclaredCommands      = common.DeclaredCommands
)

// Call uses this plugin generation's authenticated host bus.
func Call[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], request Req) (Resp, error) {
	if p == nil {
		var zero Resp
		return zero, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "aidecision: nil plugin"}
	}
	return common.Call(ctx, p.Bus(), command, request)
}

func Serve[Req, Resp Payload](ctx context.Context, p *pluginsdk.Base, command Command[Req, Resp], fn func(context.Context, Req, pluginsdk.Caller) (Resp, error)) (pluginsdk.Service, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: command.Name(), Message: "aidecision: nil plugin"}
	}
	return common.Serve(ctx, p.Bus(), command, fn)
}

func Emit[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], value T) error {
	if p == nil {
		return pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "aidecision: nil plugin"}
	}
	return common.Emit(ctx, p.Bus(), event, value)
}

func On[T Payload](ctx context.Context, p *pluginsdk.Base, event Event[T], fn func(context.Context, T, pluginsdk.Caller)) (pluginsdk.Subscription, error) {
	if p == nil {
		return nil, pluginsdk.Fault{Code: pluginsdk.FaultUnavailable, Op: event.Name(), Message: "aidecision: nil plugin"}
	}
	return common.On(ctx, p.Bus(), event, fn)
}
