package biz_limiter

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"git.bybit.com/svc/mod/pkg/bplatform"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/remoting/redis"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service"
	"bgw/pkg/service/symbolconfig"
)

var (
	defaultRate = 10000
)

type flags struct {
	group             string
	hasSymbol         bool
	disableCustomRate bool
	unified           bool
	gredis.Limit
}

type rateLimiter struct {
	limiter *gredis.Limiter
	flags   flags
}

func newLimiter() filter.Filter {
	client := redis.NewClient()
	if client == nil {
		panic("connect redis error")
	}

	return &rateLimiter{
		limiter: gredis.NewLimiter(client),
	}
}

// GetName returns the name of the filter
func (*rateLimiter) GetName() string {
	return filter.BizRateLimitFilterKey
}

// Do the limit
func (r *rateLimiter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		var (
			err   error
			md    = metadata.MDFromContext(ctx)
			route = md.GetRoute()
		)

		if !route.Valid() {
			return berror.ErrRouteKeyInvalid
		}

		switch route.ACL.Group {
		case constant.ResourceGroupBlockTrade:
			if err := r.blockTradeLimit(ctx, route, md); err != nil {
				return err
			}
		default:
			// member limit & api limit
			err = r.Limit(ctx, route, md.UID, md.Extension.Platform)
			if err != nil {
				return err
			}
		}

		return next(ctx)
	}
}

func (r *rateLimiter) Limit(c *types.Ctx, route metadata.RouteKey, memberID int64, platform string) (err error) {
	var now = time.Now()

	// get default limit
	limit := r.flags.Limit

	redisKey := route.String()
	// !NOTE: get quota from loaders only when group is not empty, if rate is zero, use user's quota
	if r.flags.group != "" {
		if memberID <= 0 {
			return berror.NewInterErr("redis limiter memberID is 0, group mode")
		}

		appName := route.GetAppName(c)
		if !r.flags.disableCustomRate {
			var rate int
			if r.flags.unified { // unified use option limit
				rate = r.getQuota(c, "option", memberID, r.flags.group)
			} else {
				rate = r.getQuota(c, appName, memberID, r.flags.group)
			}
			if limit.Rate <= 0 && rate <= 0 {
				err = berror.NewInterErr("redis limiter group rate is 0")
				return
			}
			// use user's quota first
			if rate > 0 {
				limit.Rate = rate
			}
		}

		// group by group&account
		if r.flags.unified {
			redisKey = getUnifiedKey(memberID, r.flags.group)
		} else {
			redisKey = fmt.Sprintf("%s:%s:%d", appName, r.flags.group, memberID)
		}
		glog.Debug(c, "redis limiter group", glog.String("key", redisKey), glog.Any("rule", r.flags), glog.Any("rate", limit))
	}

	// add symbol
	if r.flags.hasSymbol {
		symbol := symbolconfig.GetSymbol(c)

		if symbol == "" || strings.Contains(symbol, ",") {
			symbol = BTCUSD
		}
		redisKey = fmt.Sprintf("%s:%s", redisKey, symbol)
	}

	if limit.Rate > defaultRate {
		limit.Rate = defaultRate
	}
	if limit.Rate <= 0 {
		err = berror.NewInterErr("redis group limiter rate is 0")
		return
	}
	if limit.Burst <= 0 {
		limit.Burst = limit.Rate
	}

	if err = r.doLimit(c, redisKey, limit, platform); err != nil {
		return
	}

	glog.Debug(c, "redis_limiter allowed", glog.Duration("cost", time.Since(now)), glog.Bool("rule.hasSymbol", r.flags.hasSymbol),
		glog.String("redis key", redisKey), glog.Any("route", route))

	return
}

func (r *rateLimiter) blockTradeLimit(ctx *types.Ctx, route metadata.RouteKey, md *metadata.Metadata) error {
	blockTrade := bizmetedata.BlockTradeFromContext(ctx)
	if blockTrade == nil {
		return berror.NewInterErr("block trade redis_limiter error, no block trade info")
	}
	// blocktrade taker limit
	if err := r.Limit(ctx, route, blockTrade.TakerMemberId, md.Extension.Platform); err != nil {
		ctx.SetUserValue(constant.BlockTradeKey, constant.BlockTradeTaker)
		return err
	}

	// blocktrade maker limit
	if blockTrade.MakerMemberId <= 0 {
		return nil
	}
	if err := r.Limit(ctx, route, blockTrade.MakerMemberId, md.Extension.Platform); err != nil {
		ctx.SetUserValue(constant.BlockTradeKey, constant.BlockTradeMaker)
		return err
	}
	return nil
}

// Init implement filter.Initializer
// args like: ( "routeKey", "--group=group --rate=10 --burst=10 --period=10m" )
func (r *rateLimiter) Init(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return berror.NewInterErr("invalid args, can't must filter")
	}

	routeKey := metadata.RouteKey{}
	routeKey = routeKey.Parse(args[0])

	if routeKey.AppName == "" {
		return berror.NewInterErr(errInvalidAppName.Error())
	}

	if err := r.parseFlags(args); err != nil {
		return err
	}

	if !r.flags.disableCustomRate {
		if !routeKey.AllApp {
			return r.initLoaders(ctx, routeKey.AppName)
		}
		return fmt.Errorf("uta_engin should not use v1 redis limiter")
	}
	return nil
}

func (r *rateLimiter) doLimit(ctx *types.Ctx, key string, limit gredis.Limit, platform string) error {
	now := time.Now()
	defer func() {
		gmetric.ObserveDefaultLatencySince(now, "rate", "limit")
	}()

	result, err := redis.AllowN(service.GetContext(ctx), r.limiter, key, limit, 1)
	if err != nil {
		// if redis internal error, not block biz request
		glog.Error(ctx, "redis.AllowN error", glog.String("key", key), glog.Any("limit", limit), glog.String("err", err.Error()))
		gmetric.IncDefaultError("redis_limit_v1", "allown_error")
		return nil
	}

	setHeader(ctx, limit.Rate, result, platform == string(bplatform.OpenAPI))

	if result != nil && result.Allowed == 0 {
		glog.Debug(ctx, "redis_limiter blocked", glog.Duration("cost", time.Since(now)), glog.String("route", key),
			glog.Any("limit", limit), glog.Any("result", result))
		return berror.ErrVisitsLimit
	}
	return nil
}

func setHeader(ctx *types.Ctx, quote int, result *gredis.Result, withHeader bool) {
	if result == nil || !withHeader {
		return
	}
	rateLimit := metadata.RateLimitInfo{
		RateLimitStatus: result.Remaining,
		RateLimit:       quote,
	}
	ctx.Response.Header.Set(constant.HeaderAPILimit, cast.Itoa(quote))
	ctx.Response.Header.Set(constant.HeaderAPILimitStatus, cast.Itoa(result.Remaining))
	var restTime int
	if result.Allowed == 0 { // block
		restTime = int(time.Now().UnixNano()+int64(result.ResetAfter)) / 1e6
	} else {
		restTime = int(time.Now().UnixNano() / 1e6)
	}
	rateLimit.RateLimitResetMs = restTime
	ctx.Response.Header.Set(constant.HeaderAPILimitResetTimestamp, cast.Itoa(restTime))
	metadata.ContextWithRateLimitInfo(ctx, rateLimit)
}

func (r *rateLimiter) getQuota(ctx context.Context, app string, uid int64, group string) int {
	value, ok := limiterRules.Load(app)
	if !ok {
		glog.Error(ctx, "load loader failed", glog.String("app", app), glog.Int64("uid", uid), glog.String("group", group))
		return 0
	}
	loader, ok := value.(*quotaLoader)
	if !ok {
		glog.Error(ctx, "get loader failed", glog.String("app", app), glog.Int64("uid", uid), glog.String("group", group))
		return 0
	}

	return loader.getQuota(ctx, uid, group)
}

func (r *rateLimiter) initLoaders(ctx context.Context, app string) error {
	actual, loaded := limiterRules.LoadOrStore(app, newQuotaLoader(app))
	if !loaded {
		q, ok := actual.(*quotaLoader)
		if !ok {
			return berror.NewInterErr(errInvalidQuotaLoader.Error())
		}
		return q.init(ctx)
	}

	return nil
}

func (r *rateLimiter) parseFlags(args []string) (err error) {
	var f flags

	parse := flag.NewFlagSet("limiter", flag.ContinueOnError)
	parse.StringVar(&f.group, "group", "", "rate group")
	parse.IntVar(&f.Rate, "rate", 0, "rate val")
	parse.IntVar(&f.Burst, "burst", 0, "burst val")
	parse.DurationVar(&f.Period, "period", time.Second, "period like: 30s, 10m or 1h")
	parse.BoolVar(&f.hasSymbol, "hasSymbol", false, "rate limit use symbol or not")
	parse.BoolVar(&f.disableCustomRate, "disableCustomRate", false, "rate limit disable get custom rate from etcd")
	parse.BoolVar(&f.unified, "unified", false, "unified limit")
	if err = parse.Parse(args[1:]); err != nil {
		return
	}

	// group must default rate
	if f.group != "" && f.Rate <= 0 {
		return berror.NewInterErr("biz limit group is nil, but not default rate")
	}

	r.flags = f
	return
}
