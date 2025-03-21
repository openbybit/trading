package grpc

import (
	"bytes"
	"io"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"google.golang.org/grpc/metadata"
)

// RPCRequest is a rpc request
type RPCRequest struct {
	ctx       *types.Ctx
	namespace string
	service   string
	method    string
	md        metadata.MD
}

// NewRPCRequest create a rpc request
func NewRPCRequest(ctx *types.Ctx, namespace, service, method string) *RPCRequest {
	return &RPCRequest{
		ctx:       ctx,
		namespace: namespace,
		service:   service,
		method:    method,
	}
}

// GetNamespace get namespace
func (r *RPCRequest) GetNamespace() string {
	return r.namespace
}

// GetService get service
func (r *RPCRequest) GetService() string {
	return r.service
}

// GetMethod get method
func (r *RPCRequest) GetMethod() string {
	return r.method
}

// QueryString get query string
func (r *RPCRequest) QueryString() []byte {
	return nil
}

// PayLoad get payload
func (r *RPCRequest) PayLoad() (reader io.Reader) {
	var (
		body []byte
		ok   bool
		err  error
	)

	defer func() {
		if err == nil {
			reader = bytes.NewBuffer(body)
		}
	}()

	if body, ok = gmetadata.RequestHandledBodyFromContext(r.ctx); ok {
		return
	}

	if !r.ctx.IsGet() && !bytes.HasPrefix(r.ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
		body = r.ctx.Request.Body()
		return
	}

	data := make(map[string][]string)
	iter := func(key []byte, val []byte) {
		data[cast.UnsafeBytesToString(key)] = append(data[cast.UnsafeBytesToString(key)], cast.UnsafeBytesToString(val))
	}

	if r.ctx.IsGet() {
		r.ctx.QueryArgs().VisitAll(iter)
	} else {
		r.ctx.PostArgs().VisitAll(iter)
	}

	body, err = util.JsonMarshal(data)
	return
}

// GetMetadata get metadata
func (r *RPCRequest) GetMetadata() metadata.MD {
	if r.md == nil {
		r.md = gmetadata.MDFromContext(r.ctx).Request()
	}
	return r.md
}

// SetMetadata get metadata
func (r *RPCRequest) SetMetadata(key, value string) {
	if r.md == nil {
		r.md = gmetadata.MDFromContext(r.ctx).Request()
	}
	r.md.Set(key, value)
}

// String to string
func (r *RPCRequest) String() string {
	data := map[string]interface{}{
		"service":  r.GetService(),
		"method":   r.GetMethod(),
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
