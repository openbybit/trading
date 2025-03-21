package context

import (
	"bytes"
	"context"
	"strings"
	"time"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
)

const (
	rootDomain = "root-domain"
)

type globalContextFilter struct{}

func newGlobalFilter() filter.Filter {
	return &globalContextFilter{}
}

func (f *globalContextFilter) GetName() string {
	return filter.ContextFilterKeyGlobal
}

func (f *globalContextFilter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (err error) {
		f.inbound(ctx)

		err = next(ctx)

		f.outbound(ctx)

		return
	}
}

func (f *globalContextFilter) Init(ctx context.Context, args ...string) error {
	return nil
}

func (f *globalContextFilter) inbound(ctx *types.Ctx) {
	md := gmetadata.MDFromContext(ctx)

	if md.Extension.RemoteIP == "" {
		md.Extension.RemoteIP = bhttp.GetRemoteIP(ctx)
	}

	if !md.WssFlag {
		md.ReqInitTime = time.Now()
	}
	md.ReqInitAtE9 = string(ctx.Request.Header.Peek(constant.ReqInitAtE9)) // parse traffic init time
	if md.ReqInitAtE9 == "" {
		md.ReqInitAtE9 = cast.Int64toa(md.ReqInitTime.UnixNano())
	}

	parseCommon(ctx, md)

	md.Version = constant.Version
	md.Path = string(ctx.Path())
	md.BrokerID = int32(cast.Atoi(string(ctx.Request.Header.Peek(constant.BrokerID))))
	md.SiteID = string(ctx.Request.Header.Peek(constant.SiteID))
	md.ContentType = string(ctx.Request.Header.ContentType())
	md.Extension.Origin = string(ctx.Request.Header.Peek(constant.Origin))
	md.Extension.XOriginFrom = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.XOriginFrom))
	md.Extension.DeviceID = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.DeviceID))
	md.Extension.Guid = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.Guid))
	md.Extension.XClientTag = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.XClientTag))
	md.Extension.XReferer = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.XReferer))
	md.Extension.Fingerprint = md.Extension.DeviceID
	if md.Extension.Fingerprint == "" {
		md.Extension.Fingerprint = md.Extension.Guid
	}
	md.Extension.GFingerprint = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.GFingerprint))
	md.Intermediate.CallOrigin = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.CallOrigin))
	md.AKMTrace = util.DecodeHeaderValue(ctx.Request.Header.Peek(constant.XAKMTraceID))
	md.Extension.Language = parseLang(ctx)
	md.Extension.GWSource = constant.GWSource

	parseBaggage(ctx, md)

	md.WithContext(ctx)
}

func (f *globalContextFilter) outbound(ctx *types.Ctx) {
	var rmd metadata.MD
	if carrier, ok := ctx.UserValue(constant.CtxInvokeResult).(mdCarrier); ok && carrier != nil {
		rmd = carrier.Metadata()
	}

	md := gmetadata.MDFromContext(ctx)
	if md.Intermediate.SecureToken != nil {
		ck := fasthttp.AcquireCookie()
		ck.SetKey(config.GetSecureTokenKey())
		ck.SetValue(*md.Intermediate.SecureToken)
		ck.SetPath("/")
		ck.SetHTTPOnly(true)
		ck.SetSecure(true)
		ck.SetSameSite(fasthttp.CookieSameSiteLaxMode)
		domain := string(ctx.Request.Header.Peek(rootDomain))
		if domain != "" {
			ck.SetDomain("." + domain)
		} else {
			strSplit := bytes.Split(ctx.Request.Host(), []byte("."))
			if len(strSplit) > 2 {
				ck.SetDomain(string(bytes.Join(strSplit[1:], []byte("."))))
			} else {
				ck.SetDomainBytes(ctx.Request.Host())
			}
		}
		expireT := time.Now().Add(3 * 24 * time.Hour)
		ck.SetExpire(expireT)
		ctx.Response.Header.SetCookie(ck)
		fasthttp.ReleaseCookie(ck)
	}
	if md.Intermediate.WeakToken != nil {
		ctx.Response.Header.Set("token", *md.Intermediate.WeakToken)
	}

	ctx.Response.Header.Set("traceID", md.TraceID)
	ctx.Response.Header.Set("timeNow", cast.Int64toa(time.Now().UnixMilli()))
	ctx.Response.Header.SetContentType("application/json; charset=utf-8")

	vs := rmd.Get("content-type")
	if len(vs) == 0 {
		return
	}

	if strings.Contains(vs[0], "application/grpc") || strings.Contains(vs[0], "text/plain") {
		return
	}

	ctx.Response.Header.SetContentType(vs[0])
}
