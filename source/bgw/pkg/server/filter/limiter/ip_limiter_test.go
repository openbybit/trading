package limiter

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

func TestIpLimiter(t *testing.T) {
	Convey("test iplimiter", t, func() {
		i := newIPLimiter().(*ipLimiter)
		So(i.GetName(), ShouldEqual, filter.IPRateLimitFilterKey)
		err := i.Init(context.Background())
		So(err, ShouldBeNil)
		err = i.Init(context.Background(), []string{"route", "--allowIPs=127.0.0.1,127.0.0.2"}...)
		So(err, ShouldBeNil)

		next := func(ctx *types.Ctx) error { return nil }
		handler := i.Do(next)

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Extension.RemoteIP = "127.0.0.5"
		err = handler(ctx)
		So(err, ShouldNotBeNil)

		md.Extension.RemoteIP = "127.0.0.1"
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}
