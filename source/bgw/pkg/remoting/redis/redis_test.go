package redis

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestNewClient(t *testing.T) {
	convey.Convey("TestNewClient", t, func() {
		client := NewClient()
		convey.So(client, convey.ShouldNotBeNil)
	})
}

func TestAllowN(t *testing.T) {
	convey.Convey("TestAllowN", t, func() {
		patches := gomonkey.ApplyMethod(reflect.TypeOf(gredis.Limiter{}), "AllowN", func(_ gredis.Limiter, _ context.Context, _ string, _ gredis.Limit, _ int) (*gredis.Result, error) {
			return &gredis.Result{
				Limit:     gredis.PerSecond(10),
				Allowed:   1,
				Remaining: 9,
			}, nil
		})
		defer patches.Reset()
		result, err := AllowN(context.Background(), rateLimiter(), "TestAllowN", gredis.PerSecond(10), 1)
		convey.So(err, convey.ShouldBeNil)
		convey.So(result, convey.ShouldNotBeNil)

		convey.Convey("TestAllowN error", func() {
			gmetric.Init("test")
			applyMethod := gomonkey.ApplyMethod(reflect.TypeOf(*rateLimiter()), "AllowN", func(_ gredis.Limiter, _ context.Context, _ string, _ gredis.Limit, _ int) (*gredis.Result, error) {
				log.Println("mock AllowN")
				return nil, errors.New("test error")
			})
			defer applyMethod.Reset()
			result, err = AllowN(context.Background(), rateLimiter(), "TestAllowN", gredis.PerSecond(10), 1)
			convey.So(result, convey.ShouldBeNil)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func TestAllowM(t *testing.T) {
	convey.Convey("TestAllowM", t, func() {
		patches := gomonkey.ApplyMethod(reflect.TypeOf(gredis.Limiter{}), "AllowM", func(_ gredis.Limiter, _ context.Context, _ string, _ gredis.Limit, _ int) (*gredis.Result, error) {
			return &gredis.Result{
				Limit:     gredis.PerSecond(10),
				Allowed:   1,
				Remaining: 9,
			}, nil
		})
		defer patches.Reset()

		result, err := AllowM(context.Background(), rateLimiter(), "test", gredis.PerSecond(10), 1)
		convey.So(err, convey.ShouldBeNil)
		convey.So(result, convey.ShouldNotBeNil)
		convey.So(result.Allowed, convey.ShouldEqual, 1)
		convey.So(result.Remaining, convey.ShouldEqual, 9)
	})
}

func rateLimiter() *gredis.Limiter {
	client := NewClient()
	return gredis.NewLimiter(client)
}
