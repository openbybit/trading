package core

import (
	"fmt"
	"io"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/runtime/protoiface"
)

// Result is a RPC result
type Result interface {
	// SetStatus sets error.
	SetStatus(int)
	// GetStatus gets error.
	GetStatus() int
	// SetData sets data.
	SetData(io.Reader) error
	// GetData gets data.
	GetData() ([]byte, error)
	// Metadata gets response metadata.
	Metadata() metadata.MD
	// SetMetadata set response metadata
	SetMetadata(metadata.MD)
	// SetMessage set proto message
	SetMessage(protoiface.MessageV1)
	// Stringer response string
	fmt.Stringer
	// Closer & write payload
	io.Closer
}
