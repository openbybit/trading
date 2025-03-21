package auth

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
)

func Test_GetToken(t *testing.T) {
	Convey("test GetToken", t, func() {
		ctx := &types.Ctx{}
		tk := GetToken(ctx)
		So(tk, ShouldEqual, "")

		ctx.Request.Header.SetCookie("secure-token", "123")
		tk = GetToken(ctx)
		So(tk, ShouldEqual, "123")
	})
}
