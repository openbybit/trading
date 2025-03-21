package biz_limiter

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"git.bybit.com/svc/mod/pkg/bplatform"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/server/filter/biz_limiter/rate"
)

var (
	_ filter.Filter      = &limiterMemo{}
	_ filter.Initializer = &limiterMemo{}
)

type limiterMemo struct {
	flags          flagsV2
	setHeader      bool
	defaultLimiter *rate.Limiter
	mutex          sync.RWMutex
	limiters       map[string]*rate.Limiter
}

func newLimiterMemo() filter.Filter {
	return &limiterMemo{
		limiters: make(map[string]*rate.Limiter),
	}
}

// GetName returns the name of the filter
func (l *limiterMemo) GetName() string {
	return filter.BizRateLimitFilterMEMO
}

// Do limit handle
func (l *limiterMemo) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		var (
			err error
			md  = metadata.MDFromContext(ctx)
		)

		if l.flags.selectType == selectTypeOne {
			if err = l.limitOne(ctx, md); err != nil {
				return err
			}

			return next(ctx)
		}

		// As soon as one current limiterMemo gets rejected, return
		for _, rule := range l.flags.rules {
			if rule.IsZero() {
				continue
			}

			dataProvider := l.flags.dataProvider
			if rule.DataProvider != "" {
				dataProvider = rule.DataProvider
			}

			limitType := l.flags.limitType
			if rule.LimitType != "" {
				limitType = rule.LimitType
			}

			err = l.Limit(ctx, md, dataProvider, limitType, rule)
			if err != nil {
				return err
			}
		}

		return next(ctx)
	}
}

func (l *limiterMemo) limitOne(ctx *types.Ctx, md *metadata.Metadata) (err error) {
	app := md.GetRoute().GetAppName(ctx)
	var (
		rule limitRule
		hit  bool
	)
	for _, ru := range l.flags.rules {
		if ru.Category == app {
			rule = ru
			hit = true
			break
		}
	}
	if !hit {
		glog.Debug(ctx, "LimiterV3 un match app or category rule:"+app)
		return nil
	}

	dataProvider := l.flags.dataProvider
	if rule.DataProvider != "" {
		dataProvider = rule.DataProvider
	}

	limitType := l.flags.limitType
	if rule.LimitType != "" {
		limitType = rule.LimitType
	}
	return l.Limit(ctx, md, dataProvider, limitType, rule)
}

func (l *limiterMemo) Limit(ctx *types.Ctx, md *metadata.Metadata, dataProvider, limitType string, rule limitRule) (err error) {
	if rule.UID && md.UID <= 0 {
		return berror.NewInterErr("limiter_v3 uid is 0")
	}

	limit, key, err := l.loadQuota(ctx, md, dataProvider, rule)
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

	l.mutex.RLock()
	limiter, ok := l.limiters[key]
	l.mutex.RUnlock()
	if !ok || int(limiter.Limit()) != limit.Rate {
		l.mutex.Lock()
		limiter, ok = l.limiters[key]
		if !ok || int(limiter.Limit()) != limit.Rate {
			limiter = rate.NewLimiter(rate.Limit(limit.Rate), limit.Burst)
			l.limiters[key] = limiter
		}
		l.mutex.Unlock()
	}
	remain, allow := limiter.AllowAvailable()
	if !allow {
		setHeaderMemo(ctx, limit.Rate, remain, bplatform.Client(md.Extension.Platform) == bplatform.OpenAPI)
		glog.Debug(ctx, "limiter memo block", glog.String("key", key))
		return berror.ErrVisitsLimit
	}

	md.LimitRule = key
	md.LimitValue = limit.Rate
	md.LimitPeriod = rule.PeriodSec

	app := md.GetRoute().GetAppName(ctx)
	if l.flags.batch && app == "futures" {
		symbols, err := symbolconfig.GetBatchSymbol(ctx)
		if err != nil {
			return berror.ErrInvalidRequest
		}
		md.BatchItems = int32(len(symbols))
	}
	if l.setHeader {
		setHeaderMemo(ctx, limit.Rate, remain, bplatform.Client(md.Extension.Platform) == bplatform.OpenAPI)
	}

	glog.Debug(ctx, "limiter memo allowed", glog.Any("limit", limit), glog.Any("routeKey", md.GetRoute().GetAppName(ctx)))
	return
}

func (l *limiterMemo) loadQuota(ctx *types.Ctx, md *metadata.Metadata, dataProvider string, rule limitRule) (gredis.Limit, string, error) {
	appName := md.GetRoute().GetAppName(ctx)
	if md.Route.AllApp {
		return getAutoQuota(ctx, appName, md, rule)
	}
	return gredis.Limit{}, "", fmt.Errorf("limiter memo not support")
}

// Init set limiterMemo rate
func (l *limiterMemo) Init(ctx context.Context, args ...string) (err error) {
	if len(args) == 0 {
		return berror.NewInterErr("invalid args, can't must filter")
	}

	if err = l.initRules(ctx, args...); err != nil {
		return
	}

	var enableCustomRate bool
	for _, rule := range l.flags.rules {
		if rule.EnableCustomRate {
			enableCustomRate = true
			break
		}
	}
	if enableCustomRate {
		return l.initLoaders(ctx, args...)
	}
	return
}

func (l *limiterMemo) initRules(ctx context.Context, args ...string) (err error) {
	if err = l.parseFlags(args); err != nil {
		return
	}

	isFutures := l.flags.dataProvider == futuresService
	isOption := l.flags.dataProvider == optionService
	for _, r2 := range l.flags.rules {
		if r2.DataProvider == futuresService || r2.Category == futuresService {
			isFutures = true
		}
		if r2.DataProvider == optionService || r2.Category == optionService {
			isOption = true
		}
	}

	if isFutures {
		if err = symbolconfig.InitSymbolConfig(); err != nil {
			return fmt.Errorf("redis limiterMemo InitSymbolConfig error: %w", err)
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
	if l.flags.unified || isOption {
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

func (l *limiterMemo) parseFlags(args []string) (err error) {
	var (
		rules []limitRule
		s     string
		f     flagsV2
		optsT string
	)

	parse := flag.NewFlagSet("limiterMemo", flag.ContinueOnError)

	parse.StringVar(&s, "rules", "[]", "rule json string")
	parse.StringVar(&f.dataProvider, "data", "", "rate data provider")
	parse.StringVar(&f.limitType, "type", "", "rate limit type, counter or default")
	parse.BoolVar(&f.unified, "unified", false, "unified limit")
	parse.StringVar(&f.selectType, "selectType", "all", "limit rule select type")
	parse.StringVar(&optsT, "options", "[]", "limit options")
	parse.BoolVar(&l.setHeader, "setHeader", false, "if bgw set limit response header")
	parse.BoolVar(&f.batch, "batch", false, "if batch items need")

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
			return fmt.Errorf("redis limiterMemo: unknown data provider: %s", rule.DataProvider)
		}
		rule.LimitType = opt.LimitType
		if rule.LimitType != "" && rule.LimitType != limitCounter {
			glog.Info(context.TODO(), "redis limiterMemo: unknown limit type:"+rule.LimitType)
			rule.LimitType = ""
		}
		rules = append(rules, rule)
	}

	f.rules = rules
	l.flags = f

	return
}

func (l *limiterMemo) initLoaders(ctx context.Context, args ...string) error {
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

func setHeaderMemo(ctx *types.Ctx, quote int, remain int, withHeader bool) {
	if !withHeader {
		return
	}
	rateLimit := metadata.RateLimitInfo{
		RateLimitStatus: remain,
		RateLimit:       quote,
	}
	ctx.Response.Header.Set(constant.HeaderAPILimit, cast.Itoa(quote))
	ctx.Response.Header.Set(constant.HeaderAPILimitStatus, cast.Itoa(remain))
	var restTime int
	if remain == 0 { // block
		restTime = int((time.Now().UnixNano() + time.Second.Nanoseconds()) / int64(time.Millisecond))
	} else {
		restTime = int(time.Now().UnixNano() / 1e6)
	}
	rateLimit.RateLimitResetMs = restTime
	ctx.Response.Header.Set(constant.HeaderAPILimitResetTimestamp, cast.Itoa(restTime))
	metadata.ContextWithRateLimitInfo(ctx, rateLimit)
}
