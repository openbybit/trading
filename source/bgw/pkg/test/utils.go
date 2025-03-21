package test

import (
	"bgw/pkg/common/constant"
	gmetadata "bgw/pkg/server/metadata"
	"github.com/valyala/fasthttp"
)

func NewReqCtx() (*fasthttp.RequestCtx, *gmetadata.Metadata) {
	r := &fasthttp.RequestCtx{
		Request:  fasthttp.Request{Header: fasthttp.RequestHeader{}},
		Response: fasthttp.Response{Header: fasthttp.ResponseHeader{}},
	}

	md := gmetadata.NewMetadata()
	md.Route = gmetadata.RouteKey{
		AppName:     "test",
		ModuleName:  "ccc",
		ServiceName: "aaa",
		Registry:    "ddd",
		MethodName:  "xxx",
		HttpMethod:  "get",
	}
	r.SetUserValue(constant.METADATA_CTX, md)
	return r, md
}
