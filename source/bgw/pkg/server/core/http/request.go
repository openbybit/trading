package http

import (
	"bytes"
	"io"

	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"google.golang.org/grpc/metadata"
)

// Request http request
type Request struct {
	ctx *types.Ctx
	md  metadata.MD
}

// NewRequest new http request
func NewRequest(ctx *types.Ctx) *Request {
	return &Request{
		ctx: ctx,
	}
}

// GetNamespace get namespace
func (r *Request) GetNamespace() string {
	panic("implement me")
}

// GetService get service
func (r *Request) GetService() string {
	return string(r.ctx.Path())
}

// GetMethod get method
func (r *Request) GetMethod() string {
	return string(r.ctx.Method())
}

// QueryString get query string
func (r *Request) QueryString() []byte {
	return r.ctx.QueryArgs().QueryString()
}

// PayLoad get payload
func (r *Request) PayLoad() io.Reader {
	// any
	// post application/json application/x-www-form-urlencoded text/plain
	return bytes.NewBuffer(r.ctx.Request.Body())
}

// GetMetadata get metadata
func (r *Request) GetMetadata() metadata.MD {
	if r.md == nil {
		r.md = gmetadata.MDFromContext(r.ctx).Request()
	}
	return r.md
}

// SetMetadata append metadata
func (r *Request) SetMetadata(key, value string) {
	if r.md == nil {
		r.md = gmetadata.MDFromContext(r.ctx).Request()
	}
	r.md.Set(key, value)
}

// String to string
func (r *Request) String() string {
	data := map[string]interface{}{
		"service":  r.GetService(),
		"method":   string(r.ctx.Method()),
		"metadata": r.GetMetadata(),
		"query":    string(r.QueryString()),
	}

	payload := r.ctx.Request.Body()
	if len(payload) <= 1024 {
		data["payload"] = string(payload)
	} else {
		data["payload"] = string(payload[:1024])
	}

	return util.ToJSONString(data)
}
