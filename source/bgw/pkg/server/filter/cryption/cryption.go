package cryption

import (
	"context"
	"flag"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	gmd "google.golang.org/grpc/metadata"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service"
)

func init() {
	filter.Register(filter.CryptionFilterKey, newCrypter)
}

const (
	cryptionReqHeader  = "X-Signature"
	cyrptionRespHeader = "X-Signature"
)

type crypter struct {
	Req  bool
	Resp bool
}

func newCrypter() filter.Filter {
	return &crypter{}
}

// GetName returns the name of the filter
func (s *crypter) GetName() string {
	return filter.CryptionFilterKey
}

// Do will call next handler
func (s *crypter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (respErr error) {
		md := metadata.MDFromContext(ctx)
		if !getCipher(ctx).check(md.UID) {
			return next(ctx)
		}

		otelCtx := gtrace.OtelCtxFromOtraCtx(service.GetContext(ctx))

		if s.Req {
			sign := ctx.Request.Header.Peek(cryptionReqHeader)
			glog.Debug(ctx, "req sign", glog.String("sign", string(sign)))
			ok, err := getCipher(ctx).VerifySign(otelCtx, string(ctx.Request.URI().QueryString()), ctx.Request.Body(), string(sign))
			if err != nil {
				glog.Error(ctx, "check req sign failed", glog.String("err", err.Error()))
				gmetric.IncDefaultError("crypter", "req")
				return berror.ErrBadSign
			}

			if !ok {
				return berror.ErrBadSign
			}
		}

		defer func() {
			if !s.Resp || respErr != nil {
				return
			}
			res, ok := ctx.UserValue(constant.CtxInvokeResult).(result)
			if !ok || res == nil {
				glog.Debug(ctx, "no result carrier")
				return
			}

			// 只对code为0的成功请求进行响应加签，不适合passthrough的返回
			codes := res.Metadata().Get(constant.BgwAPIResponseCodes)
			if len(codes) == 0 || codes[0] != "0" {
				return
			}

			data, err := res.GetData()
			if err != nil {
				glog.Error(ctx, "result carrier get data failed")
				return
			}
			outSign, err := getCipher(ctx).SignRespResult(otelCtx, data)
			if err != nil {
				glog.Error(ctx, "sign resp failed", glog.String("err", err.Error()))
				gmetric.IncDefaultError("crypter", "resp")
				respErr = berror.ErrBadSign
				return
			}
			glog.Debug(ctx, "resp outSign", glog.String("outSign", outSign))
			ctx.Response.Header.Set(cyrptionRespHeader, outSign)
		}()

		respErr = next(ctx)
		return
	}
}

// Init will init the filter
func (s *crypter) Init(ctx context.Context, args ...string) (err error) {
	_ = initCipher(ctx)

	if len(args) == 0 {
		return nil
	}

	p := flag.NewFlagSet("crypter", flag.ContinueOnError)
	p.BoolVar(&s.Req, "request", false, "request check sign")
	p.BoolVar(&s.Resp, "response", false, "resp add sign")

	if err = p.Parse(args[1:]); err != nil {
		glog.Error(ctx, "crypter parse error", glog.Any("args", args), glog.String("error", err.Error()))
	}
	return
}

type result interface {
	GetData() ([]byte, error)
	Metadata() gmd.MD
}
