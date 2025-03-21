package context

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"strings"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	gmetadata "bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"google.golang.org/grpc/metadata"
)

const (
	nextToken = "next-token"
	weakToken = "weak-token"
)

type contextFilter struct {
	flagInboundHeaders  []string
	flagOutboundHeaders []string
	flagInboundCookies  []string
	flagWithToken       bool
	flagWithRaw         bool
}

func new() filter.Filter {
	return &contextFilter{}
}

func (f *contextFilter) GetName() string {
	return filter.ContextFilterKey
}

func (f *contextFilter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (err error) {
		f.inbound(ctx)

		err = next(ctx)

		f.outbound(ctx)

		return
	}
}

func (f *contextFilter) parseFlags(ctx context.Context, args []string) (err error) {
	var (
		inboundHeader  string
		outboundHeader string
		inboundCookie  string
		outboundCookie string
	)

	parse := flag.NewFlagSet("context", flag.ContinueOnError)
	parse.StringVar(&inboundHeader, "header", "", "context header, old flag") // compatible
	parse.StringVar(&inboundHeader, "inboundHeader", "", "context inboundHeader")
	parse.StringVar(&outboundHeader, "outboundHeader", "", "context outboundHeader")
	parse.StringVar(&inboundCookie, "inboundCookie", "", "context inboundCookie")
	parse.StringVar(&outboundCookie, "outboundCookie", "", "context inboundCookie")
	parse.BoolVar(&f.flagWithToken, "withToken", false, "response body need fill token, just for member login, member register")
	parse.BoolVar(&f.flagWithRaw, "needRaw", false, "request need raw data, only for POST")

	if err = parse.Parse(args[1:]); err != nil {
		return
	}

	if inboundHeader != "" {
		if err = json.Unmarshal([]byte(inboundHeader), &f.flagInboundHeaders); err != nil {
			glog.Error(ctx, "context filter rule Unmarshal inboundHeader error", glog.Any("args", args), glog.String("error", err.Error()))
			return
		}
	}

	if outboundHeader != "" {
		if err = json.Unmarshal([]byte(outboundHeader), &f.flagOutboundHeaders); err != nil {
			glog.Error(ctx, "context filter rule Unmarshal outboundHeader error", glog.Any("args", args), glog.String("error", err.Error()))
			return
		}
	}

	if inboundCookie != "" {
		if err = json.Unmarshal([]byte(inboundCookie), &f.flagInboundCookies); err != nil {
			glog.Error(ctx, "context filter rule Unmarshal inboundCookie error", glog.Any("args", args), glog.String("error", err.Error()))
			return
		}
	}

	return
}

func (f *contextFilter) Init(ctx context.Context, args ...string) (err error) {
	if len(args) == 0 {
		return nil
	}

	return f.parseFlags(ctx, args)
}

func (f *contextFilter) inbound(ctx *types.Ctx) {
	md := gmetadata.MDFromContext(ctx)
	// inboundHeader
	f.inboundHeader(ctx)
	// inboundCookie
	f.inboundCookie(ctx)

	if f.flagWithRaw && md.Method != http.MethodGet {
		md.Extension.RawBody = ctx.Request.Body()
	}
}

func (f *contextFilter) outbound(ctx *types.Ctx) {
	var rmd metadata.MD
	if carrier, ok := ctx.UserValue(constant.CtxInvokeResult).(mdCarrier); ok && carrier != nil {
		rmd = carrier.Metadata()
	}

	md := gmetadata.MDFromContext(ctx)

	// withToken
	if f.flagWithToken {
		// add next-token
		v := rmd.Get(nextToken)
		if len(v) == 0 || v[0] == "" {
			glog.Info(ctx, "peek next token missing", glog.String("key", nextToken), glog.Any("out metadata", rmd))
		} else {
			nToken := v[0]
			md.Intermediate.SecureToken = &nToken
		}

		// add weak-token
		v = rmd.Get(weakToken)
		if len(v) == 0 || v[0] == "" {
			glog.Info(ctx, "peek weak token missing", glog.String("key", weakToken), glog.Any("out metadata", rmd))
		} else {
			wToken := v[0]
			md.Intermediate.WeakToken = &wToken
		}
	}

	// outboundHeader
	f.outboundHeader(ctx, rmd)
}

func (f *contextFilter) outboundHeader(ctx *types.Ctx, rmd metadata.MD) {
	if len(f.flagOutboundHeaders) == 0 {
		// outboundCookie
		f.outboundCookie(ctx, rmd)
		return
	}

	if len(f.flagOutboundHeaders) == 1 && f.flagOutboundHeaders[0] == all {
		for k, vs := range rmd {
			// inner header ignore
			if _, ok := innerOutboundHeaders[k]; ok {
				continue
			}
			// other header
			for _, v := range vs {
				ctx.Response.Header.Add(k, v)
			}
		}
		return
	}

	for _, key := range f.flagOutboundHeaders {
		if key == "" {
			continue
		}
		values := rmd.Get(key)
		if len(values) > 0 {
			ctx.Response.Header.Set(key, values[0])
		} else {
			glog.Debug(ctx, "peek outboundHeader missing", glog.String("outboundHeader", key))
		}
	}

	// outboundCookie
	f.outboundCookie(ctx, rmd)
}

func (f *contextFilter) outboundCookie(c *types.Ctx, rmd metadata.MD) {
	values := rmd.Get(constant.SetCookie)
	if len(values) == 0 {
		glog.Debug(c, "upstream no outboundCookie")
		return
	}

	for _, value := range values {
		c.Response.Header.Add(constant.SetCookie, value)
	}
}

func (f *contextFilter) inboundHeader(ctx *types.Ctx) {
	if len(f.flagInboundHeaders) == 0 {
		return
	}

	hds := bizmetedata.HeadersFromContext(ctx)
	defer func() { bizmetedata.WithHeadersMetadata(ctx, hds) }()

	if len(f.flagInboundHeaders) == 1 && f.flagInboundHeaders[0] == all {
		hds.AllInbound = true
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			// extension header not pass though
			if _, ok := innerInboundHeaders[strings.ToLower(string(key))]; ok {
				return
			}
			hds.InboundHeader[string(key)] = string(value)
		})
		return
	}

	for _, k := range f.flagInboundHeaders {
		if k == "" {
			continue
		}
		v := ctx.Request.Header.Peek(k)
		if len(v) == 0 {
			glog.Debug(ctx, "peek inboundHeader missing", glog.String("inboundHeader", k))
			continue
		}
		hds.InboundHeader[k] = cast.ToString(v)
	}
}

func (f *contextFilter) inboundCookie(ctx *types.Ctx) {
	if len(f.flagInboundCookies) == 0 {
		return
	}

	hds := bizmetedata.HeadersFromContext(ctx)
	for _, key := range f.flagInboundCookies {
		if key == "" {
			continue
		}
		if v := ctx.Request.Header.Cookie(key); len(v) > 0 {
			hds.InboundCookie[key] = string(v)
		}
	}
	bizmetedata.WithHeadersMetadata(ctx, hds)
}
