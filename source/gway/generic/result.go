package generic

import (
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/runtime/protoiface"
)

// Result is a RPC result
type Result interface {
	// SetMetadata set response metadata
	SetMetadata(metadata.MD)
	SetMessage(protoiface.MessageV1)
}
