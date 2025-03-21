package core

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
)

func TestCtxRouteDataProvider(t *testing.T) {
	Convey("test CtxRouteDataProvider", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.SetMethod("GET")
		ctx.URI().SetPath("/create")

		cp := &CtxRouteDataProvider{
			ctx: ctx,
		}

		m := cp.GetMethod()
		So(m, ShouldEqual, "GET")

		p := cp.GetPath()
		So(p, ShouldEqual, "/create")

		v := cp.GetValue(keyCategory)
		So(v, ShouldEqual, "")

		ctx = &types.Ctx{}
		ctx.Request.Header.SetMethod("POST")
		b := `{
    "category":"spot",
}
`
		ctx.Request.SetBody([]byte(b))
		cp = &CtxRouteDataProvider{
			ctx: ctx,
		}
		v = cp.GetValue(keyCategory)
		So(v, ShouldEqual, "spot")

		vs := cp.GetValues(keyCategory)
		So(len(vs), ShouldEqual, 1)

		ctx.Request.Header.SetContentType(string(bhttp.ContentTypePostForm))
		vs = cp.GetValues(keyCategory)
		So(len(vs), ShouldEqual, 0)

		ctx = &types.Ctx{}
		cp = &CtxRouteDataProvider{
			ctx: ctx,
		}

		vs = cp.GetValues(keyCategory)
		So(len(vs), ShouldEqual, 0)

	})
}

var body = `{
    "category":"spot",
    "symbol":"BTCUSDT",
    "orderType":"Limit", 
    "side":"sell",
    "qty":"0.01",
    "price":"21000",
    "timeInForce":"GTC",
    "smpType":"CancelTaker",
	"category":"spot",
}
`

// 300ns
func BenchmarkCtxRouteDataProvider_GetValues(b *testing.B) {
	ctx := &types.Ctx{}
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte(body))
	cp := NewCtxRouteDataProvider(ctx, nil, nil)

	for i := 0; i < b.N; i++ {
		_ = cp.GetValues("category")
	}
}
