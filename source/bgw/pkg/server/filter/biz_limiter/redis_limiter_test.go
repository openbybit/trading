package biz_limiter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/remoting/redis"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/test"
)

func TestDoLimit(t *testing.T) {
	lm := &rateLimiter{
		flags: flags{
			Limit: gredis.Limit{
				Rate: 0,
			},
		},
	}
	h := lm.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})
	assert.NotNil(t, h)
	assert.Equal(t, filter.BizRateLimitFilterKey, lm.GetName())
}

func TestParseFlag(t *testing.T) {
	rl := &rateLimiter{
		flags: flags{
			Limit: gredis.Limit{
				Rate: 0,
			},
		},
	}
	err := rl.parseFlags([]string{"", "--22=22"})
	assert.EqualError(t, err, "flag provided but not defined: -22")

	err = rl.parseFlags([]string{"limiter",
		"-burst=22", "-group=123", "-rate=22", "-period=30s",
		"-unified=true",
		"-hasSymbol=true", "-disableCustomRate=true"})
	assert.Equal(t, time.Second*30, rl.flags.Period)
	assert.Equal(t, true, rl.flags.hasSymbol)
	assert.Equal(t, true, rl.flags.disableCustomRate)
	assert.Equal(t, true, rl.flags.unified)
	assert.Equal(t, 22, rl.flags.Burst)
	assert.Equal(t, 22, rl.flags.Rate)
	assert.Equal(t, "123", rl.flags.group)

	err = rl.parseFlags([]string{"limiter",
		"-burst=22", "-group=123", "-rate=0"})
	assert.EqualError(t, err, "biz limit group is nil, but not default rate")
}

func TestInitLimitRedis(t *testing.T) {
	rl := &rateLimiter{
		flags: flags{
			Limit: gredis.Limit{
				Rate: 0,
			},
		},
	}
	err := rl.Init(context.Background())
	assert.EqualError(t, err, "invalid args, can't must filter")

	err = rl.Init(context.Background(), ",b,c,d,e,f,g")
	assert.EqualError(t, err, "invalid app name")

	err = rl.Init(context.Background(), "a.b.c.d.e.f.g", "--212=22")
	assert.EqualError(t, err, "flag provided but not defined: -212")

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(rl), "parseFlags", func(args []string) (err error) {
		return nil
	})
	err = rl.Init(context.Background(), "a.b.c.d.e.f.true", "--burst=22")
	assert.EqualError(t, err, "uta_engin should not use v1 redis limiter")
	p.Reset()

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(rl), "parseFlags", func(args []string) (err error) {
		// rl.flags.disableCustomRate = false
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(rl), "initLoaders", func(ctx context.Context, app string) error {
		return errors.New("xxx")
	})
	err = rl.Init(context.Background(), "a.b.c.d.e.f.false", "--burst=22")
	assert.EqualError(t, err, "xxx")
	p.Reset()

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(rl), "parseFlags", func(args []string) (err error) {
		rl.flags.disableCustomRate = true
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(rl), "initLoaders", func(ctx context.Context, app string) error {
		return errors.New("xxx")
	})
	err = rl.Init(context.Background(), "a.b.c.d.e.f.false", "--burst=22")
	assert.NoError(t, err)
	p.Reset()
}

func TestLimitRedis(t *testing.T) {
	rctx, md := test.NewReqCtx()
	lm := &rateLimiter{
		flags: flags{
			Limit: gredis.Limit{
				Rate: 0,
			},
		},
	}
	err := lm.Limit(rctx, md.Route, 123, "123")
	assert.EqualError(t, err, "redis group limiter rate is 0")

	rctx, md = test.NewReqCtx()
	lm = &rateLimiter{
		flags: flags{
			group: "123",
			Limit: gredis.Limit{
				Rate: 0,
			},
		},
	}
	err = lm.Limit(rctx, md.Route, 0, "123")
	assert.EqualError(t, err, "redis limiter memberID is 0, group mode")

	rctx, md = test.NewReqCtx()
	lm = &rateLimiter{
		flags: flags{
			hasSymbol: true,
			group:     "123",
			Limit: gredis.Limit{
				Rate: defaultRate + 10,
			},
		},
	}
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "doLimit", func(ctx *types.Ctx, key, limitType string, step int, limit gredis.Limit, withHeader bool) (err error) {
		return errors.New("fff")
	})

	err = lm.Limit(rctx, md.Route, 123, "123")
	assert.EqualError(t, err, "fff")

	p.ApplyPrivateMethod(reflect.TypeOf(lm), "doLimit", func(ctx *types.Ctx, key, limitType string, step int, limit gredis.Limit, withHeader bool) (err error) {
		return nil
	})

	lm.flags.unified = false
	err = lm.Limit(rctx, md.Route, 123, "123")
	assert.NoError(t, err)

	lm.flags.unified = true
	err = lm.Limit(rctx, md.Route, 123, "123")
	assert.NoError(t, err)

	p.Reset()
}

func TestSetHeaderRedis(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	setHeader(rctx, 1, &gredis.Result{Remaining: 1, Allowed: 1}, false)

	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))
	assert.Nil(t, rctx.UserValue(constant.BgwRateLimitInfo))

	setHeader(rctx, 1, &gredis.Result{Remaining: 10, Allowed: 1}, true)
	assert.Equal(t, "1", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "10", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.NotEqual(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))
	rr := rctx.UserValue(constant.BgwRateLimitInfo).(metadata.RateLimitInfo)
	assert.Equal(t, 1, rr.RateLimit)
	assert.Equal(t, 10, rr.RateLimitStatus)
	assert.NotEqual(t, 0, rr.RateLimitResetMs)

	setHeader(rctx, 1, &gredis.Result{Remaining: 0, Allowed: 0, ResetAfter: time.Duration(time.Now().UnixNano() + time.Second.Nanoseconds())}, true)
	assert.Equal(t, "1", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.NotEqual(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))

	rr = rctx.UserValue(constant.BgwRateLimitInfo).(metadata.RateLimitInfo)
	assert.Equal(t, 1, rr.RateLimit)
	assert.Equal(t, 0, rr.RateLimitStatus)
	assert.NotEqual(t, 0, rr.RateLimitResetMs)
}

func TestRateLimiter_Limit(t *testing.T) {
	Convey("test ratelimiter limit", t, func() {
		rl := &rateLimiter{}
		rl.flags.group = "mockGroup"
		rl.flags.disableCustomRate = false

		patch := gomonkey.ApplyFunc((*rateLimiter).getQuota, func(r *rateLimiter, ctx context.Context, app string, uid int64, group string) int {
			if app == "option" {
				return 0
			}
			return 10
		})
		defer patch.Reset()

		patch1 := gomonkey.ApplyFunc((*rateLimiter).doLimit, func(r *rateLimiter, ctx *types.Ctx, key string, limit gredis.Limit, platform string) error {
			return nil
		})
		defer patch1.Reset()

		err := rl.Limit(&types.Ctx{}, metadata.RouteKey{}, 123, "platform")
		So(err, ShouldBeNil)

		rl.flags.unified = true
		err = rl.Limit(&types.Ctx{}, metadata.RouteKey{}, 123, "platform")
		So(err, ShouldNotBeNil)
	})
}

func TestRateLimiter_blockTradeLimit(t *testing.T) {
	Convey("test blockTradeLimit", t, func() {
		rl := &rateLimiter{}
		ctx := &types.Ctx{}
		err := rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldNotBeNil)

		ctx.SetUserValue("blocktrade", &bizmetedata.BlockTrade{})
		patch := gomonkey.ApplyFunc((*rateLimiter).Limit, func(r *rateLimiter, c *types.Ctx, route metadata.RouteKey, memberID int64, platform string) (err error) {
			return errors.New("mock err")
		})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*rateLimiter).Limit, func(r *rateLimiter, c *types.Ctx, route metadata.RouteKey, memberID int64, platform string) (err error) {
			return nil
		})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldBeNil)

		ctx.SetUserValue("blocktrade", &bizmetedata.BlockTrade{MakerMemberId: 123})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldBeNil)
		patch.Reset()
	})
}

func TestRateLimiter_doLimit(t *testing.T) {
	Convey(" test doLimit", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.ObserveDefaultLatencySince, func(t time.Time, typ, label string) {})
		defer patch.Reset()
		patch1 := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
		defer patch1.Reset()

		rl := &rateLimiter{}

		patch2 := gomonkey.ApplyFunc(redis.AllowN, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return nil, errors.New("mock err")
		})
		err := rl.doLimit(&types.Ctx{}, "key", gredis.Limit{}, "platform")
		So(err, ShouldBeNil)
		patch2.Reset()

		patch2 = gomonkey.ApplyFunc(redis.AllowN, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return &gredis.Result{}, nil
		})
		err = rl.doLimit(&types.Ctx{}, "key", gredis.Limit{}, "platform")
		So(err, ShouldNotBeNil)
		patch2.Reset()

		patch2 = gomonkey.ApplyFunc(redis.AllowN, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return &gredis.Result{Allowed: 1}, nil
		})
		err = rl.doLimit(&types.Ctx{}, "key", gredis.Limit{}, "platform")
		So(err, ShouldBeNil)
		patch2.Reset()
	})
}

func TestRateLimiter_blockTradeLimit2(t *testing.T) {
	Convey("test blockTradeLimit", t, func() {
		rl := &rateLimiter{}
		ctx := &types.Ctx{}
		err := rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldNotBeNil)

		ctx.SetUserValue("blocktrade", &bizmetedata.BlockTrade{})
		patch := gomonkey.ApplyFunc((*rateLimiter).Limit, func(r *rateLimiter, c *types.Ctx, route metadata.RouteKey, memberID int64, platform string) (err error) {
			return errors.New("mock err")
		})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*rateLimiter).Limit, func(r *rateLimiter, c *types.Ctx, route metadata.RouteKey, memberID int64, platform string) (err error) {
			return nil
		})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldBeNil)

		ctx.SetUserValue("blocktrade", &bizmetedata.BlockTrade{MakerMemberId: 123})
		err = rl.blockTradeLimit(ctx, metadata.RouteKey{}, &metadata.Metadata{})
		So(err, ShouldBeNil)
		patch.Reset()
	})
}

func TestRateLimiter_Do(t *testing.T) {
	Convey("test ratelimiter do", t, func() {
		rl := &rateLimiter{}
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		next := func(ctx2 *types.Ctx) error {
			return nil
		}

		handler := rl.Do(next)
		err := handler(ctx)
		So(err, ShouldNotBeNil)

		md.Route = metadata.RouteKey{
			AppName:     "1",
			ModuleName:  "1",
			ServiceName: "1",
			Registry:    "1",
			MethodName:  "1",
			HttpMethod:  "1",
		}
		err = handler(ctx)
		So(err, ShouldNotBeNil)

		md.Route.ACL.Group = constant.ResourceGroupBlockTrade
		err = handler(ctx)
		So(err, ShouldNotBeNil)
	})
}

func TestRateLimiter_initLoaders(t *testing.T) {
	Convey("test ratelimiter initLoaders", t, func() {
		rl := &rateLimiter{}
		err := rl.initLoaders(context.Background(), "app1")
		So(err, ShouldBeNil)

		err = rl.initLoaders(context.Background(), "app1")
		So(err, ShouldBeNil)
	})
}
