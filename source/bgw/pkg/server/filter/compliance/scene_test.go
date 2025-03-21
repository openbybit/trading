package compliance

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
)

func TestCompliance_getScene(t *testing.T) {
	Convey("test get scene", t, func() {
		c := &complianceWall{}
		c.multiScenes = map[string]string{"f": "123"}

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.AppName = "f"

		s := c.getScene(ctx)
		So(s, ShouldEqual, "123")
	})
}

func TestSLTMatch(t *testing.T) {
	Convey(" test slt match", t, func() {
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.AppName = "f"

		res := SLTMatch(ctx, "spot", "symbol", md)
		So(res, ShouldBeFalse)

		ctx = &types.Ctx{}
		md.Route.AppName = "spot"
		ctx.Request.Header.SetMethod("POST")
		res = SLTMatch(ctx, "spot", "symbol", md)
		So(res, ShouldBeFalse)
	})
}
