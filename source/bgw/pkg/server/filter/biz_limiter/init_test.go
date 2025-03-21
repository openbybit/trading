package biz_limiter

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"

	"bgw/pkg/server/filter"
)

func TestInit(t *testing.T) {

	Init()

	convey.Convey("TestInit", t, func() {
		rateLimiter, err := filter.GetFilter(context.Background(), filter.BizRateLimitFilterKey)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(rateLimiter, convey.ShouldBeNil)
	})
}
