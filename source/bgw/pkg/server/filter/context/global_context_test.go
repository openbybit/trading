package context

import (
	"bgw/pkg/config"
	"context"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	gmetadata "bgw/pkg/server/metadata"
)

func TestGlobalContextFilter_Do(t *testing.T) {
	Convey("test global context filter do", t, func() {
		g := &globalContextFilter{}
		n := g.GetName()
		So(n, ShouldEqual, filter.ContextFilterKeyGlobal)

		err := g.Init(context.Background())
		So(err, ShouldBeNil)

		next := func(ctx *types.Ctx) error {
			return nil
		}

		handler := g.Do(next)
		ctx := &types.Ctx{}

		patch := gomonkey.ApplyFunc(parseBaggage, func(*types.Ctx, *gmetadata.Metadata) {})
		defer patch.Reset()

		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

func TestGlobalContextFilter_outbound(t *testing.T) {
	Convey("test globalContextFilter outbound", t, func() {
		g := &globalContextFilter{}
		ctx := &types.Ctx{}

		m := gmetadata.MDFromContext(ctx)
		s := "secure_token"
		w := "weak_token"
		m.Intermediate.SecureToken = &s
		m.Intermediate.WeakToken = &w

		ctx.SetUserValue(constant.CtxInvokeResult, &md{})
		g.outbound(ctx)

		So(string(ctx.Response.Header.Peek("traceID")), ShouldEqual, "")
		So(string(ctx.Response.Header.Peek("token")), ShouldEqual, w)
		So(ctx.Response.Header.PeekCookie(config.GetSecureTokenKey()), ShouldNotBeNil)
		So(string(ctx.Response.Header.ContentType()), ShouldEqual, "application/json; charset=utf-8")
	})
}
