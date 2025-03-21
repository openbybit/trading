package manual_intervent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bgw/pkg/common/types"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	"bgw/pkg/registry"
	"bgw/pkg/server/filter/biz_limiter/rate"
	"bgw/pkg/server/metadata"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/armon/go-radix"
	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"
)

func TestInterveneLimiter_Init(t *testing.T) {
	convey.Convey("TestInterveneLimiter_Init", t, func() {
		gmetric.Init("test")
		il := &InterveneLimiter{}
		il.Init(context.Background())
		convey.So(il.pathRadixTree, convey.ShouldNotBeNil)
	})
}

func TestInterveneLimiter_doInit(t *testing.T) {
	convey.Convey("TestInterveneLimiter_doInit with error", t, func() {
		limiter := &InterveneLimiter{}
		applyFunc := gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(ctx context.Context, opts ...nacos.Options) (config_center.Configure, error) {
			return nil, errors.New("mock error")
		})
		convey.So(limiter.doInit(context.Background()), convey.ShouldNotBeNil)
		applyFunc.Reset()

		ctrl := gomock.NewController(t)
		mockNacos := config_center.NewMockConfigure(ctrl)
		mockNacos.EXPECT().Listen(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("mock error")).AnyTimes()

		patch := gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(ctx context.Context, opts ...nacos.Options) (config_center.Configure, error) {
			return mockNacos, nil
		})
		defer patch.Reset()
		limiter = &InterveneLimiter{}
		convey.So(limiter.doInit(context.Background()), convey.ShouldBeNil)
	})
}

func TestInterveneLimiter_OnEvent(t *testing.T) {
	convey.Convey("TestInterveneLimiter_OnEvent", t, func() {
		//empty event
		convey.So((&InterveneLimiter{}).OnEvent(&registry.ServiceInstancesChangedEvent{}), convey.ShouldNotBeNil)

		//wrong json format
		convey.So((&InterveneLimiter{}).OnEvent(&observer.DefaultEvent{Value: "{"}), convey.ShouldNotBeNil)

		//wrong ruleType format
		convey.So((&InterveneLimiter{}).OnEvent(&observer.DefaultEvent{Value: "{\n\t\"enable\": true,\n\t\"rules\": [{\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"fff\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"clientIp\": \"192.178.32.11\"\n\t\t}, {\n\t\t\t\"clientIp\": \"192.181.2.11\"\n\t\t}],\n\t\t\"limit\": 10\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"requestHost\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"requestHost\": \"api.bybit.com\"\n\t\t}],\n\t\t\"limit\": 13\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"clientOpFrom\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"clientOpFrom\": \"api\"\n\t\t}],\n\t\t\"limit\": 10\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"requestUrl\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v5/order/create\",\n\t\t\t\t\"httpMethod\": \"GET\",\n\t\t\t\t\"limit\": 10\n\t\t\t}\n\t\t}, {\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v5/order/cancel\",\n\t\t\t\t\"httpMethod\": \"POST\",\n\t\t\t\t\"limit\": 20\n\t\t\t}\n\t\t}, {\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v3/\",\n\t\t\t\t\"httpMethod\": \"POST\",\n\t\t\t\t\"limit\": 10\n\t\t\t}\n\t\t},{\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/contract/v3/private/order/\",\n\t\t\t\t\"httpMethod\": \"*\",\n\t\t\t\t\"limit\": 0\n\t\t\t}\n\t\t}],\n\t\t\"limit\": 10\n\t}]\n}"}), convey.ShouldBeNil)

		//correct default event
		interveneLimiter := &InterveneLimiter{}
		interveneLimiter.doInit(context.Background())
		observerDefaultEvent := &observer.DefaultEvent{Value: "{\n\t\"enable\": true,\n\t\"rules\": [{\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"clientIp\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"clientIp\": \"192.178.32.11\"\n\t\t}, {\n\t\t\t\"clientIp\": \"192.181.2.11\"\n\t\t}],\n\t\t\"limit\": 10\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"requestHost\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"requestHost\": \"api.bybit.com\"\n\t\t}],\n\t\t\"limit\": 13\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"clientOpFrom\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"clientOpFrom\": \"api\"\n\t\t}],\n\t\t\"limit\": 10\n\t}, {\n\t\t\"enable\": true,\n\t\t\"ruleType\": \"requestUrl\",\n\t\t\"effectiveEnvName\": \"unify-test-1\",\n\t\t\"effectPeriod\": {\n\t\t\t\"startDate\": \"2023-10-16 00:00:00\",\n\t\t\t\"endDate\": \"2023-10-21 23:59:59\"\n\t\t},\n\t\t\"extData\": [{\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v5/order/create\",\n\t\t\t\t\"httpMethod\": \"GET\",\n\t\t\t\t\"limit\": 10\n\t\t\t}\n\t\t}, {\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v5/order/cancel\",\n\t\t\t\t\"httpMethod\": \"POST\",\n\t\t\t\t\"limit\": 20\n\t\t\t}\n\t\t}, {\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/v3/\",\n\t\t\t\t\"httpMethod\": \"POST\",\n\t\t\t\t\"limit\": 10\n\t\t\t}\n\t\t},{\n\t\t\t\"requestUrl\": {\n\t\t\t\t\"path\": \"/contract/v3/private/order/\",\n\t\t\t\t\"httpMethod\": \"*\",\n\t\t\t\t\"limit\": 0\n\t\t\t}\n\t\t}],\n\t\t\"limit\": 10\n\t}]\n}"}
		convey.So(interveneLimiter.OnEvent(observerDefaultEvent), convey.ShouldBeNil)

	})
}

func TestvalidConfig(t *testing.T) {
	convey.Convey("TestvalidConfig", t, func() {
		gmetric.Init("bgw")

		//illegal ruleType
		cfg := &config{
			Enable: true,
			Rules: []*rule{
				{
					Enable:           false,
					RuleType:         clientIpRuleType + "_illegal",
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-15 00:00:00", EndDateInUTC: "2023-10-21 23:59:59"},
					ExtData:          []*extData{{ClientIp: ""}},
					Limit:            10,
				},
			},
		}
		convey.So(cfg.validConfig(), convey.ShouldBeFalse)

		//invalid effectPeriod
		cfg = &config{
			Enable: true,
			Rules: []*rule{
				{
					Enable:           false,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-14", EndDateInUTC: "2023-10-03 23:59:59"},
					ExtData:          []*extData{{ClientIp: ""}},
					Limit:            10,
				},
			},
		}
		convey.So(cfg.validConfig(), convey.ShouldBeFalse)

		//invalid extData
		cfg = &config{
			Enable: true,
			Rules: []*rule{
				{
					Enable:           false,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-14 00:00:00", EndDateInUTC: "2023-10-25 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1/24"}},
					Limit:            10,
				},
			},
		}
		convey.So(cfg.validConfig(), convey.ShouldBeFalse)

		//invalid limit
		cfg = &config{
			Enable: true,
			Rules: []*rule{
				{
					Enable:           false,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-14 00:00:00", EndDateInUTC: "2023-10-25 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
					Limit:            -1,
				},
			},
		}
		convey.So(cfg.validConfig(), convey.ShouldBeFalse)

		cfg = &config{
			Enable: true,
			Rules: []*rule{
				{
					Enable:           true,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2023-10-21 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
					Limit:            10,
				},
			},
		}
		convey.So(cfg.validConfig(), convey.ShouldBeTrue)
	})
}

func TestInterveneLimiter_loadRules(t *testing.T) {
	convey.Convey("TestInterveneLimiter_loadRules", t, func() {
		limiter := &InterveneLimiter{}
		limiter.doInit(context.Background())
		limiter.loadRules(&config{
			Enable: false,
		})
		convey.So(limiter.limiterCacheMap, convey.ShouldBeNil)

		applyFunc := gomonkey.ApplyFunc(env.ProjectEnvName, func() string {
			return "bybit-test-1"
		})
		defer applyFunc.Reset()

		limiter.loadRules(&config{
			Enable: true,
			Rules: []*rule{{ //invalid effectiveEnvName
				Enable:           true,
				RuleType:         clientIpRuleType,
				EffectiveEnvName: "",
				EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2023-10-16 23:59:59"},
				ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
				Limit:            10,
			}},
		})

		config := &config{
			Enable: true,
			Rules: []*rule{
				{ // invalid period
					Enable:           true,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2023-10-16 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
					Limit:            10,
				},
				{
					//unEnable config
					Enable:           false,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2023-10-16 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
					Limit:            10,
				}, {
					//valid client_ip config
					Enable:           true,
					RuleType:         clientIpRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{ClientIp: "127.0.0.1"}},
					Limit:            10,
				},
				{
					//valid host config
					Enable:           true,
					RuleType:         requestHostRuleType,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{RequestHost: "127.0.0.1"}},
					Limit:            10,
				}, {
					//valid op_from config
					Enable:           true,
					RuleType:         clientOpFromRule,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{ClientOpFrom: "pcweb"}},
					Limit:            10,
				}, {
					//valid url config
					Enable:           true,
					RuleType:         requestUrlRule,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{RequestUrl: requestUrl{Path: "/v5/order/create", HttpMethod: "GET", Limit: 10}}},
					Limit:            10,
				},
				{
					//valid url config
					Enable:           true,
					RuleType:         requestUrlRule,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{RequestUrl: requestUrl{Path: "/v5/order/create", HttpMethod: "GET", Limit: -1}}},
					Limit:            10,
				},
				{
					//valid url config
					Enable:           true,
					RuleType:         requestUrlRule,
					EffectiveEnvName: "unify-test-1,bybit-test-1",
					EffectPeriod:     &StandardPeriod{StartDateInUTC: "2023-10-16 00:00:00", EndDateInUTC: "2039-10-16 23:59:59"},
					ExtData:          []*extData{{RequestUrl: requestUrl{Path: "/v5/order/create", HttpMethod: "*", Limit: -1}}},
					Limit:            10,
				},
			},
		}
		limiter.loadRules(config)
		convey.So(limiter.limiterCacheMap, convey.ShouldNotBeNil)
		convey.So(len(limiter.limiterCacheMap), convey.ShouldEqual, 9)
		convey.So(len(limiter.enableRuleTypesInOrder), convey.ShouldEqual, 4)
	})
}

func TestInterveneLimiter_Intervene(t *testing.T) {
	convey.Convey("TestInterveneLimiter_Intervene", t, func() {
		il := &InterveneLimiter{}
		ctx := &types.Ctx{}
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)

		_ = metadata.MDFromContext(ctx)
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)

		md := metadata.MDFromContext(ctx)
		md.Extension.RemoteIP = "127.0.0.1"
		md.Extension.Host = "127.0.0.1:8080"
		md.Extension.OpFrom = "pcweb"
		md.Method = "GET"
		md.Path = "/v5/order/create"
		//il.enableRuleTypeSet = container.NewSet(clientIpRuleType, requestHostRuleType, clientOpFromRule, requestUrlRule)

		//intervene ip
		il.enableRuleTypesInOrder = []string{clientIpRuleType, "illegalRuleType"}
		cacheKey := buildCacheKey(clientIpRuleType, md.GetClientIP())
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   clientIpRuleType,
				limit:      rate.NewLimiter(rate.Limit(10), 10)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)

		//limit 0,interven ip
		il.limiterCacheMap = map[string]limiter{buildCacheKey(clientIpRuleType, md.GetClientIP()): {startInUTC: time.Now(), endInUTC: time.Now().Add(time.Minute), ruleType: clientIpRuleType,
			limit: rate.NewLimiter(rate.Limit(0), 0)}}
		convey.So(il.Intervene(ctx), convey.ShouldBeTrue)

		//limit 0,interven ip, invalid period
		il.limiterCacheMap = map[string]limiter{buildCacheKey(clientIpRuleType, md.GetClientIP()): {startInUTC: time.Now().Add(-2 * time.Minute), endInUTC: time.Now().Add(-5 * time.Minute * 2), ruleType: clientIpRuleType,
			limit: rate.NewLimiter(rate.Limit(0), 0)}}
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)

		il.enableRuleTypesInOrder = []string{requestHostRuleType}
		cacheKey = buildCacheKey(requestHostRuleType, md.Extension.Host)
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   requestHostRuleType,
				limit:      rate.NewLimiter(rate.Limit(10), 10)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   requestHostRuleType,
				limit:      rate.NewLimiter(rate.Limit(0), 0)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeTrue)

		il.enableRuleTypesInOrder = []string{clientOpFromRule}
		cacheKey = buildCacheKey(clientOpFromRule, md.Extension.OpFrom)
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   clientOpFromRule,
				limit:      rate.NewLimiter(rate.Limit(10), 10)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeFalse)

		il.enableRuleTypesInOrder = []string{clientOpFromRule}
		cacheKey = buildCacheKey(clientOpFromRule, md.Extension.OpFrom)
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   clientOpFromRule,
				limit:      rate.NewLimiter(rate.Limit(10), 0)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeTrue)

		il.enableRuleTypesInOrder = []string{requestUrlRule}
		cacheKey = buildRequestUrlCacheKey(requestUrlRule, md.Method, md.Path, nil)
		il.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(),
				endInUTC:   time.Now().Add(time.Minute),
				ruleType:   requestUrlRule,
				limit:      rate.NewLimiter(rate.Limit(0), 0)},
		}
		convey.So(il.Intervene(ctx), convey.ShouldBeTrue)

	})
}

func TestInterveneLimiter_buildRequestUrlCacheKey(t *testing.T) {
	convey.Convey("TestInterveneLimiter_buildRequestUrlCacheKey", t, func() {
		pathRadixTree := radix.New()
		key := buildRequestUrlCacheKey(requestUrlRule, "GET", "/v3/order/create", pathRadixTree)
		expectedKey := fmt.Sprintf(requestUrlCacheKeyFormat, requestUrlRule, "GET", "/v3/order/create")
		convey.So(key, convey.ShouldEqual, expectedKey)

		pathRadixTree.Insert("/v3/order", 1)
		key = buildRequestUrlCacheKey(requestUrlRule, "GET", "/v3/order/create", pathRadixTree)
		expectedKey = fmt.Sprintf(requestUrlCacheKeyFormat, requestUrlRule, "GET", "/v3/order")
		convey.So(key, convey.ShouldEqual, expectedKey)
	})
}

func TestInterveneLimiter_cleanCache(t *testing.T) {
	convey.Convey("TestInterveneLimiter_cleanCache", t, func() {
		l := InterveneLimiter{}
		l.startPeriodicCacheClean()
	})
}
func TestInterveneLimiter_doCleanCache(t *testing.T) {
	convey.Convey("TestInterveneLimiter_cleanCache", t, func() {
		l := InterveneLimiter{}
		cacheKey := buildCacheKey(clientIpRuleType, "127.0.0.1")
		cacheKey2 := buildCacheKey(clientIpRuleType, "127.0.0.2")

		l.limiterCacheMap = map[string]limiter{
			cacheKey: {
				startInUTC: time.Now(), endInUTC: time.Now().Add(time.Minute), ruleType: clientIpRuleType,
				limit: rate.NewLimiter(rate.Limit(0), 0),
			},
			cacheKey2: {
				startInUTC: time.Now(), endInUTC: time.Now().Add(-time.Minute), ruleType: clientIpRuleType,
				limit: rate.NewLimiter(rate.Limit(0), 0),
			},
		}

		l.doCleanCache()
	})
}
