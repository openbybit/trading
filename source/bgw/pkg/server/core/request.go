package core

import (
	"fmt"
	"io"

	"google.golang.org/grpc/metadata"
)

// Request is a RPC request
type Request interface {
	GetNamespace() string
	// GetService get service name
	GetService() string
	// GetMethod get service name
	GetMethod() string
	// QueryString QueryString
	QueryString() []byte
	// PayLoad Payload
	PayLoad() io.Reader
	// GetMetadata metadata
	GetMetadata() metadata.MD
	// SetMetadata append metadata
	SetMetadata(key, value string)
	// Stringer string format
	fmt.Stringer
}
