package cors

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
)

func TestCorsHandler_Do(t *testing.T) {
	Convey("test cors handler do", t, func() {
		f := defaultHandler()
		n := f.GetName()
		So(n, ShouldEqual, filter.CorsFilterKey)

		next := func(ctx *types.Ctx) error {
			return nil
		}
		handler := f.Do(next)

		ctx := &types.Ctx{}
		ctx.Request.Header.SetMethod("OPTIONS")

		err := handler(ctx)
		So(err, ShouldBeNil)
		So(ctx.Response.StatusCode(), ShouldEqual, berror.HttpStatusOK)

		ctx.Request.Header.SetMethod("GET")
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

func TestCorsHandler_handlePreflight(t *testing.T) {
	Convey("test cors handlePreflight", t, func() {
		g := &corsHandler{}
		g.allowedOriginsAll = true
		g.allowCredentials = true
		g.maxAge = 10
		g.allowedMethods = []string{"POST"}

		ctx := &types.Ctx{}
		ctx.Request.Header.Set("Origin", "origin")
		ctx.Request.Header.Set("Access-Control-Request-Method", "GET")
		g.handlePreflight(ctx)

		ctx.Request.Header.Set("Access-Control-Request-Method", "POST")
		g.handlePreflight(ctx)

		ctx.Request.Header.Set("Access-Control-Request-Headers", "header1,header2")
		g.handlePreflight(ctx)
	})
}

func TestCorsHandler_handleActual(t *testing.T) {
	Convey("test cors handlePreflight", t, func() {
		g := &corsHandler{}
		g.allowedOriginsAll = true
		g.allowCredentials = true
		g.exposedHeaders = []string{"123", "234"}
		g.maxAge = 10
		g.allowedMethods = []string{"POST"}

		ctx := &types.Ctx{}
		ctx.Request.Header.Set("Origin", "origin")
		ctx.Request.Header.Set("exposedHeaders", "GET")
		g.handleActual(ctx)
	})
}

func TestCorsHandler_isAllowedOrigin(t *testing.T) {
	Convey("test cors isAllowedOrigin", t, func() {
		g := &corsHandler{}
		g.allowedOrigins = []string{"123", "abc"}

		res := g.isAllowedOrigin("123")
		So(res, ShouldBeTrue)
		res = g.isAllowedOrigin("456")
		So(res, ShouldBeFalse)
	})
}

func TestCorsHandler_isAllowedMethod(t *testing.T) {
	Convey("test cors isAllowedMethod", t, func() {
		g := &corsHandler{}
		res := g.isAllowedMethod("GET")
		So(res, ShouldBeFalse)

		g.allowedMethods = []string{"GET"}
		res = g.isAllowedMethod("OPTIONS")
		So(res, ShouldBeTrue)
	})
}

func TestCorsHandler_areHeadersAllowed(t *testing.T) {
	Convey("test cors areHeadersAllowed", t, func() {
		g := &corsHandler{}
		g.allowedHeadersAll = true
		res := g.areHeadersAllowed(nil)
		So(res, ShouldBeTrue)

		g.allowedHeadersAll = false
		g.allowedHeaders = []string{"123", "234"}

		res = g.areHeadersAllowed([]string{"123"})
		So(res, ShouldBeTrue)

		res = g.areHeadersAllowed([]string{"456"})
		So(res, ShouldBeFalse)
	})
}

func TestNewCorsHandler(t *testing.T) {
	Convey("test newCorsHandler", t, func() {
		Init()
		op := &Options{}
		op.AllowedHeaders = []string{"*"}
		ch := newCorsHandler(op)
		So(ch, ShouldNotBeNil)
	})
}
