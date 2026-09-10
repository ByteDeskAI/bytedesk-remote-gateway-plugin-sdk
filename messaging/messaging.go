// Package messaging re-exports the common SDK's agent-messaging contract so a
// gateway plugin imports this module only. Every type here is an alias of
// github.com/ByteDeskAI/bytedesk-sdk-dependencies/messaging, so values cross
// the boundary unchanged; the four wrappers exist only because Go cannot alias
// a generic function.
//
// The contract is owned upstream. Do not add a field, a fault code or an
// operation here — change it in the common SDK, tag a release, and bump the
// require in go.mod.
package messaging

import (
	"context"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/messaging"
)

const (
	ServiceID        = common.ServiceID
	ContractRevision = common.ContractRevision

	CommandEndpointUpsert   = common.CommandEndpointUpsert
	CommandEndpointRemove   = common.CommandEndpointRemove
	CommandConversationOpen = common.CommandConversationOpen
	CommandMessageSend      = common.CommandMessageSend
	CommandMessageList      = common.CommandMessageList
	CommandReceiptAck       = common.CommandReceiptAck
	CommandDeliveryUpdate   = common.CommandDeliveryUpdate
	CommandInboxSummary     = common.CommandInboxSummary
	EventChanged            = common.EventChanged

	// Budgets a caller must respect; the service rejects anything larger.
	MaxBodyBytes             = common.MaxBodyBytes
	MaxMetadataBytes         = common.MaxMetadataBytes
	MaxListItems             = common.MaxListItems
	MaxListResultBytes       = common.MaxListResultBytes
	MaxEnvelopeBytes         = common.MaxEnvelopeBytes
	OperationTimeoutSeconds  = common.OperationTimeoutSeconds
	MaxInFlight              = common.MaxInFlight
	IdempotencyRetentionDays = common.IdempotencyRetentionDays

	FaultInvalidArgument = common.FaultInvalidArgument
	FaultDenied          = common.FaultDenied
	FaultNotFound        = common.FaultNotFound
	FaultConflict        = common.FaultConflict
	FaultCapacity        = common.FaultCapacity
	FaultUnavailable     = common.FaultUnavailable

	DeliveryPending     = common.DeliveryPending
	DeliveryDispatching = common.DeliveryDispatching
	DeliveryDelivered   = common.DeliveryDelivered
	DeliveryFailed      = common.DeliveryFailed
	DeliveryUnknown     = common.DeliveryUnknown
)

// Counters above 2^53 are carried as decimal strings, so a browser consumer
// cannot silently round a sequence or a cursor.
type (
	DecimalInt64  = common.DecimalInt64
	DecimalUint64 = common.DecimalUint64

	FaultCode       = common.FaultCode
	Fault           = common.Fault
	OperationStatus = common.OperationStatus

	EndpointIdentity = common.EndpointIdentity
	Conversation     = common.Conversation

	MetadataEntry    = common.MetadataEntry
	MessageReference = common.MessageReference
	DeliveryState    = common.DeliveryState
	MessageDelivery  = common.MessageDelivery
	Message          = common.Message
	Receipt          = common.Receipt

	IdempotencyRecord = common.IdempotencyRecord

	EndpointUpsertRequest    = common.EndpointUpsertRequest
	EndpointUpsertResult     = common.EndpointUpsertResult
	EndpointRemoveRequest    = common.EndpointRemoveRequest
	EndpointRemoveResult     = common.EndpointRemoveResult
	ConversationOpenRequest  = common.ConversationOpenRequest
	ConversationOpenResult   = common.ConversationOpenResult
	MessageSendRequest       = common.MessageSendRequest
	MessageSendResult        = common.MessageSendResult
	MessageListRequest       = common.MessageListRequest
	MessageListResult        = common.MessageListResult
	ReceiptAckRequest        = common.ReceiptAckRequest
	ReceiptAckResult         = common.ReceiptAckResult
	DeliveryUpdateRequest    = common.DeliveryUpdateRequest
	DeliveryUpdateResult     = common.DeliveryUpdateResult
	InboxSummaryRequest      = common.InboxSummaryRequest
	InboxConversationSummary = common.InboxConversationSummary
	InboxSummaryResult       = common.InboxSummaryResult
	ChangedEvent             = common.ChangedEvent

	// Payload is the closed set this contract accepts. The union omits ~ so a
	// named wrapper carrying an unclassified field cannot join it.
	Payload = common.Payload

	Command[Req, Resp Payload] = common.Command[Req, Resp]
	Event[T Payload]           = common.Event[T]
)

// The generated descriptors. Each pins an operation name, the contract
// revision and the schema hash, so a peer on another revision is refused
// rather than mis-decoded.
var (
	UpsertEndpoint   = common.UpsertEndpoint
	RemoveEndpoint   = common.RemoveEndpoint
	OpenConversation = common.OpenConversation
	SendMessage      = common.SendMessage
	ListMessages     = common.ListMessages
	AckReceipt       = common.AckReceipt
	UpdateDelivery   = common.UpdateDelivery
	GetInboxSummary  = common.GetInboxSummary
	Changed          = common.Changed
)

// Call invokes an agent-messaging command through the host.
func Call[Req, Resp Payload](ctx context.Context, host pluginsdk.Host, command Command[Req, Resp], req Req) (Resp, error) {
	return common.Call(ctx, host, command, req)
}

// Handle registers a typed command handler on the shared registrar.
func Handle[Req, Resp Payload](registrar *pluginsdk.Registrar, command Command[Req, Resp], fn func(context.Context, pluginsdk.Caller, Req) (Resp, error)) {
	common.Handle(registrar, command, fn)
}

// On subscribes to the lossy changed event. Silence is indistinguishable from
// loss on a host without StatusSubscriber, so reconcile by cursor as well.
func On[T Payload](host pluginsdk.Host, event Event[T], fn func(T)) (pluginsdk.Subscription, error) {
	return common.On(host, event, fn)
}

// Emit publishes an agent-messaging event through the host.
func Emit[T Payload](host pluginsdk.Host, event Event[T], value T) error {
	return common.Emit(host, event, value)
}
