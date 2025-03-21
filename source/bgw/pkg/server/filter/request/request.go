package request

import (
	"bytes"
	"context"
	"flag"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func Init() {
	filter.Register(filter.RequestFilterKey, new) // route filter
}

type requestData struct {
	Request []byte `json:"request"`
}
type request struct {
	duplicateKey bool
}

func new() filter.Filter {
	return &request{}
}

// GetName returns the name of the filter
func (*request) GetName() string {
	return filter.RequestFilterKey
}

func (r *request) Init(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return nil
	}

	return r.parseFloags(args)
}

// Do implements the filter interface
func (r *request) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		payload, ok := metadata.RequestHandledBodyFromContext(ctx)
		glog.Debug(ctx, "RequestHandledFromContext handle", glog.Bool("handled", ok))

		if !ok {
			if ctx.IsGet() {
				payload = r.parseRequest(ctx, true)
			} else {
				fields := []glog.Field{
					glog.String("content-type", string(ctx.Request.Header.ContentType())),
					glog.String("query", cast.UnsafeBytesToString(ctx.URI().QueryString())),
				}
				if len(ctx.Request.Body()) > 1024 {
					fields = append(fields, glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body()[:1024])))
				} else {
					fields = append(fields, glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())))
				}
				glog.Debug(ctx, "request raw data", fields...)

				if len(ctx.Request.Body()) == 0 {
					payload = r.parseRequest(ctx, true)
				} else if bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
					payload = r.parseRequest(ctx, false)
				} else {
					payload = ctx.Request.Body()
				}
			}
		}

		data := &requestData{
			Request: payload,
		}

		body, err := util.JsonMarshal(data)
		if err != nil {
			glog.Error(ctx, "request filter json.Marshal error", glog.String("error", err.Error()))
			return berror.NewInterErr(err.Error())
		}

		metadata.ContextWithRequestHandledBody(ctx, body)

		return next(ctx)
	}
}

func (r *request) parseRequest(ctx *types.Ctx, isQuery bool) []byte {
	var (
		data interface{}
		iter func(key []byte, val []byte)
	)

	if r.duplicateKey {
		data = make(map[string][]string, 5)
		iter = func(key []byte, val []byte) {
			m := data.(map[string][]string)
			m[cast.UnsafeBytesToString(key)] = append(m[cast.UnsafeBytesToString(key)], cast.UnsafeBytesToString(val))
		}
	} else {
		data = make(map[string]string, 5)
		iter = func(key []byte, val []byte) {
			m := data.(map[string]string)
			if _, ok := m[cast.UnsafeBytesToString(key)]; !ok {
				m[cast.UnsafeBytesToString(key)] = cast.UnsafeBytesToString(val)
			}
		}
	}

	if isQuery {
		ctx.QueryArgs().VisitAll(iter)
	} else {
		ctx.PostArgs().VisitAll(iter)
	}

	b, err := util.JsonMarshal(data)
	if err != nil {
		glog.Error(ctx, "request filter json.Marshal error", glog.String("error", err.Error()))
	}

	return b
}

func (r *request) parseFloags(args []string) (err error) {
	parse := flag.NewFlagSet("request", flag.ContinueOnError)
	parse.BoolVar(&r.duplicateKey, "duplicateKey", false, "request have duplicate key header")

	return parse.Parse(args[1:])
}
