package biz_limiter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter/biz_limiter/rate"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/test"
)

func TestDoMemo(t *testing.T) {
	l := newLimiterMemo().(*limiterMemo)
	h := l.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})
	assert.NotNil(t, h)
}

func TestInitLimitMemo(t *testing.T) {
	l := newLimiterMemo().(*limiterMemo)
	err := l.Init(context.Background())
	assert.EqualError(t, err, "invalid args, can't must filter")

	err = l.Init(context.Background(), "limiterMemo", "--212=22")
	assert.EqualError(t, err, "flag provided but not defined: -212")

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(l), "initRules", func(ctx context.Context, args ...string) (err error) {
		l.flags.rules = []limitRule{
			{
				EnableCustomRate: true,
			},
		}
		return nil
	})
	err = l.Init(context.Background(), "limiterMemo", "--rules=[{\"rate\": 10}]")
	assert.EqualError(t, err, "invalid app name")
	p.Reset()

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(l), "initRules", func(ctx context.Context, args ...string) (err error) {
		l.flags.rules = []limitRule{
			{
				EnableCustomRate: false,
			},
		}
		return nil
	})
	err = l.Init(context.Background(), "limiterMemo", "--rules=[{\"rate\": 10}]")
	assert.NoError(t, err)
	p.Reset()

}

func TestLoadQuota(t *testing.T) {
	lm := &limiterMemo{
		flags: flagsV2{
			unified: true,
		},
	}
	rctx, md := test.NewReqCtx()
	md.Route.AllApp = false
	_, s, err := lm.loadQuota(rctx, md, "1", limitRule{})
	assert.EqualError(t, err, "limiter memo not support")
	assert.Equal(t, "", s)

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
}

func TestInitRulesMem(t *testing.T) {
	lm := &limiterMemo{
		flags: flagsV2{
			unified: true,
		},
	}
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
	assert.EqualError(t, err, "redis limiterMemo InitSymbolConfig error: ddd")

	p.ApplyFuncReturn(symbolconfig.InitSymbolConfig, nil)
	err = lm.initRules(context.Background())
	assert.NoError(t, err)
	p.Reset()
}

func TestLimitMemo(t *testing.T) {
	rctx, md := test.NewReqCtx()
	lm := &limiterMemo{
		flags:    flagsV2{},
		limiters: make(map[string]*rate.Limiter),
	}

	err := lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: true},
	})
	assert.EqualError(t, err, "limiter_v3 uid is 0")

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{}, "", errors.New("nnn")
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.EqualError(t, err, "nnn")

	p = p.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{
			Rate: 0,
		}, "", nil
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.EqualError(t, err, "redis_limiter_v2 rate is 0, 123, 123")

	p.Reset()

	rctx, md = test.NewReqCtx()
	lm = &limiterMemo{
		flags: flagsV2{
			batch: true,
		},
		limiters:  make(map[string]*rate.Limiter),
		setHeader: true,
	}
	md.Route.AppName = "futures"
	p2 := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lm), "loadQuota", func(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
		return gredis.Limit{
			Rate:   defaultRate * 3,
			Burst:  0,
			Period: time.Second * 2,
		}, "key", nil
	})
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.Equal(t, "key", md.LimitRule)
	assert.Equal(t, 20000, md.LimitValue)
	assert.Equal(t, int64(0), md.LimitPeriod)

	p2.ApplyFuncReturn(rate.NewLimiter, rate.NewLimiter(rate.Limit(0), 0))
	lll := lm.limiters["key"]
	lll.SetLimit(0)
	lll.SetBurst(0)
	err = lm.Limit(rctx, md, "123", "123", limitRule{
		extractor: extractor{UID: false},
	})
	assert.Equal(t, berror.ErrVisitsLimit, err)

	p2.Reset()
}

func TestParseFlags(t *testing.T) {
	lm := &limiterMemo{
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

	lm = &limiterMemo{
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

	lm = &limiterMemo{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 0}]`})
	assert.EqualError(t, err, "rate is invalid, 0")
	assert.Equal(t, 0, len(lm.flags.rules))

	lm = &limiterMemo{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 10}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(10), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Duration(10)*time.Second, lm.flags.rules[0].Period)

	lm = &limiterMemo{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 0}]`})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lm.flags.rules))
	assert.Equal(t, int64(0), lm.flags.rules[0].PeriodSec)
	assert.Equal(t, time.Second, lm.flags.rules[0].Period)

	lm = &limiterMemo{
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

	lm = &limiterMemo{
		flags: flagsV2{},
	}
	err = lm.parseFlags([]string{"limiterMemo", `--options=[{"rate": 10, "periodSec": 0,"dataProvider":"xxx"}]`})
	assert.EqualError(t, err, "redis limiterMemo: unknown data provider: xxx")

	lm = &limiterMemo{
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

func TestSetHeaderMemo(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	setHeaderMemo(rctx, 1, 1, false)

	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.Equal(t, "", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))
	assert.Nil(t, rctx.UserValue(constant.BgwRateLimitInfo))

	setHeaderMemo(rctx, 1, 10, true)
	assert.Equal(t, "1", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "10", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.NotEqual(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))
	rr := rctx.UserValue(constant.BgwRateLimitInfo).(metadata.RateLimitInfo)
	assert.Equal(t, 1, rr.RateLimit)
	assert.Equal(t, 10, rr.RateLimitStatus)
	assert.NotEqual(t, 0, rr.RateLimitResetMs)

	setHeaderMemo(rctx, 1, 0, true)
	assert.Equal(t, "1", string(rctx.Response.Header.Peek(constant.HeaderAPILimit)))
	assert.Equal(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitStatus)))
	assert.NotEqual(t, "0", string(rctx.Response.Header.Peek(constant.HeaderAPILimitResetTimestamp)))

	rr = rctx.UserValue(constant.BgwRateLimitInfo).(metadata.RateLimitInfo)
	assert.Equal(t, 1, rr.RateLimit)
	assert.Equal(t, 0, rr.RateLimitStatus)
	assert.NotEqual(t, 0, rr.RateLimitResetMs)
}

func TestLimitOne(t *testing.T) {
	lm := &limiterMemo{
		flags: flagsV2{},
	}
	rctx, md := test.NewReqCtx()
	err := lm.limitOne(rctx, md)
	assert.NoError(t, err)
	assert.Equal(t, "", md.LimitRule)
	assert.Equal(t, 0, md.LimitValue)
	assert.Equal(t, int64(0), md.LimitPeriod)

	l := limitRule{Category: "test", DataProvider: "1", LimitType: "2"}
	lm = &limiterMemo{
		flags: flagsV2{
			rules: []limitRule{l},
		},
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

func TestLimiterMemo_Do(t *testing.T) {
	Convey("test RateLimiterV2 Do", t, func() {
		next := func(*types.Ctx) error {
			return nil
		}

		v2 := &limiterMemo{}
		v2.flags.selectType = selectTypeOne
		handler := v2.Do(next)

		patch := gomonkey.ApplyFunc((*limiterMemo).limitOne, func(r *limiterMemo, ctx *types.Ctx, md *metadata.Metadata) (err error) {
			return errors.New("mock err")
		})
		err := handler(&types.Ctx{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*limiterMemo).limitOne, func(r *limiterMemo, ctx *types.Ctx, md *metadata.Metadata) (err error) {
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

		patch = gomonkey.ApplyFunc((*limiterMemo).Limit, func(r *limiterMemo, ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
			return errors.New("mock err")
		})
		err = handler(&types.Ctx{})
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*limiterMemo).Limit, func(r *limiterMemo, ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
			return nil
		})
		err = handler(&types.Ctx{})
		So(err, ShouldBeNil)
		patch.Reset()
	})
}

func TestLimiterMemo_InitLoaders(t *testing.T) {
	Convey("test LimiterMemo InitLoaders", t, func() {
		rl2 := &limiterMemo{}
		err := rl2.initLoaders(context.Background(), "")
		So(err, ShouldNotBeNil)

		err = rl2.initLoaders(context.Background(), "AppName.ModuleName.ServiceName.Registry.MethodName.HttpMethod.false")
		So(err, ShouldBeNil)

		err = rl2.initLoaders(context.Background(), "AppName.ModuleName.ServiceName.Registry.MethodName.HttpMethod.true")
		So(err, ShouldBeNil)
	})
}

// func TestLimiterMemo_ParseFlags(t *testing.T) {
// 	Convey("test LimiterMemo parseFlags", t, func() {
// 		lm := &limiterMemo{
// 			limiters: make(map[string]*rate.Limiter),
// 		}
//
// 		err := lm.parseFlags([]string{"--selectType=one", `--options=[
//                         {
//                             "category": "spot",
//                             "rate": 20,
//                             "uid":true,
//                             "method":true,
//                             "path":true,
//                             "enable_custom_rate":true
//                         },
//                         {
//                             "category": "option",
//                             "rate": 1,
//                             "uid":true,
//                             "group":"cancelAll",
//                             "enable_custom_rate":true
//                         },
//                         {
//                             "category": "futures",
//                             "rate": 10,
//                             "uid":true,
//                             "path":true,
//                             "group":"order",
//                             "enable_custom_rate":true
//                         }]`})
// 		So(err, ShouldBeNil)
// 	})
// }

//
// func TestLimit(t *testing.T) {
//	lm := &limiterMemo{
//		flags: flagsV2{},
//	}
//	rctx, md := test.NewReqCtx()
//	err := lm.Limit(rctx, md, "ss", "ss", limitRule{})
//	assert.EqualError(t, err, "limiter_v3 uid is 0")
// }
