package response

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/runtime/protoiface"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/filter/response/version"
	gmetadata "bgw/pkg/server/metadata"
)

func Init() {
	filter.Register(filter.ResponseFilterKey, new)
}

type resultCarrier interface {
	GetStatus() int
	GetData() ([]byte, error)
	metadataCarrier
	io.Closer
}

type metadataCarrier interface {
	Metadata() metadata.MD
}

type messager interface {
	GetMessage() protoiface.MessageV1
}

type responseFlags struct {
	version     string     // output data struct version, value: v1, v2, passthrough(not set), default v2
	rateLimit   bool       // ratelimit info
	extInfoStr  bool       // for openapi v2 only, mark ext_info is string
	any         bool       // passthrough result data only
	passthrough bool       // passthrough result raw
	convert     bool       // convert v2 to v1
	metaBatch   bool       // set batch code&message in ext_info
	translator  *translate // translate code and message
	timeInt     bool       // for cht v1, mark time_now is int on v1
}

type response struct {
	flags   responseFlags
	handler handler
}

func new() filter.Filter {
	return &response{}
}

// GetName returns the name of the filter
func (*response) GetName() string {
	return filter.ResponseFilterKey
}

// Do do response filter
func (r *response) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (err error) {
		defer func() {
			if v := recover(); v != nil {
				r.recover(ctx, v)
			}
		}()

		err = next(ctx)

		// skip response filter
		if r.skip(ctx) {
			return
		}

		var (
			source resultCarrier      // upstream result, but response filter source
			target = r.getTarget(ctx) // get target version by flags and ctx
		)

		defer func() {
			if err != nil {
				// handle any error, fall back to passthrough strategy
				if err == errInvalidAnyResult {
					if t := handlePassthrough(source); t != nil {
						target = t
						err = nil
					}
				} else {
					target = r.handleError(ctx, target.Version(), err)
				}
			}

			r.finally(ctx, source, target)
		}()

		// fail fast if error
		if err != nil {
			return
		}

		source, ok := ctx.UserValue(constant.CtxInvokeResult).(resultCarrier)
		if !ok || source == nil {
			err = errInvalidResult
			return
		}

		err = r.handler(ctx, source, target)
		return
	}
}

// Init init response filter
func (r *response) Init(_ context.Context, args ...string) (err error) {
	switch len(args) {
	case 0:
		r.flags = responseFlags{version: version.VersionV2, translator: &translate{}}
	default:
		if err = r.parseFlags(args...); err != nil {
			return
		}
	}

	switch {
	case r.flags.any:
		r.handler = handleAny
	case r.flags.convert:
		r.handler = handleConvert
	default:
		r.handler = handleDefault
	}

	return
}

func (r *response) parseFlags(args ...string) (err error) {
	var (
		msgSource, msgTag int64
		flags             = responseFlags{translator: &translate{}}
	)

	p := flag.NewFlagSet("response", flag.ContinueOnError)
	p.StringVar(&flags.version, "version", "v2", "response protocol, default v2")
	p.BoolVar(&flags.any, "any", false, "response body any, key is result, type must bytes, just for v2 grpc")
	p.BoolVar(&flags.passthrough, "passthrough", false, "response body not covert anything")
	p.BoolVar(&flags.convert, "metaInBody", false, "response code msg in body")
	p.BoolVar(&flags.metaBatch, "metaBatch", false, "response code msg is batch")
	p.BoolVar(&flags.rateLimit, "rateLimit", false, "response set rate limit info")        // for fbu v2 openapi
	p.BoolVar(&flags.extInfoStr, "extInfoToString", false, "response ext_info set string") // for fbu v2 openapi
	p.Int64Var(&msgSource, "msgSource", 0, "response msg parse source")
	p.Int64Var(&msgTag, "msgTag", 0, "response msg parse tag")
	p.BoolVar(&flags.timeInt, "timeInt", false, "response time_now set int")

	if err = p.Parse(args[1:]); err != nil {
		glog.Info(context.TODO(), "response parse error", glog.Any("args", args), glog.String("error", err.Error()))
		return
	}
	flags.translator.msgSource = msgSourceType(msgSource)
	flags.translator.msgTag = msgTag

	r.flags = flags

	routeKey := gmetadata.RouteKey{}
	routeKey = routeKey.Parse(args[0])
	if !routeKey.AllApp {
		setCodeLoader(routeKey.AppName)
		return
	}
	setCodeLoader(constant.AppTypeFUTURES)
	setCodeLoader(constant.AppTypeSPOT)
	setCodeLoader(constant.AppTypeOPTION)

	return
}

// skip skip invalid types
func (r *response) skip(ctx *types.Ctx) bool {
	return ctx.IsOptions()
}

func (r *response) recover(ctx *types.Ctx, v interface{}) {
	// get original req and resp
	req := &fasthttp.Request{}
	resp := &fasthttp.Response{}
	ctx.Request.CopyTo(req)
	ctx.Response.CopyTo(resp)

	// set panic response
	ctx.SetStatusCode(berror.HttpServerError)
	ctx.Response.ResetBody()

	glog.Error(ctx, "panic", glog.String("path", fmt.Sprintf("%s-%s", ctx.Method(), ctx.Path())),
		glog.Any("panic", v), glog.StackSkip("panic stack", 3), glog.Any("req", req),
		glog.Any("resp", resp))
	// alert
	var err error
	if e, ok := v.(error); ok {
		err = e
	} else {
		err = berror.ErrDefault
	}
	msg := fmt.Sprintf("panic recovery, err = %s", err.Error())
	galert.Error(ctx, msg)
}

// handleError handle error
func (r *response) handleError(ctx *types.Ctx, v string, err error) (target Target) {
	target = r.getErrorTarget(v)

	var (
		code    = berror.SystemInternalError
		message = berror.ErrDefault.Error()
	)

	switch e := err.(type) {
	case berror.BizErr:
		code = e.GetCode()
		message = err.Error()
	default:
		glog.Info(ctx, "server error", glog.String("error", err.Error()))
		if !env.IsProduction() {
			if tar, ok := target.(withError); ok {
				tar.SetError(err.Error())
			}
		}
	}

	target.SetCode(code)
	target.SetMessage(message)
	target.SetResult(emptyJSON)

	// blocktrade error
	if gmetadata.MDFromContext(ctx).Route.ACL.Group == constant.ResourceGroupBlockTrade {
		d := handleBlockTradeError(ctx)
		target.SetExtInfo(d)
	}

	return
}

func (r *response) finally(ctx *types.Ctx, source resultCarrier, target Target) {
	if target.Version() != version.VersionPassthrough {
		// do response translate
		r.flags.translator.do(ctx, source, target, r.flags.metaBatch)
	}

	if target.Version() == version.VersionV1 && source != nil {
		md := source.Metadata()
		if codes := md.Get(constant.BgwAPIResponseExtCode); len(codes) > 0 {
			target.SetExtCode(codes[0])
		}
	}

	// ! extend target after translate
	r.extend(ctx, target)

	body, _ := target.Marshal()
	ctx.SetBody(body)

	if source != nil {
		// set http status
		if status := source.GetStatus(); status > 0 {
			ctx.SetStatusCode(status)
		}

		_ = source.Close()
	}
}

// extend set ext data into output
// including: rate limit details, next token
// compatible with ext_info and ext_map
func (r *response) extend(ctx *types.Ctx, target Target) {
	code := target.GetCode()
	ctx.Response.Header.Set(retCode, cast.Int64toa(code))
	if code != 0 {
		// set cache control, no store if error
		ctx.Response.Header.Set("Cache-Control", "no-store")
	}

	md := gmetadata.MDFromContext(ctx)
	if isOpenAPI(md) {
		if code != 0 {
			if len(ctx.Response.Body()) > 0 {
				target.SetResult(ctx.Response.Body())
			} else {
				target.SetResult(emptyJSON)
			}
		}

		if v, ok := target.(withRateLimit); r.flags.rateLimit && ok {
			v.SetLimit(gmetadata.RateLimitInfoFromContext(ctx))
		}
	} else {
		if code != 0 && len(ctx.Response.Body()) > 0 {
			target.SetResult(ctx.Response.Body())
		}

		if v, ok := target.(withtoken); ok {
			v.SetToken(md.Intermediate.WeakToken)
		}
	}

	if v, ok := target.(replaceTime); ok {
		var t interface{}
		if r.flags.timeInt {
			t = time.Now().UnixNano() / 1e6
		} else {
			t = fmt.Sprintf("%.6f", float64(time.Now().UnixNano())/float64(time.Second))
		}
		v.SetTime(t)
	}

	switch {
	case r.flags.extInfoStr: // overwrite ext_info to string
		target.SetExtInfo(json.RawMessage(`""`))
	case target.Version() == version.VersionV2 && len(target.GetExtInfo()) == 0:
		target.SetExtInfo(emptyJSON)
	}
}

// getTarget get version from config rule, If frontend pass a FormatKey, directly cover it
func (r *response) getTarget(ctx *types.Ctx) Target {
	var v = r.flags.version
	// format response type, only peek query string
	if format := ctx.QueryArgs().Peek(formatKey); len(format) > 0 {
		glog.Debug(ctx, "Get FormatKey from frontend", glog.String("FormatKey", string(format)))
		switch {
		case bytes.Equal(format, hump):
			v = version.VersionV2
		case bytes.Equal(format, portugal):
			v = version.VersionV1
		}
	}

	switch {
	case r.flags.passthrough:
		return version.NewPassthroughResponse()
	case v == version.VersionV1:
		return version.NewV1Response()
	default:
		return version.NewV2Response()
	}
}

func (r *response) getErrorTarget(v string) (target Target) {
	switch v {
	case version.VersionV1:
		target = version.NewV1Response()
	case version.VersionPassthrough:
		switch {
		case r.flags.any:
			target = version.NewV2Response()
		case r.flags.version == version.VersionV1:
			target = version.NewV1Response()
		default:
			target = version.NewV2Response()
		}
	default:
		target = version.NewV2Response()
	}

	return
}
