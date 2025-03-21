package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"code.bydev.io/frameworks/byone/core/stores/redis"

	"bgw/pkg/config"
)

var (
	defaultClient *redis.Redis
	once          sync.Once
	downgrade     bool
	rateLog       = rate.Sometimes{
		Every:    100,
		Interval: time.Second,
	}
)

func NewClient() *redis.Redis {
	if defaultClient == nil {
		once.Do(func() {
			ctx := context.TODO()
			err := config.Global.Redis.Validate()
			if err != nil {
				glog.Error(ctx, "redis config validate error", glog.String("error", err.Error()))
				galert.Error(ctx, "redis config validate error", galert.WithField("err", err))
				return
			}
			downgrade = config.Global.App.RedisDowngrade
			if downgrade {
				glog.Info(ctx, "redis limiter downgrade")
			}
			defaultClient = config.Global.Redis.NewRedis()
		})
		return defaultClient
	}

	return defaultClient
}

// AllowN reports whether n events may happen at time now.
func AllowN(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
	span, _ := gtrace.Begin(ctx, fmt.Sprintf("redis-limiter-AllowN:%s", key))
	defer gtrace.Finish(span)

	if downgrade {
		return nil, errors.New("redis limiter downgrade")
	}

	res, err := l.AllowN(ctx, key, limit, n)
	if err != nil {
		gmetric.IncDefaultError("redis", "allowN")
		rateLog.Do(func() {
			glog.Error(ctx, "redis.AllowN error", glog.String("key", key), glog.Any("limit", limit), glog.String("err", err.Error()))
		})
		return nil, err
	}

	span.SetTag("limit-rate", res.Limit.Rate)
	span.SetTag("limit-burst", res.Limit.Burst)
	span.SetTag("limit-Period", res.Limit.Period)
	span.SetTag("limit-Allowed", n)

	return res, nil
}

// AllowM reports whether n events may happen at time now.
// this is a different implementation used for bgwg.
func AllowM(ctx context.Context, l *gredis.Limiter, key string, limit gredis.Limit, n int) (*gredis.Result, error) {
	span, _ := gtrace.Begin(ctx, fmt.Sprintf("redis-limiter-AllowM:%s", key))
	defer gtrace.Finish(span)

	if downgrade {
		return nil, errors.New("redis limiter downgrade")
	}

	res, err := l.AllowM(ctx, key, limit, n)
	if err != nil {
		gmetric.IncDefaultError("redis", "allowM")
		rateLog.Do(func() {
			glog.Error(ctx, "redis.AllowM error", glog.String("key", key), glog.Any("limit", limit), glog.String("err", err.Error()))
		})
		return nil, err
	}

	span.SetTag("limit-rate", res.Limit.Rate)
	span.SetTag("limit-burst", res.Limit.Burst)
	span.SetTag("limit-Period", res.Limit.Period)
	span.SetTag("limit-Allowed", n)

	return res, nil
}
