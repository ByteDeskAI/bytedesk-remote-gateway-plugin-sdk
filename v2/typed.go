package pluginsdk

import (
	"context"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

type Command[Req, Resp any] = plugin.Command[Req, Resp]

const (
	PointAIDecision          = plugin.PointAIDecision
	MetadataContractName     = plugin.MetadataContractName
	MetadataContractRevision = plugin.MetadataContractRevision
	MetadataContractSchema   = plugin.MetadataContractSchema
)

// NewValidatedCommand is the shared SDK constructor for generated contracts.
func NewValidatedCommand[Req interface{ Validate() error }, Resp interface{ Validate() error }](name string, revision uint32, schemaHash string, subject Subject) Command[Req, Resp] {
	return plugin.NewValidatedCommand[Req, Resp](name, revision, schemaHash, subject)
}

func DecodeValidated[T interface{ Validate() error }](data []byte) (T, error) {
	return plugin.DecodeValidated[T](data)
}

func Call[Req, Resp any](ctx context.Context, b Bus, command Command[Req, Resp], req Req) (Resp, error) {
	return plugin.Call(ctx, b, command, req)
}

func ServeAtPoint[Req, Resp any](ctx context.Context, b Bus, command Command[Req, Resp], point Point, handler func(context.Context, Req, Caller) (Resp, error)) (Service, error) {
	return plugin.ServeAtPoint(ctx, b, command, point, handler)
}

// BindDiscoveredCommand validates compatibility metadata only. The host must
// first authenticate the service owner and its live admitted generation.
func BindDiscoveredCommand[Req interface{ Validate() error }, Resp interface{ Validate() error }](service ServiceInfo, providerID string, point Point, operation string, revision uint32) (Command[Req, Resp], error) {
	return plugin.BindDiscoveredCommand[Req, Resp](service, providerID, point, operation, revision)
}
