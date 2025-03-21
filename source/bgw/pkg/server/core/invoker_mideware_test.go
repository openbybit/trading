package core

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/breaker"
	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	phttp "bgw/pkg/server/core/http"
	gmetadata "bgw/pkg/server/metadata"
)

func TestBreakerMidware(t *testing.T) {
	Convey("test BreakerMidware", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
		defer patch.Reset()
		next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
			if md.UID == 1 {
				res := phttp.NewResult()
				res.SetStatus(500)
				ctx.SetUserValue(constant.CtxInvokeResult, res)
			}

			if md.UID == 3 {
				return berror.NewBizErr(berror.UpstreamErrInvokerBreaker, "mock err")
			}

			return nil
		}

		bm := newBreakerMidware(breaker.NewBreakerMgr())
		invoke := bm.Do(next)

		Convey("test no breaker", func() {
			ctx := &types.Ctx{}
			route := &MethodConfig{
				service: &ServiceConfig{
					Registry: "service_1",
					Protocol: constant.HttpProtocol,
				},
			}
			md := gmetadata.MDFromContext(ctx)
			md.Method = "GET"
			md.Path = "/contract/symbols"
			md.UID = 1

			err := invoke(ctx, route, md)
			So(err, ShouldBeNil)
		})

		Convey("test http invoke", func() {
			ctx := &types.Ctx{}
			route := &MethodConfig{
				service: &ServiceConfig{
					Registry: "service_1",
					Protocol: constant.HttpProtocol,
				},
				Breaker: true,
			}
			md := gmetadata.MDFromContext(ctx)
			md.Method = "GET"
			md.Path = "/contract/symbols"
			md.UID = 2

			err := invoke(ctx, route, md)
			So(err, ShouldBeNil)

			md.UID = 1
			err = invoke(ctx, route, md)
			So(err, ShouldBeNil)
		})

		Convey("test grpc invoke", func() {
			ctx := &types.Ctx{}
			route := &MethodConfig{
				service: &ServiceConfig{
					Registry: "service_1",
					Package:  "option",
					Name:     "accountService",
				},
				Name:    "querySymbol",
				Breaker: true,
			}
			md := gmetadata.MDFromContext(ctx)
			md.UID = 3

			err := invoke(ctx, route, md)
			So(err, ShouldNotBeNil)

			for i := 0; i < 50; i++ {
				err = invoke(ctx, route, md)
				So(err, ShouldNotBeNil)
			}
		})
	})
}

func TestBreakerMidware_OnEvent(t *testing.T) {
	Convey("test BreakerMidware OnEvent", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
		defer patch.Reset()

		bm := &breakerMidware{}
		err := bm.OnEvent(nil)
		So(err, ShouldBeNil)

		event := &observer.DefaultEvent{}
		err = bm.OnEvent(event)
		So(err, ShouldBeNil)

		var v = `unify-dev-1: true
unify-test-1: true`

		event.Value = v
		err = bm.OnEvent(event)
		So(err, ShouldBeNil)

		bm.cluster = "unify-dev-1"
		err = bm.OnEvent(event)
		So(err, ShouldBeNil)

		event.Value = "!23"
		err = bm.OnEvent(event)
		So(err, ShouldBeNil)
	})
}

func TestNewBreakMidware(t *testing.T) {
	Convey("test newBreakerMidware", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string2 string, string3 string) {})
		defer patch.Reset()

		patch1 := gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(ctx context.Context, opts ...nacos.Options) (config_center.Configure, error) {
			return nil, errMock
		})
		bm := newBreakerMidware(breaker.NewBreakerMgr())
		So(bm, ShouldNotBeNil)
		patch1.Reset()

		ctrl := gomock.NewController(t)
		mockNacos := config_center.NewMockConfigure(ctrl)
		mockNacos.EXPECT().Listen(gomock.Any(), gomock.Any(), gomock.Any()).Return(errMock)
		patch1 = gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(ctx context.Context, opts ...nacos.Options) (config_center.Configure, error) {
			return mockNacos, nil
		})
		defer patch1.Reset()
		bm = newBreakerMidware(breaker.NewBreakerMgr())
		So(bm, ShouldNotBeNil)
	})
}

func TestAcceptable(t *testing.T) {
	Convey("test Acceptable", t, func() {
		res := acceptable(nil)
		So(res, ShouldBeTrue)

		res = acceptable(errors.New("mock err"))
		So(res, ShouldBeFalse)

		res = acceptable(berror.NewUpStreamErr(berror.UpstreamErrInvokerBreaker))
		So(res, ShouldBeFalse)

		res = acceptable(berror.NewUpStreamErr(berror.UpstreamErrInvokerFailed))
		So(res, ShouldBeTrue)

		res = acceptable(berror.NewBizErr(berror.TimeoutErr))
		So(res, ShouldBeFalse)
	})
}

var errMock = errors.New("mock err")

// 628 ns/op
// 没有报错
func BenchmarkBreakerMidware_HttpNoBreak(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Protocol: constant.HttpProtocol,
		},
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}

// 1418 ns/op
// 70% 报错
func BenchmarkBreakerMidware_HttpBreak(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		i := rand.Intn(10)
		if i < 7 {
			return errMock
		}
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Protocol: constant.HttpProtocol,
		},
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}

// 713 ns/op
// 30% 报错
func BenchmarkBreakerMidware_HttpNoBreak2(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		i := rand.Intn(10)
		if i < 3 {
			return errors.New("mock err")
		}
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Protocol: constant.HttpProtocol,
		},
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}

// 877 ns/op
// 没有报错
func BenchmarkBreakerMidware_GrpcNoBreak(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Package:  "option",
			Name:     "accountService",
		},
		Name:    "querySymbol",
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}

// 1602 ns/op
// 70% 报错
func BenchmarkBreakerMidware_GrpcBreak(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		i := rand.Intn(10)
		if i < 7 {
			return errMock
		}
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Package:  "option",
			Name:     "accountService",
		},
		Name:    "querySymbol",
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}

// 966 ns/op
// 30% 报错
func BenchmarkBreakerMidware_GrpcNoBreak2(b *testing.B) {
	patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
	defer patch.Reset()

	bgr := breaker.NewBreakerMgr()
	bgm := newBreakerMidware(bgr)
	next := func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		i := rand.Intn(10)
		if i < 3 {
			return errMock
		}
		return nil
	}
	invoke := bgm.Do(next)

	ctx := &types.Ctx{}
	route := &MethodConfig{
		service: &ServiceConfig{
			Registry: "service_1",
			Package:  "option",
			Name:     "accountService",
		},
		Name:    "querySymbol",
		Breaker: true,
	}
	md := gmetadata.MDFromContext(ctx)
	md.Method = "GET"
	md.Path = "/contract/symbols"
	md.UID = 2

	for i := 0; i < b.N; i++ {
		_ = invoke(ctx, route, md)
	}
}
