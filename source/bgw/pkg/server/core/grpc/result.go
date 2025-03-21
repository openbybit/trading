package grpc

import (
	"io"

	"bgw/pkg/common/util"

	// nolint
	"github.com/golang/protobuf/jsonpb"
	"github.com/valyala/bytebufferpool"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/runtime/protoiface"
)

var (
	marshaler = jsonpb.Marshaler{EmitDefaults: true}
	pool      = new(bytebufferpool.Pool)
)

// RPCResult is default RPC result.
type RPCResult struct {
	status  int
	message protoiface.MessageV1
	data    *bytebufferpool.ByteBuffer
	md      metadata.MD
}

// NewResult creates a new Result.
func NewResult() *RPCResult {
	return &RPCResult{}
}

// SetStatus set rpc code.
func (r *RPCResult) SetStatus(status int) {
	r.status = status
}

// GetStatus get rpc code.
func (r *RPCResult) GetStatus() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Metadata gets invoker metadata.
func (r *RPCResult) Metadata() metadata.MD {
	if r == nil {
		return metadata.New(nil)
	}
	if r.md == nil {
		r.md = metadata.New(nil)
	}
	return r.md
}

// SetMetadata set response md
func (r *RPCResult) SetMetadata(md metadata.MD) {
	r.md = md
}

// SetData set data
func (r *RPCResult) SetData(_ io.Reader) error { return nil }

// GetData get data
func (r *RPCResult) GetData() ([]byte, error) {
	if r == nil || r.message == nil {
		return nil, nil
	}

	r.data = pool.Get()
	if err := marshaler.Marshal(r.data, r.message); err != nil {
		return nil, err
	}

	return r.data.Bytes(), nil
}

// GetMessage get message
func (r *RPCResult) GetMessage() protoiface.MessageV1 {
	if r == nil {
		return nil
	}
	return r.message
}

// SetMessage set message
func (r *RPCResult) SetMessage(msg protoiface.MessageV1) { r.message = msg }

// Close close result
func (r *RPCResult) Close() error {
	if r == nil {
		return nil
	}
	if r.data != nil {
		pool.Put(r.data)
		r.data = nil
	}

	r.status = 0
	r.message = nil
	return nil
}

// String to string
func (r *RPCResult) String() string {
	if r == nil {
		return ""
	}
	data := map[string]interface{}{
		"metadata": r.Metadata(),
	}

	var v string
	if r.message != nil {
		v = r.message.String()
		if len(v) > 1024 {
			v = v[:1024]
		}
	}

	data["message"] = v
	return util.ToJSONString(data)
}
