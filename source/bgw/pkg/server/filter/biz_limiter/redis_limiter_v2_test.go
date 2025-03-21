package biz_limiter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/future/sdk.git/pkg/scmeta"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common/berror"
	"bgw/pkg/remoting/redis"

	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/test"

	"github.com/tj/assert"
)

func TestLoadQuotaRedisV2(t *testing.T) {
	Init()
	lm := &rateLimiterV2{
		flags: flagsV2{
			unified: true,
		},
	}
	rctx, md := test.NewReqCtx()

	md.Route.AllApp = true
	p := gomonkey.ApplyFuncReturn(getAutoQuota, gredis.Limit{
		Rate:   1,
		Burst:  20,
		Period: 30,
	}, "123", nil)
	l, s, err := lm.loadQuota(rctx, md, "1", limitRule{})
	assert.NoError(t, err)
	assert.Equal(t, "123", s)
	assert.Equal(t, 20, l.Burst)
	assert.Equal(t, time.Duration(30), l.Period)
	assert.Equal(t, 1, l.Rate)
	p.Reset()
	md.Route.AllApp = false
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadRate", func(ctx *types.Ctx, app string, ru limitRule, unified bool, params *rateParams) (gredis.Limit, error) {
		return gredis.Limit{
			Rate:   1,
			Burst:  20,
			Period: 30,
		}, nil
	})
	l, s, err = lm.loadQuota(rctx, md, "1", limitRule{})
	assert.NoError(t, err)
	assert.Equal(t, "unified::0", s)
	assert.Equal(t, 20, l.Burst)
	assert.Equal(t, time.Duration(30), l.Period)
	assert.Equal(t, 1, l.Rate)

	p.Reset()

	lm.flags.unified = true
	md.Route.AllApp = false
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadRate", func(ctx *types.Ctx, app string, ru limitRule, unified bool, params *rateParams) (gredis.Limit, error) {
		return gredis.Limit{
			Rate:   1,
			Burst:  20,
			Period: 30,
		}, nil
	})
	l, s, err = lm.loadQuota(rctx, md, "1", limitRule{})
	assert.NoError(t, err)
	assert.Equal(t, "unified::0", s)
	assert.Equal(t, 20, l.Burst)
	assert.Equal(t, time.Duration(30), l.Period)
	assert.Equal(t, 1, l.Rate)

	p.Reset()

	lm.flags.unified = false
	md.Route.AllApp = false
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadRate", func(ctx *types.Ctx, app string, ru limitRule, unified bool, params *rateParams) (gredis.Limit, error) {
		return gredis.Limit{
			Rate:   1,
			Burst:  20,
			Period: 30,
		}, nil
	})
	l, s, err = lm.loadQuota(rctx, md, "1", limitRule{})
	assert.NoError(t, err)
	assert.Equal(t, "test", s)
	assert.Equal(t, 20, l.Burst)
	assert.Equal(t, time.Duration(30), l.Period)
	assert.Equal(t, 1, l.Rate)

	p.Reset()
}

func TestDoLimitV2(t *testing.T) {
	lm := &rateLimiterV2{
		flags: flagsV2{
			unified: true,
		},
	}
	h := lm.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})
	assert.NotNil(t, h)
}

func TestInitRulesRedisV2(t *testing.T) {
	lm := &rateLimiterV2{
		flags: flagsV2{
			unified: true,
		},
	}
	assert.Equal(t, filter.BizRateLimitFilterV2Key, lm.GetName())

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "parseFlags", func(args []string) (err error) {
		return errors.New("ddd")
	})
	err := lm.initRules(context.Background())
	assert.EqualError(t, err, "ddd")

	p.ApplyPrivateMethod(reflect.TypeOf(lm), "parseFlags", func(args []string) (err error) {
		return nil
	})
	lm.flags.rules = []limitRule{
		{
			DataProvider: futuresService,
			Category:     futuresService,
		},
		{
			DataProvider: optionService,
			Category:     optionService,
		},
	}

	p.ApplyFuncReturn(symbolconfig.InitSymbolConfig, errors.New("ddd"))

	err = lm.initRules(context.Background())
	assert.EqualError(t, err, "redis limiterV2 InitSymbolConfig error: ddd")

	p.ApplyFuncReturn(symbolconfig.InitSymbolConfig, nil)
	err = lm.initRules(context.Background())
	assert.NoError(t, err)
	p.Reset()
}

func TestLimitOneRedisV2(t *testing.T) {
	rctx, md := test.NewReqCtx()
	lm := &rateLimiterV2{
		flags: flagsV2{},
	}
	err := lm.limitOne(rctx, md)
	assert.NoError(t, err)
	assert.Equal(t, "", md.LimitRule)
	assert.Equal(t, 0, md.LimitValue)
	assert.Equal(t, int64(0), md.LimitPeriod)

	l := limitRule{Category: "test", DataProvider: "1", LimitType: "2"}
	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	rctx, md = test.NewReqCtx()

	p := gomonkey.ApplyMethodFunc(reflect.TypeOf(lm), "Limit", func(ctx *types.Ctx, md *metadata.Metadata,
		dataProvider, limitType string, rule limitRule) (err error) {
		assert.Equal(t, "1", dataProvider)
		assert.Equal(t, "2", limitType)
		assert.Equal(t, l, rule)
		return nil
	})

	err = lm.limitOne(rctx, md)
	assert.NoError(t, err)
	assert.Equal(t, "", md.LimitRule)
	assert.Equal(t, 0, md.LimitValue)
	assert.Equal(t, int64(0), md.LimitPeriod)
	p.Reset()
}

func TestLoadRateRedisV2(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	lm := &rateLimiterV2{
		flags: flagsV2{},
	}
	lll, err := lm.loadRate(rctx, "aa", limitRule{
		extractor: extractor{UID: false},
		Limit: gredis.Limit{
			Rate: 20,
		},
	}, true, &rateParams{})
	assert.Equal(t, 20, lll.Rate)
	assert.NoError(t, err)

	lll, err = lm.loadRate(rctx, "aa", limitRule{
		extractor: extractor{UID: true},
		Limit: gredis.Limit{
			Rate: 20,
		},
	}, true, &rateParams{})
	assert.Equal(t, 20, lll.Rate)
	assert.NoError(t, err)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "getQuota", func(ctx context.Context, app string, unified bool, params *rateParams) (int, error) {
		return 0, errors.New("fff")
	})
	lll, err = lm.loadRate(rctx, "aa", limitRule{
		extractor:        extractor{UID: true},
		EnableCustomRate: true,
		Limit: gredis.Limit{
			Rate: 20,
		},
	}, true, &rateParams{})
	assert.Equal(t, 20, lll.Rate)
	assert.EqualError(t, err, "fff")

	p.ApplyPrivateMethod(reflect.TypeOf(lm), "getQuota", func(ctx context.Context, app string, unified bool, params *rateParams) (int, error) {
		return 0, nil
	})
	lll, err = lm.loadRate(rctx, "aa", limitRule{
		extractor:        extractor{UID: true},
		EnableCustomRate: true,
		Limit: gredis.Limit{
			Rate: 0,
		},
	}, true, &rateParams{})
	assert.Equal(t, 0, lll.Rate)
	assert.EqualError(t, err, "redis limiterV2 v2 group rate is 0")
	p.ApplyPrivateMethod(reflect.TypeOf(lm), "getQuota", func(ctx context.Context, app string, unified bool, params *rateParams) (int, error) {
		return 30, nil
	})
	lll, err = lm.loadRate(rctx, "aa", limitRule{
		extractor:        extractor{UID: true},
		EnableCustomRate: true,
		Cap:              20,
		Limit: gredis.Limit{
			Rate: 40,
		},
	}, true, &rateParams{})
	assert.Equal(t, 20, lll.Rate)
	assert.NoError(t, err)
	p.Reset()

}

func TestLimitRedisV2(t *testing.T) {
	rctx, md := test.NewReqCtx()
	lm := &rateLimiterV2{
		flags: flagsV2{},
	}
	err := lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: true},
	})
	assert.EqualError(t, err, "redis_limiter_v2 uid is 0")
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{}, "", errors.New("nnn")
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.EqualError(t, err, "nnn")

	p.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{
			Rate:   0,
			Period: time.Second * 2,
		}, "key", nil
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.EqualError(t, err, "redis_limiter_v2 rate is 0, 123, 123")

	p.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{
			Rate:   defaultRate * 4,
			Period: time.Second * 2,
		}, "key", nil
	})
	p.ApplyPrivateMethod(reflect.TypeOf(lm), "doLimit", func(ctx *types.Ctx, key string, limit gredis.Limit, platform string) error {
		return errors.New("fff")
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.EqualError(t, err, "fff")
	p.ApplyPrivateMethod(reflect.TypeOf(lm), "doLimit", func(ctx *types.Ctx, key string, limit gredis.Limit, platform string) error {
		return nil
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.NoError(t, err)
	p.Reset()
}

func TestRedisLimiterParseFlags(t *testing.T) {
	lm := &rateLimiterV2{
		flags: flagsV2{},
	}

	err := lm.parseFlags([]string{"limiterMemo", "--212=22"})
	assert.EqualError(t, err, "flag provided but not defined: -212")

	err = lm.parseFlags([]string{"limiterMemo", "--rules=22"})
	assert.EqualError(t, err, "[]biz_limiter.limitRule: decode slice: expect [ or n, but found 2, error found in #1 byte of ...|22|..., bigger context ...|22|...")

	err = lm.parseFlags([]string{"limiterMemo", "--options=22"})
	assert.EqualError(t, err, "[]biz_limiter.options: decode slice: expect [ or n, but found 2, error found in #1 byte of ...|22|..., bigger context ...|22|...")

	err = lm.parseFlags([]string{"limiterMemo", "--rules=[{\"rate\": 0}]"})
	assert.EqualError(t, err, "rate is invalid, 0")

	err = lm.parseFlags([]string{"limiterMemo", "--rules=[{\"rate\": 10}]"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, 1, lm.flags.rules[0].Step)
	assert.Equal(t, int64(0), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second, lm.flags.rules[0].Period)

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--rules=[{"rate": 10, "period_sec":10}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, 1, lm.flags.rules[0].Step)
	assert.Equal(t, int64(10), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second*time.Duration(10), lm.flags.rules[0].Period)
	assert.Equal(t, "", lm.flags.rules[0].Category)
	assert.Equal(t, 10, lm.flags.rules[0].Rate)
	assert.Equal(t, 0, lm.flags.rules[0].Burst)
	assert.Equal(t, false, lm.flags.rules[0].UID)
	assert.Equal(t, false, lm.flags.rules[0].Path)
	assert.Equal(t, false, lm.flags.rules[0].Method)
	assert.Equal(t, false, lm.flags.rules[0].Symbol)
	assert.Equal(t, false, lm.flags.rules[0].EnableCustomRate)
	assert.Equal(t, "", lm.flags.rules[0].Group)
	assert.Equal(t, "", lm.flags.rules[0].DataProvider)
	assert.Equal(t, "", lm.flags.rules[0].LimitType)

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 0}]`})
	assert.EqualError(t, err, "rate is invalid, 0")
	assert.Equal(t, 0, len(lm.flags.rules))

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 10}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(10), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Duration(10)*time.Second, lm.flags.rules[0].Period)

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 0}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(0), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second, lm.flags.rules[0].Period)

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo",
		`--options=[{"rate":10,"periodSec":0,"category":"1","burst":1,"uid":true,"path":true,"method":true,"symbol":true,"enable_custom_rate":true,"group":"2","dataProvider":"futures","limitType":"counter"}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(0), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second, lm.flags.rules[0].Period)
	assert.Equal(t, "1", lm.flags.rules[0].Category)
	assert.Equal(t, 10, lm.flags.rules[0].Rate)
	assert.Equal(t, 1, lm.flags.rules[0].Burst)
	assert.Equal(t, 1, lm.flags.rules[0].Step)
	assert.Equal(t, true, lm.flags.rules[0].UID)
	assert.Equal(t, true, lm.flags.rules[0].Path)
	assert.Equal(t, true, lm.flags.rules[0].Method)
	assert.Equal(t, true, lm.flags.rules[0].Symbol)
	assert.Equal(t, true, lm.flags.rules[0].EnableCustomRate)
	assert.Equal(t, "2", lm.flags.rules[0].Group)
	assert.Equal(t, futuresService, lm.flags.rules[0].DataProvider)
	assert.Equal(t, limitCounter, lm.flags.rules[0].LimitType)

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 0,"dataProvider":"xxx"}]`})
	assert.EqualError(t, err, "redis limiterV2: unknown data provider: xxx")

	lm = &rateLimiterV2{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo",
		`--options=[{"rate":10,"periodSec":0,"category":"1","burst":1,"uid":true,"path":true,"method":true,"symbol":true,"enable_custom_rate":true,"group":"2","dataProvider":"futures","limitType":"xxx"}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(0), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second, lm.flags.rules[0].Period)
	assert.Equal(t, "1", lm.flags.rules[0].Category)
	assert.Equal(t, 10, lm.flags.rules[0].Rate)
	assert.Equal(t, 1, lm.flags.rules[0].Burst)
	assert.Equal(t, 1, lm.flags.rules[0].Step)
	assert.Equal(t, true, lm.flags.rules[0].UID)
	assert.Equal(t, true, lm.flags.rules[0].Path)
	assert.Equal(t, true, lm.flags.rules[0].Method)
	assert.Equal(t, true, lm.flags.rules[0].Symbol)
	assert.Equal(t, true, lm.flags.rules[0].EnableCustomRate)
	assert.Equal(t, "2", lm.flags.rules[0].Group)
	assert.Equal(t, futuresService, lm.flags.rules[0].DataProvider)
	assert.Equal(t, "", lm.flags.rules[0].LimitType)
}

func TestRateLimiterV2_limitone(t *testing.T) {
	Convey("test rateLimiterV2 limitone", t, func() {
		r := &rateLimiterV2{}
		r.flags.rules = append(r.flags.rules, limitRule{Category: "futures", DataProvider: "futures", LimitType: "bucket"})
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.Category = "futures"
		md.Route.AppName = "futures"
		err := r.limitOne(ctx, md)
		So(err, ShouldNotBeNil)
	})
}

func TestRateLimiterV2_doLimit(t *testing.T) {
	Convey("test rateLimiterV2 doLimit", t, func() {
		metricsFunc := gomonkey.ApplyFunc(gmetric.ObserveDefaultLatencySince, func(t time.Time, typ, label string) {
			return
		})
		defer metricsFunc.Reset()

		IncDefaultError := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {
			return
		})
		defer IncDefaultError.Reset()

		r := &rateLimiterV2{}
		ctx := &types.Ctx{}

		mockErr := errors.New("mock")
		AllowMFunc := gomonkey.ApplyFunc(redis.AllowM, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return nil, mockErr
		})
		err := r.doLimit(ctx, "test", limitCounter, 1, gredis.PerSecond(1), false)
		So(err, ShouldBeNil)
		AllowMFunc.Reset()

		AllowNFunc := gomonkey.ApplyFunc(redis.AllowN, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return nil, mockErr
		})
		err = r.doLimit(ctx, "test", selectTypeOne, 1, gredis.PerSecond(1), false)
		So(err, ShouldBeNil)
		AllowNFunc.Reset()

		applyFunc2 := gomonkey.ApplyFunc(redis.AllowM, func(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
			return &gredis.Result{
				Allowed: 0,
			}, nil
		})
		err = r.doLimit(ctx, "test", limitCounter, 1, gredis.PerSecond(1), false)
		So(err, ShouldEqual, berror.ErrVisitsLimit)
		applyFunc2.Reset()

	})
}

func TestRateLimiterV2_getQuota(t *testing.T) {
	Convey("test rateLimiterV2 getQuota", t, func() {
		r := &rateLimiterV2{}
		p := &rateParams{}
		p.dataProvider = futuresService
		_, err := r.getQuota(context.Background(), "futures", false, p)
		So(err, ShouldNotBeNil)

		p.dataProvider = "otherService"
		_, err = r.getQuota(context.Background(), optionService, false, p)
		So(err, ShouldBeNil)

		_, err = r.getQuota(context.Background(), "spot", false, p)
		So(err, ShouldBeNil)
	})
}

func TestRateLimiterV2_loadQuota(t *testing.T) {
	Convey("test loadQuota", t, func() {
		rl2 := &rateLimiterV2{}
		ctx := &types.Ctx{}
		md := &metadata.Metadata{}
		rule := limitRule{}
		rule.Symbol = true
		_, _, err := rl2.loadQuota(ctx, md, "dataProvider", rule)
		So(err, ShouldBeNil)

		patch0 := gomonkey.ApplyFunc(symbolconfig.GetSymbol, func(c *types.Ctx) string {
			return "BTCUSDT"
		})
		defer patch0.Reset()
		patch := gomonkey.ApplyFunc(symbolconfig.GetSymbolModule, func() (scmeta.Module, error) {
			return nil, errors.New("mock err")
		})
		_, _, err = rl2.loadQuota(ctx, md, "dataProvider", rule)
		So(err, ShouldBeNil)
		patch.Reset()
	})
}

func TestRateLimiterV2_Do(t *testing.T) {
	Convey("test RateLimiterV2 Do", t, func() {
		next := func(*types.Ctx) error {
			return nil
		}

		v2 := &rateLimiterV2{}
		v2.flags.selectType = selectTypeOne
		handler := v2.Do(next)

		patch := gomonkey.ApplyFunc((*rateLimiterV2).limitOne, func(r *rateLimiterV2, ctx *types.Ctx, md *metadata.Metadata) (err error) {
			return errors.New("mock err")
		})
		err := handler(&types.Ctx{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*rateLimiterV2).limitOne, func(r *rateLimiterV2, ctx *types.Ctx, md *metadata.Metadata) (err error) {
			return nil
		})
		err = handler(&types.Ctx{})
		So(err, ShouldBeNil)
		patch.Reset()

		v2.flags.rules = []limitRule{
			{},
			{
				DataProvider: "dp1",
				LimitType:    "lt1",
				Limit:        gredis.Limit{Rate: 10},
			},
		}
		v2.flags.selectType = ""
		handler = v2.Do(next)

		patch = gomonkey.ApplyFunc((*rateLimiterV2).Limit, func(r *rateLimiterV2, ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
			return errors.New("mock err")
		})
		err = handler(&types.Ctx{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*rateLimiterV2).Limit, func(r *rateLimiterV2, ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
			return nil
		})
		err = handler(&types.Ctx{})
		So(err, ShouldBeNil)
		patch.Reset()
	})
}

func TestRateLimiterV2_Init(t *testing.T) {
	Convey("test RateLimiterV2 init", t, func() {
		rl2 := &rateLimiterV2{}
		err := rl2.Init(context.Background())
		So(err, ShouldNotBeNil)

		args := []string{"", "-empty=1"}
		err = rl2.Init(context.Background(), args...)
		So(err, ShouldNotBeNil)

		patch := gomonkey.ApplyFunc((*rateLimiterV2).initRules, func(r *rateLimiterV2, ctx context.Context, args ...string) (err error) {
			return nil
		})
		defer patch.Reset()

		rl2.flags.rules = []limitRule{
			{EnableCustomRate: true},
		}

		err = rl2.Init(context.Background(), args...)
		So(err, ShouldNotBeNil)
	})
}

func TestRateLimiterV2_InitLoaders(t *testing.T) {
	Convey("test RateLimiterV2 InitLoaders", t, func() {
		rl2 := &rateLimiterV2{}
		err := rl2.initLoaders(context.Background(), "")
		So(err, ShouldNotBeNil)

		err = rl2.initLoaders(context.Background(), "AppName.ModuleName.ServiceName.Registry.MethodName.HttpMethod.false")
		So(err, ShouldBeNil)

		err = rl2.initLoaders(context.Background(), "AppName.ModuleName.ServiceName.Registry.MethodName.HttpMethod.true")
		So(err, ShouldBeNil)
	})
}
