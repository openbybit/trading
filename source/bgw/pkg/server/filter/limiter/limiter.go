package limiter

import (
	"context"
	"flag"
	"time"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"golang.org/x/time/rate"
)

var (
	_ filter.Filter      = &limiter{}
	_ filter.Initializer = &limiter{}

	defaultRate = 100
)

type limiter struct {
	rateLimit int
	limiter   *rate.Limiter
}

func new() filter.Filter {
	if d := config.AppCfg().UpstreamQpsRate; d > 0 {
		defaultRate = d
	}

	return &limiter{
		rateLimit: defaultRate,
	}
}

// GetName returns the name of the filter
func (l *limiter) GetName() string {
	return filter.QPSRateLimitFilterKey
}

// Do limit handle
func (l *limiter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		now := time.Now()
		if !l.limiter.Allow() {
			glog.Debug(ctx, "local limiter block", glog.Duration("cost", time.Since(now)), glog.String("path", cast.UnsafeBytesToString(ctx.Path())))
			return berror.ErrVisitsLimit
		}

		glog.Debug(ctx, "local limiter allow", glog.Duration("cost", time.Since(now)), glog.String("path", cast.UnsafeBytesToString(ctx.Path())))

		return next(ctx)
	}
}

// Init set limiter rate
func (l *limiter) Init(ctx context.Context, args ...string) (err error) {
	if err = l.parseFlags(args); err != nil {
		glog.Error(ctx, "limiter limiterFlagParse error", glog.String("error", err.Error()))
		return
	}

	l.limiter = rate.NewLimiter(rate.Limit(l.rateLimit), l.rateLimit)
	return
}

func (l *limiter) parseFlags(args []string) (err error) {
	if len(args) == 0 {
		return nil
	}

	parse := flag.NewFlagSet("limiter", flag.ContinueOnError)
	parse.IntVar(&l.rateLimit, "rate", defaultRate, "rate val")

	return parse.Parse(args[1:])
}
