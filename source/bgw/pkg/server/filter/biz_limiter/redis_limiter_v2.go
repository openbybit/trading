package biz_limiter

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"git.bybit.com/svc/mod/pkg/bplatform"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/remoting/redis"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service"
	"bgw/pkg/service/symbolconfig"
)

const (
	futuresService = "futures" // mixer
	optionService  = "option"
	limitCounter   = "counter"
	selectTypeOne  = "one"
)

type flagsV2 struct {
	rules        []limitRule // multiple limit rules
	dataProvider string      // rules data provider
	limitType    string      // counter,bucket
	unified      bool        // unified account
	selectType   string      // one, all
	batch        bool        // batch
}

type limitRule struct {
	gredis.Limit
	extractor
	Category         string `json:"category"`           // futures,option,spot
	PeriodSec        int64  `json:"period_sec"`         // 1s, 1m, 1h, default second
	Step             int    `json:"step"`               // increase step
	EnableCustomRate bool   `json:"enable_custom_rate"` // related to data provider, otherwise use default rate
	DataProvider     string `json:"data"`               // data provider, default is futures
	LimitType        string `json:"type"`               // counter,bucket
	Cap              int    `json:"cap"`                // upper limit of quota
}

type rateLimiterV2 struct {
	limiter *gredis.Limiter
	flags   flagsV2
}

func newV2() filter.Filter {
	client := redis.NewClient()
	if client == nil {
		panic("connect redis error")
	}

	return &rateLimiterV2{
		limiter: gredis.NewLimiter(client),
	}
}

// GetName get name
func (r *rateLimiterV2) GetName() string {
	return filter.BizRateLimitFilterV2Key
}

// Do do redis limit v2
func (r *rateLimiterV2) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		var (
			err error
			md  = metadata.MDFromContext(ctx)
		)

		if r.flags.selectType == selectTypeOne {
			if err = r.limitOne(ctx, md); err != nil {
				return err
			}

			return next(ctx)
		}

		// As soon as one current limiterV2 gets rejected, return
		for _, rule := range r.flags.rules {
			if rule.IsZero() {
				continue
			}

			dataProvider := r.flags.dataProvider
			if rule.DataProvider != "" {
				dataProvider = rule.DataProvider
			}

			limitType := r.flags.limitType
			if rule.LimitType != "" {
				limitType = rule.LimitType
			}

			err = r.Limit(ctx, md, dataProvider, limitType, rule)
			if err != nil {
				return err
			}
		}

		return next(ctx)
	}
}

func (r *rateLimiterV2) limitOne(ctx *types.Ctx, md *metadata.Metadata) (err error) {
	app := md.GetRoute().GetAppName(ctx)
	var (
		rule limitRule
		hit  bool
	)
	for _, ru := range r.flags.rules {
		if ru.Category == app {
			rule = ru
			hit = true
			break
		}
	}
	if !hit {
		glog.Debug(ctx, "rateLimiterV2 un match app or category rule:"+app)
		return nil
	}

	dataProvider := r.flags.dataProvider
	if rule.DataProvider != "" {
		dataProvider = rule.DataProvider
	}

	limitType := r.flags.limitType
	if rule.LimitType != "" {
		limitType = rule.LimitType
	}
	return r.Limit(ctx, md, dataProvider, limitType, rule)
}

// Limit gen rate and burst and redis-key
func (r *rateLimiterV2) Limit(ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
	now := time.Now()

	if rule.UID && md.UID <= 0 {
		return berror.NewInterErr("redis_limiter_v2 uid is 0")
	}

	limit, key, err := r.loadQuota(ctx, md, dataProvider, rule)
	if err != nil {
		return err
	}

	ratio := int(limit.Period / time.Second)
	defaultValue := defaultRate * ratio
	if limit.Rate > defaultValue {
		limit.Rate = defaultValue
	}
	if limit.Rate <= 0 {
		err = berror.NewInterErr("redis_limiter_v2 rate is 0", dataProvider, limitType)
		return
	}
	if limit.Burst <= 0 {
		limit.Burst = limit.Rate
	}
	glog.Debug(ctx, "redis key", glog.String("key", key), glog.Any("rule-limit", rule))

	if err = r.doLimit(ctx, key, limitType, rule.Step, limit, bplatform.Client(md.Extension.Platform) == bplatform.OpenAPI); err != nil {
		return
	}

	glog.Debug(ctx, "redis_limiter_v2 allowed", glog.Duration("cost", time.Since(now)), glog.Any("limit", limit), glog.Any("routeKey", md.GetRoute().GetAppName(ctx)))
	return
}

func (r *rateLimiterV2) loadQuota(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
	appName := md.GetRoute().GetAppName(ctx)
	if md.Route.AllApp {
		return getAutoQuota(ctx, appName, md, rule)
	}

	// obtain user-defined limit rate
	params := &rateParams{
		dataProvider: dataProvider,
		uid:          md.UID,
		group:        rule.Group,
	}

	var symbol string
	if rule.Symbol {
		symbol = symbolconfig.GetSymbol(ctx)
		if symbol == "" || strings.Contains(symbol, ",") {
			symbol = BTCUSD
		} else {
			sc, err := symbolconfig.GetSymbolModule()
			if err != nil {
				galert.Error(ctx, "redis limiterV2 GetSymbolModule error, "+err.Error())
				symbol = BTCUSD
			} else {
				sy := sc.SymbolFromName(symbol)
				if sy == 0 {
					symbol = BTCUSD
				}
			}
		}
	}

	if r.flags.unified {
		params.key = getUnifiedKey(md.UID, rule.Group)
	} else {
		ks := rule.Values(ctx, symbol)
		keys := append([]string{appName}, ks...)
		params.key = strings.Join(keys, ":")
		params.symbol = symbol
	}

	// get limit rate, burst, period
	limit, err := r.loadRate(ctx, appName, rule, r.flags.unified, params)
	if err != nil {
		return gredis.Limit{}, "", err
	}

	return limit, params.key, nil
}

type rateParams struct {
	dataProvider string
	uid          int64
	group        string
	symbol       string
	key          string
}

// loadRate obtain user-defined limit rate
func (r *rateLimiterV2) loadRate(ctx *types.Ctx, app string, ru limitRule, unified bool, params *rateParams) (gredis.Limit, error) {
	// if rule does not have uid dimensions
	if !ru.UID {
		return ru.Limit, nil
	}

	if !ru.EnableCustomRate {
		return ru.Limit, nil
	}

	// !NOTE: get quota from loaders only when group is not empty, if rate is zero, use user's quota
	quota, err := r.getQuota(ctx, app, unified, params)
	if err != nil {
		return ru.Limit, err
	}

	if ru.Rate <= 0 && quota <= 0 {
		glog.Error(ctx, "redis limiterV2 v2 group rate is 0")
		return ru.Limit, berror.NewInterErr("redis limiterV2 v2 group rate is 0")
	}
	// use user's quota first
	if quota > 0 {
		ru.Rate = quota
	}
	// cap limit maximum of rate
	if ru.Cap > 0 && ru.Rate > ru.Cap {
		ru.Rate = ru.Cap
	}
	return ru.Limit, nil
}

// Init implement filter.Initializer
// args like: ( "routeKey", "--group=group --rate=10 --burst=10 --period=10m" )
func (r *rateLimiterV2) Init(ctx context.Context, args ...string) (err error) {
	if len(args) == 0 {
		return berror.NewInterErr("invalid args, can't must filter")
	}

	if err = r.initRules(ctx, args...); err != nil {
		return
	}

	var enableCustomRate bool
	for _, rule := range r.flags.rules {
		if rule.EnableCustomRate {
			enableCustomRate = true
			break
		}
	}
	if enableCustomRate {
		return r.initLoaders(ctx, args...)
	}
	return
}

func (r *rateLimiterV2) doLimit(ctx *types.Ctx, key, limitType string, step int, limit gredis.Limit, withHeader bool) (err error) {
	now := time.Now()
	defer func() {
		gmetric.ObserveDefaultLatencySince(now, "rate_v2", "limit")
	}()

	var result *gredis.Result

	switch limitType {
	case limitCounter:
		result, err = redis.AllowM(service.GetContext(ctx), r.limiter, key, limit, step)
		if err != nil {
			// !!NOTE ignore redis internal error, not block biz request
			glog.Error(ctx, "redis.AllowM error", glog.String("key", key), glog.Any("limit", limit), glog.String("err", err.Error()))
			gmetric.IncDefaultError("redis_limit_v2", "allowm_error")
			return nil
		}
	default:
		result, err = redis.AllowN(service.GetContext(ctx), r.limiter, key, limit, step)
		if err != nil {
			// !!NOTE ignore redis internal error, not block biz request
			glog.Error(ctx, "redis.AllowN error", glog.String("key", key), glog.Any("limit", limit), glog.String("err", err.Error()))
			gmetric.IncDefaultError("redis_limit_v2", "allown_error")
			return nil
		}
	}

	// output limit result to header
	setHeader(ctx, limit.Rate, result, withHeader)
	glog.Debug(ctx, "redis_limiter_v2 result", glog.Any("limit-result", result))

	if result != nil && result.Allowed == 0 {
		glog.Debug(ctx, "redis_limiter_v2 blocked", glog.Duration("cost", time.Since(now)), glog.String("route", key),
			glog.Any("limit", limit), glog.Any("limit-result", result))
		return berror.ErrVisitsLimit
	}
	return
}

// getQuota get limit rate value from service provider
// return rate, resetSymbol, error
func (r *rateLimiterV2) getQuota(ctx context.Context, app string, unified bool, params *rateParams) (int, error) {
	switch params.dataProvider {
	case futuresService: // futures only
		rate, err := rateLimitMgr.GetQuota(service.GetContext(ctx), params.uid, app, params.group, params.symbol)
		if err != nil {
			glog.Error(ctx, "rateLimitMgr.GetQuota error", glog.String("error", err.Error()))
			return 0, err
		}
		return int(rate), nil
	}

	if app == optionService && !unified {
		params.key = getOptionKey(params.uid, params.group)
	}

	if unified || app == optionService {
		glog.Debug(ctx, "get unified quota", glog.Int64("uid", params.uid), glog.String("group", params.group))
		return getUnifiedQuota(ctx, params.uid, params.group), nil
	}

	value, ok := limitV2Loaders.Load(app)
	if !ok {
		glog.Error(ctx, "load loader failed", glog.String("app", app), glog.String("key", params.key))
		return 0, nil
	}
	loader, ok := value.(*quotaLoaderV2)
	if !ok {
		glog.Error(ctx, "get loader failed", glog.String("app", app), glog.String("key", params.key))
		return 0, nil
	}

	return loader.getQuota(ctx, params), nil
}

func (r *rateLimiterV2) initLoaders(ctx context.Context, args ...string) error {
	routeKey := metadata.RouteKey{}
	routeKey = routeKey.Parse(args[0])

	if routeKey.AppName == "" {
		return berror.NewInterErr(errInvalidAppName.Error())
	}

	initFun := func(app string) error {
		actual, loaded := limitV2Loaders.LoadOrStore(app, newQuotaLoaderV2(app))
		if !loaded {
			q, ok := actual.(*quotaLoaderV2)
			if !ok {
				return berror.NewInterErr(errInvalidQuotaLoader.Error())
			}
			return q.init(ctx)
		}

		return nil
	}
	if !routeKey.AllApp {
		return initFun(routeKey.AppName)
	}
	if err := initFun(constant.AppTypeSPOT); err != nil {
		return err
	}

	return nil
}

// options filter options
type options struct {
	Category         string `json:"category,omitempty" yaml:"category,omitempty"`                     // futures,option,spot
	Rate             int32  `json:"rate,omitempty" yaml:"rate,omitempty"`                             // 额度
	Burst            int32  `json:"burst,omitempty" yaml:"burst,omitempty"`                           // 桶大小，默认和rate相同
	Step             int32  `json:"step,omitempty" yaml:"step,omitempty"`                             // 限流步长
	PeriodSec        int32  `json:"periodSec,omitempty" yaml:"periodSec,omitempty"`                   // 限流周期，默认1，单位秒
	Uid              bool   `json:"uid,omitempty" yaml:"uid,omitempty"`                               // uid维度
	Path             bool   `json:"path,omitempty" yaml:"path,omitempty"`                             // path维度
	Method           bool   `json:"method,omitempty" yaml:"method,omitempty"`                         // method维度
	Symbol           bool   `json:"symbol,omitempty" yaml:"symbol,omitempty"`                         // symbol维度
	EnableCustomRate bool   `json:"enable_custom_rate,omitempty" yaml:"enable_custom_rate,omitempty"` // 是否启用自定义提频
	Group            string `json:"group,omitempty" yaml:"group,omitempty"`                           // group维度名称
	DataProvider     string `json:"dataProvider,omitempty" yaml:"dataProvider,omitempty"`             // 限频数据源，futures，etcd，默认为网关etcd
	LimitType        string `json:"limitType,omitempty" yaml:"limitType,omitempty"`                   // 限流方式，counter,bucket,默认为令牌桶
}

func (r *rateLimiterV2) initRules(ctx context.Context, args ...string) (err error) {
	if err = r.parseFlags(args); err != nil {
		return
	}

	isFutures := r.flags.dataProvider == futuresService
	isOption := r.flags.dataProvider == optionService
	for _, r2 := range r.flags.rules {
		if r2.DataProvider == futuresService || r2.Category == futuresService {
			isFutures = true
		}
		if r2.DataProvider == optionService || r2.Category == optionService {
			isOption = true
		}
	}

	if isFutures {
		if err = symbolconfig.InitSymbolConfig(); err != nil {
			return fmt.Errorf("redis limiterV2 InitSymbolConfig error: %w", err)
		}
		rateOnce.Do(func() {
			rateLimitMgr, err = NewQuotaManager(ctx)
		})
		if err != nil {
			glog.Error(ctx, "futures NewQuotaManager error", glog.String("error", err.Error()))
			return
		}
	}

	// build unified quota loader
	if r.flags.unified || isOption {
		actual, loaded := limiterRules.LoadOrStore(optionService, newQuotaLoader(optionService))
		if !loaded {
			q, ok := actual.(*quotaLoader)
			if !ok {
				return berror.NewInterErr(errInvalidQuotaLoader.Error())
			}
			// init first time
			if err = q.init(ctx); err != nil {
				return
			}
		}
	}

	return
}

func (r *rateLimiterV2) parseFlags(args []string) (err error) {
	var (
		rules []limitRule
		s     string
		f     flagsV2
		optsT string
	)

	parse := flag.NewFlagSet("limiterV2", flag.ContinueOnError)

	parse.StringVar(&s, "rules", "[]", "rule json string")
	parse.StringVar(&f.dataProvider, "data", "", "rate data provider")
	parse.StringVar(&f.limitType, "type", "", "rate limit type, counter or default")
	parse.BoolVar(&f.unified, "unified", false, "unified limit")
	parse.StringVar(&f.selectType, "selectType", "all", "limit rule select type")
	parse.StringVar(&optsT, "options", "[]", "limit options")

	if err = parse.Parse(args[1:]); err != nil {
		return
	}

	if err = util.JsonUnmarshal([]byte(s), &rules); err != nil {
		return
	}

	for i := 0; i < len(rules); i++ {
		if rules[i].Rate <= 0 {
			return berror.NewInterErr("rate is invalid", cast.ToString(rules[i].Rate))
		}
		if rules[i].Step <= 0 {
			rules[i].Step = 1
		}
		// default one second
		if rules[i].PeriodSec <= 0 {
			rules[i].Period = time.Second
		} else {
			rules[i].Period = time.Second * time.Duration(rules[i].PeriodSec)
		}
	}

	var opts []options
	if err = util.JsonUnmarshal([]byte(optsT), &opts); err != nil {
		return
	}

	for _, opt := range opts {
		var rule limitRule
		rule.Category = opt.Category
		rule.Rate = int(opt.Rate)
		if rule.Rate <= 0 {
			return berror.NewInterErr("rate is invalid", cast.ToString(rule.Rate))
		}
		rule.Burst = int(opt.Burst)
		rule.Step = int(opt.Step)
		if rule.Step <= 0 {
			rule.Step = 1
		}
		rule.PeriodSec = int64(opt.PeriodSec)
		rule.Period = time.Second
		if opt.PeriodSec > 0 {
			rule.Period = time.Duration(opt.PeriodSec) * time.Second
		}
		rule.UID = opt.Uid
		rule.Path = opt.Path
		rule.Method = opt.Method
		rule.Symbol = opt.Symbol
		rule.EnableCustomRate = opt.EnableCustomRate
		rule.Group = opt.Group
		rule.DataProvider = opt.DataProvider
		if rule.DataProvider != "" && rule.DataProvider != futuresService {
			return fmt.Errorf("redis limiterV2: unknown data provider: %s", rule.DataProvider)
		}
		rule.LimitType = opt.LimitType
		if rule.LimitType != "" && rule.LimitType != limitCounter {
			glog.Info(context.TODO(), "redis limiterV2: unknown limit type:"+rule.LimitType)
			rule.LimitType = ""
		}
		rules = append(rules, rule)
	}

	f.rules = rules
	r.flags = f

	return
}
