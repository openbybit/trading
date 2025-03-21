package generic

import "io"

// Request is a RPC request
type Request interface {
	GetNamespace() string
	// GetService get service name
	GetService() string
	// GetMethod get service name
	GetMethod() string
	// PayLoad Payload
	PayLoad() io.Reader
}
