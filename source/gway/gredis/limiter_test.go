package gredis

import (
	"context"
	"testing"

	"code.bydev.io/frameworks/byone/core/stores/redis"
	"github.com/smartystreets/goconvey/convey"
)

func TestLimiter(t *testing.T) {
	// Create a Redis connection for testing (replace with your Redis connection details)
	rdb := newClient()

	// Create a Limiter instance
	limiter := NewLimiter(rdb)

	// Run the tests using Convey
	convey.Convey("Limiter", t, func() {
		convey.Convey("Allow", func() {
			convey.Convey("Should allow events within the rate limit", func() {
				// Test data
				key := "test-key"
				limit := PerSecond(10)

				// Allow an event
				result, err := limiter.Allow(context.Background(), key, limit)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Allowed, convey.ShouldBeGreaterThan, 0)

				err = limiter.Reset(context.Background(), key, defaultRateType)
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("Should reject events beyond the rate limit", func() {
				// Test data
				key := "test-key"
				limit := PerMinute(5)

				// Allow more events than the rate limit
				result, err := limiter.AllowN(context.Background(), key, limit, 10)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Allowed, convey.ShouldEqual, 0)

				err = limiter.Reset(context.Background(), key, defaultRateType)
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("Should allow events within the rata limit by counter", func() {
				// Test data
				key := "test-key"
				limit := PerMinute(10)

				// Allow more events than the rate limit
				result, err := limiter.AllowM(context.Background(), key, limit, 1)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Allowed, convey.ShouldEqual, 1)

				err = limiter.Reset(context.Background(), key, defaultRateType)
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("Should allow events when we use the two rate algorithms cross", func() {
				// Test data
				key := "test-key"
				limit := PerMinute(10)

				// Allow an event
				result, err := limiter.Allow(context.Background(), key, limit)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Allowed, convey.ShouldEqual, 1)

				// AlloM an events
				result, err = limiter.AllowM(context.Background(), key, limit, 1)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Allowed, convey.ShouldEqual, 1)

				err = limiter.Reset(context.Background(), key, defaultRateType)
				convey.So(err, convey.ShouldBeNil)

				err = limiter.Reset(context.Background(), key, counterRateType)
				convey.So(err, convey.ShouldBeNil)
			})
		})

	})
}

func newClient() *redis.Redis {
	cfg := redis.RedisConf{
		//bybit-test-1 redis-cluster
		Host:     "k8s-istiosys-bybittes-d6ee46754a-f7b06eb42e2fcb36.elb.ap-southeast-1.amazonaws.com:6379",
		Type:     redis.ClusterType,
		Tls:      false,
		Breaker:  true,
		DB:       0,
		NonBlock: false,
	}
	err := cfg.Validate()
	if err != nil {
		return nil
	}
	return cfg.NewRedis()
}
