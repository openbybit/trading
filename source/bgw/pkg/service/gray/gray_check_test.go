package gray

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
)

func TestRegisterChecker(t *testing.T) {
	Convey("test registerChecker", t, func() {
		registerChecker(GrayStrategyPath, nil)
		c, ok := getChecker(GrayStrategyPath)
		So(c, ShouldNotBeNil)
		So(ok, ShouldBeTrue)

		c, ok = getChecker("emptyChecker")
		So(c, ShouldBeNil)
		So(ok, ShouldBeFalse)
	})
}

func TestCheckers(t *testing.T) {
	Convey("test checkers", t, func() {
		ok, err := allOnCheck(context.Background(), nil)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		ok, err = allCloseCheck(context.Background(), nil)
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.UID = 123
		ok, err = uidGrayCheck(ctx, []any{123, 456})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		ok, err = uidGrayCheck(ctx, []any{456, 789})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		ok, err = tailGrayCheck(ctx, []any{456, 789})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		ok, err = tailGrayCheck(ctx, []any{23})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		ok, err = tailGrayCheck(ctx, []any{12})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		md.UID = 100
		ok, err = tailGrayCheck(ctx, []any{0})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		md.Route.Registry = "serviceA"
		ok, err = serviceGrayCheck(ctx, []any{"serviceA"})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		ok, err = serviceGrayCheck(ctx, []any{"serviceB"})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		md.Path = "PathA"
		ok, err = pathGrayCheck(ctx, []any{"PathA"})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		ok, err = pathGrayCheck(ctx, []any{"pathB"})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		md.Extension.RemoteIP = "127.0.0.1"
		ok, err = ipGrayCheck(ctx, []any{"127.0.0.1"})
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		md.Extension.RemoteIP = "127.0.0.1"
		ok, err = ipGrayCheck(ctx, []any{"127.0.0.2"})
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}
