package types

import (
	"github.com/valyala/fasthttp"
)

// Handler type Handler = ffc.L4HandlerFunc
type Handler = func(rctx *fasthttp.RequestCtx) error
type Ctx = fasthttp.RequestCtx
type Header = fasthttp.RequestHeader
type RequestHandler = fasthttp.RequestHandler
