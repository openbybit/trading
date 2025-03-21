package bizmetedata

import (
	"context"

	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"google.golang.org/grpc/metadata"
)

type headers struct {
}

var headersKey = "headers"

func init() {
	gmetadata.Register(headersKey, &headers{})
}

type Headers struct {
	AllInbound    bool
	InboundHeader map[string]string
	InboundCookie map[string]string
}

// NewHeaders returns a new Headers
func NewHeaders() *Headers {
	return &Headers{
		InboundHeader: make(map[string]string, 10),
		InboundCookie: make(map[string]string, 10),
	}
}

type headersCtxKey struct{}

// WithHeadersMetadata sets the headers metadata
func WithHeadersMetadata(ctx context.Context, data *Headers) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(headersKey, data)
	} else {
		return context.WithValue(ctx, headersCtxKey{}, data)
	}
	return nil
}

// HeadersFromContext returns the headers metadata from the context
func HeadersFromContext(ctx context.Context) *Headers {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(headersKey)
	} else {
		v = ctx.Value(headersCtxKey{})
	}
	data, ok := v.(*Headers)
	if !ok {
		return NewHeaders()
	}
	return data
}

// Extract returns the headers metadata from the context
func (h *headers) Extract(ctx context.Context) metadata.MD {
	data := HeadersFromContext(ctx)

	md := make(metadata.MD, 5)
	if len(data.InboundHeader) > 0 {
		if !data.AllInbound {
			md.Set("header", util.ToJSONString(data.InboundHeader))
		}
		for k, v := range data.InboundHeader {
			md.Set(k, v)
		}
	}
	if len(data.InboundCookie) > 0 {
		md.Set("cookie", util.ToJSONString(data.InboundCookie))
	}
	return md
}
