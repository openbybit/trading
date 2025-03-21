package http

import (
	"io"

	"bgw/pkg/common/util"

	"github.com/valyala/bytebufferpool"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/runtime/protoiface"
)

var (
	pool = new(bytebufferpool.Pool)
)

// Result http result
type Result struct {
	status int
	data   *bytebufferpool.ByteBuffer
	md     metadata.MD
}

// NewResult create a new result
func NewResult() *Result {
	return &Result{}
}

// SetStatus set http code.
func (r *Result) SetStatus(code int) {
	r.status = code
}

// GetStatus get http code.
func (r *Result) GetStatus() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Metadata gets invoker metadata.
func (r *Result) Metadata() metadata.MD {
	if r == nil {
		return metadata.New(nil)
	}
	if r.md == nil {
		r.md = metadata.New(nil)
	}
	return r.md
}

// SetMetadata set response md
func (r *Result) SetMetadata(md metadata.MD) {
	r.md = md
}

// SetData set data
func (r *Result) SetData(data io.Reader) error {
	r.data = pool.Get()
	_, err := r.data.ReadFrom(data)
	return err
}

// GetData get data
func (r *Result) GetData() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return r.data.Bytes(), nil
}

// SetMessage set message
func (r *Result) SetMessage(_ protoiface.MessageV1) {
	// do nothing, not support
}

// Close retrieve buffer
func (r *Result) Close() error {
	if r == nil {
		return nil
	}

	if r.data != nil {
		pool.Put(r.data)
		r.data = nil
	}

	r.status = 0
	return nil
}

// String string
func (r *Result) String() string {
	data := map[string]interface{}{
		"metadata": r.Metadata(),
	}
	if r.data != nil {
		if r.data.Len() <= 1024 {
			data["payload"] = r.data.String()
		} else {
			data["payload"] = string(r.data.Bytes()[:1024])
		}
	}

	return util.ToJSONString(data)
}
