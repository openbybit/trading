package ghttp

import (
	"io"

	"google.golang.org/grpc/metadata"
)

// Result is a RPC result
type Result interface {
	// SetStatus sets error.
	SetStatus(int)
	// SetMetadata set response metadata
	SetMetadata(metadata.MD)
	// SetData sets response data.
	SetData(io.Reader) error
}
