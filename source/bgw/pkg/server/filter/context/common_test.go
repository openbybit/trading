package context

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/opentracing/opentracing-go"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	gmetadata "bgw/pkg/server/metadata"
)

func Test_parseCommon(t *testing.T) {
	Convey("test parseBaggage", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("lll/asas")
		ctx.Request.Header.SetReferer("9")
		ctx.Request.Header.SetUserAgent("9")
		ctx.Request.Header.Set("Versioncode", "9")
		ctx.Request.Header.Set(constant.BrokerID, "9")
		ctx.Request.Header.Set(constant.Platform, "pcweb")
		ctx.Request.Header.Set(constant.SiteID, "9")

		m := &gmetadata.Metadata{}
		parseCommon(ctx, m)
		So(m.Path, ShouldEqual, "/lll/asas")
		So(m.BrokerID, ShouldEqual, 9)
		So(m.SiteID, ShouldEqual, "9")
		So(m.Method, ShouldEqual, "POST")
		So(m.Extension.URI, ShouldEqual, "/lll/asas")
		So(m.Extension.Host, ShouldEqual, "")
		So(m.Extension.UserAgent, ShouldEqual, "9")
		So(m.Extension.Referer, ShouldEqual, "9")
		So(m.Extension.OriginPlatform, ShouldEqual, "pcweb")
		So(m.Extension.AppVersion, ShouldEqual, "")
		So(m.Extension.AppName, ShouldEqual, "")
		So(m.Extension.AppVersionCode, ShouldEqual, "")
		So(m.Extension.OpFrom, ShouldEqual, "pcweb")
		So(m.Extension.Platform, ShouldEqual, "pcweb")
		So(m.Extension.OpPlatform, ShouldEqual, "pcweb")
		So(m.Extension.EOpPlatform, ShouldEqual, 1)
		So(m.Extension.EPlatform, ShouldEqual, 3)
	})
}

func Test_parseLang(t *testing.T) {
	Convey("test parseLang", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.Set(constant.AcceptLanguage, "aa-BB")
		l := parseLang(ctx)
		So(l, ShouldEqual, "aa-bb")

		ctx.Request.Header.Set(constant.AcceptLanguage, "abcd")
		l = parseLang(ctx)
		So(l, ShouldEqual, "")

		ctx.Request.Header.Set(constant.Lang, "aa-BB")
		l = parseLang(ctx)
		So(l, ShouldEqual, "aa-BB")

		pp, c := parsePlatform("abc")
		So(pp, ShouldEqual, "")
		So(c, ShouldEqual, 0)
		pp, c = parsePlatform("pcweb")
		So(pp, ShouldEqual, "pcweb")
		So(c, ShouldEqual, 3)
		pp, c = parseOpPlatform("abc")
		So(pp, ShouldEqual, "")
		So(c, ShouldEqual, 0)
		pp, c = parseOpPlatform("pcweb")
		So(pp, ShouldEqual, "pcweb")
		So(c, ShouldEqual, 1)
	})
}

func Test_parseVersionCode(t *testing.T) {
	Convey("test parseVersionCode", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.Set("Versioncode", "abcd")
		v := parseVersionCode(ctx)
		So(v, ShouldEqual, "")

		ctx.Request.Header.Set("Versioncode", "123")
		v = parseVersionCode(ctx)
		So(v, ShouldEqual, "123")
	})
}

func Test_ParseBaggage(t *testing.T) {
	Convey("test parseBaggage", t, func() {
		ctx := &types.Ctx{}
		span, _ := gtrace.Begin(ctx, "test")
		span = span.SetBaggageItem("baggage", "hehe,,x-lane-env=env")

		patch := gomonkey.ApplyFunc(gtrace.SpanFromContext, func(ctx context.Context) opentracing.Span {
			return span
		})
		defer patch.Reset()
		parseBaggage(ctx, &gmetadata.Metadata{})
	})
}
