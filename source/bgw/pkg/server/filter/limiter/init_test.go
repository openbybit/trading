package limiter

import (
	"bgw/pkg/server/filter"
	"context"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInit(t *testing.T) {
	convey.Convey("TestInit", t, func() {
		Init()
		f, err := filter.GetFilter(context.Background(), filter.QPSRateLimitFilterKey)
		convey.So(err, convey.ShouldBeNil)
		convey.So(f, convey.ShouldNotBeNil)

		f, err = filter.GetFilter(context.Background(), filter.QPSRateLimitFilterKeyGlobal)
		convey.So(err, convey.ShouldBeNil)
		convey.So(f, convey.ShouldNotBeNil)

		f, err = filter.GetFilter(context.Background(), filter.IPRateLimitFilterKey)
		convey.So(err, convey.ShouldBeNil)
		convey.So(f, convey.ShouldNotBeNil)
	})
}
