package cors

import (
	"strconv"
	"strings"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var allowedMethods = []string{"GET", "POST", "PUT", "OPTIONS", "DELETE", "HEAD", "PATCH"}

func Init() {
	filter.Register(filter.CorsFilterKey, defaultHandler())
}

// Options is struct that defined cors properties
type Options struct {
	AllowedOrigins   []string
	AllowedHeaders   []string
	AllowMaxAge      int
	AllowedMethods   []string
	ExposedHeaders   []string
	AllowCredentials bool
	Debug            bool
}

type corsHandler struct {
	allowedOriginsAll bool
	allowedOrigins    []string
	allowedHeadersAll bool
	allowedHeaders    []string
	allowedMethods    []string
	exposedHeaders    []string
	allowCredentials  bool
	maxAge            int
}

var defaultOptions = &Options{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   allowedMethods,
	ExposedHeaders:   []string{"token", "X-Signature"},
	AllowCredentials: true,
	AllowMaxAge:      7200,
}

func defaultHandler() filter.Filter {
	return newCorsHandler(defaultOptions)
}

func newCorsHandler(options *Options) *corsHandler {
	cors := &corsHandler{
		allowedOrigins:   options.AllowedOrigins,
		allowedHeaders:   options.AllowedHeaders,
		allowCredentials: options.AllowCredentials,
		allowedMethods:   options.AllowedMethods,
		exposedHeaders:   options.ExposedHeaders,
		maxAge:           options.AllowMaxAge,
	}

	if len(cors.allowedOrigins) == 0 {
		cors.allowedOrigins = defaultOptions.AllowedOrigins
		cors.allowedOriginsAll = true
	} else {
		for _, v := range options.AllowedOrigins {
			if v == "*" {
				cors.allowedOrigins = defaultOptions.AllowedOrigins
				cors.allowedOriginsAll = true
				break
			}
		}
	}
	if len(cors.allowedHeaders) == 0 {
		cors.allowedHeaders = defaultOptions.AllowedHeaders
		cors.allowedHeadersAll = true
	} else {
		for _, v := range options.AllowedHeaders {
			if v == "*" {
				cors.allowedHeadersAll = true
				break
			}
		}
	}
	if len(cors.allowedMethods) == 0 {
		cors.allowedMethods = defaultOptions.AllowedMethods
	}
	return cors
}

// GetName returns the name of the filter
func (c *corsHandler) GetName() string {
	return filter.CorsFilterKey
}

// Do is the main function of the filter
func (c *corsHandler) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		if cast.UnsafeBytesToString(ctx.Method()) == "OPTIONS" {
			c.handlePreflight(ctx)
			ctx.SetStatusCode(berror.HttpStatusOK)
			return nil
		}

		c.handleActual(ctx)
		return next(ctx)
	}
}

func (c *corsHandler) handlePreflight(ctx *types.Ctx) {
	originHeader := cast.UnsafeBytesToString(ctx.Request.Header.Peek("Origin"))
	if len(originHeader) == 0 || !c.isAllowedOrigin(originHeader) {
		glog.Debug(ctx, "origin is not allowed", glog.String("origin", originHeader))
		return
	}
	method := cast.UnsafeBytesToString(ctx.Request.Header.Peek("Access-Control-Request-Method"))
	if !c.isAllowedMethod(method) {
		glog.Debug(ctx, "method is not allowed", glog.String("origin", originHeader))
		return
	}
	var headers []string
	if len(ctx.Request.Header.Peek("Access-Control-Request-Headers")) > 0 {
		headers = strings.Split(string(ctx.Request.Header.Peek("Access-Control-Request-Headers")), ",")
	}

	ctx.Response.Header.Set("Access-Control-Allow-Origin", originHeader)
	ctx.Response.Header.Set("Access-Control-Allow-Methods", method)

	if len(headers) > 0 {
		ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
	}
	if c.allowCredentials {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		ctx.Response.Header.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
}

func (c *corsHandler) handleActual(ctx *types.Ctx) {
	originHeader := cast.UnsafeBytesToString(ctx.Request.Header.Peek("Origin"))
	if len(originHeader) == 0 || !c.isAllowedOrigin(originHeader) {
		return
	}

	ctx.Response.Header.Set("Access-Control-Allow-Origin", originHeader)

	if len(c.exposedHeaders) > 0 {
		ctx.Response.Header.Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
}

func (c *corsHandler) isAllowedOrigin(originHeader string) bool {
	if c.allowedOriginsAll {
		return true
	}
	for _, val := range c.allowedOrigins {
		if val == originHeader {
			return true
		}
	}
	return false
}

func (c *corsHandler) isAllowedMethod(methodHeader string) bool {
	if len(c.allowedMethods) == 0 {
		return false
	}
	if methodHeader == "OPTIONS" {
		return true
	}
	for _, m := range c.allowedMethods {
		if m == methodHeader {
			return true
		}
	}
	return false
}

// nolint
func (c *corsHandler) areHeadersAllowed(headers []string) bool {
	if c.allowedHeadersAll || len(headers) == 0 {
		return true
	}
	for _, header := range headers {
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
