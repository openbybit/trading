package gray

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
)

func TestStrategies_grayCheck(t *testing.T) {
	Convey("test grayCheck", t, func() {
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.UID = 123

		var sgs Strategies = []*strategy{{Strags: GrayStrategyUid, Value: []any{234}}, {Strags: GrayStrategyFullOn}}
		ok, err := sgs.grayCheck(ctx)
		So(ok, ShouldBeTrue)
		So(err, ShouldBeNil)

		sgs = []*strategy{{Strags: GrayStrategyUid, Value: []any{234}}, {Strags: "empty", Value: []any{234}}}
		ok, err = sgs.grayCheck(ctx)
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)

		sgs = []*strategy{{Strags: GrayStrategyUid, Value: []any{234}}, {Strags: GrayStrategyFullClose}}
		ok, err = sgs.grayCheck(ctx)
		So(ok, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}
