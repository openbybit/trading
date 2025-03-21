package open_interest

import (
	"context"
	"errors"
	"testing"

	fut "code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gsymbol/future"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"
)

var mockErr = errors.New("mock err")

func Test_newOI(t *testing.T) {
	Convey("test new oi", t, func() {
		Init()
		o := newOI()
		n := o.GetName()
		So(n, ShouldEqual, filter.OpenInterestFilterKey)
	})
}

func TestOi_Init(t *testing.T) {
	Convey("test oi init", t, func() {
		o := &oi{}
		patch := gomonkey.ApplyFunc(initOI, func() error { return mockErr })
		defer patch.Reset()
		err := o.Init(context.Background(), "oi")
		So(err, ShouldNotBeNil)
	})
}

func TestOi_Do(t *testing.T) {
	Convey("test oi do", t, func() {
		next := func(c *types.Ctx) error {
			return nil
		}
		o := &oi{}
		handler := o.Do(next)

		oBatch := &oi{true}
		handlerBatch := oBatch.Do(next)

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)

		md.Route.AppName = "spot"
		err := handler(ctx)
		So(err, ShouldBeNil)

		md.Route.AppName = "futures"
		md.UID = 0
		err = handler(ctx)
		So(err, ShouldNotBeNil)

		patch0 := gomonkey.ApplyFunc(gmetric.IncDefaultCounter, func(string, string) {})
		defer patch0.Reset()

		md.UID = 10
		err = handler(ctx)
		So(err, ShouldNotBeNil)

		patch := gomonkey.ApplyFunc(symbolconfig.GetSymbolConfig, func() (*future.Scmeta, error) { return nil, mockErr })
		ctx.Request.SetBody([]byte(`{"symbol": "usdt"}`))
		md.UID = 10
		err = handler(ctx)
		So(err, ShouldBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc(symbolconfig.GetSymbolConfig, func() (*future.Scmeta, error) {
			return &future.Scmeta{}, nil
		})
		defer patch.Reset()

		batchCtx := &types.Ctx{}
		mdb := metadata.MDFromContext(batchCtx)
		mdb.Route.AppName = "futures"
		mdb.UID = 10
		batchCtx.Request.SetBody([]byte(batchBody))

		patch1 := gomonkey.ApplyFunc((*future.Scmeta).SymbolFromName, func(*future.Scmeta, string) fut.Symbol { return fut.Symbol(0) })
		err = handler(ctx)
		So(err, ShouldNotBeNil)
		err = handlerBatch(batchCtx)
		So(err, ShouldBeNil)
		patch1.Reset()

		patch1 = gomonkey.ApplyFunc((*future.Scmeta).SymbolFromName, func(*future.Scmeta, string) fut.Symbol { return fut.Symbol(1) })
		limiter = &mockLimiter{}
		err = handler(ctx)
		So(err, ShouldBeNil)
		patch1.Reset()

		o.batch = true
		batchCtx.Request.SetBody([]byte("null"))
		err = handler(ctx)
		So(err, ShouldNotBeNil)
	})
}

func Test_initOI(t *testing.T) {
	Convey("test init oi", t, func() {
		_ = initOI()
	})
}

func Test_convertVal(t *testing.T) {
	Convey("convertVal", t, func() {
		s := convertVal(true, true)
		So(s, ShouldEqual, "1#1")
	})
}

type mockLimiter struct{}

func (m *mockLimiter) Limit(uid int64, symbol int32, side int32) bool {
	return false
}

func (m *mockLimiter) CheckUserOpenInterestExceeded(uid int64, symbol int32) (buyOI, sellOI bool) {
	return true, false
}

var batchBody = `{
    "category":"linear",
    "request":[
        {
            "symbol":"BTCPERP",
            "side":"Buy",
            "positionIdx":0,
            "orderType":"Limit",
            "qty":"0.01",
            "price":"17000",
            "triggerDirection":2,
            "triggerPrice":"17001",
            "triggerBy":"MarkPrice",
            "timeInForce":"GTC",
            "takeProfit":"",
            "stopLoss":"",
            "reduce_only":false,
            "closeOnTrigger":false
        },
        {
            "symbol":"BTCPERP",
            "side":"Buy",
            "positionIdx":0,
            "orderType":"Limit",
            "qty":"0.01",
            "price":"17000",
            "triggerDirection":2,
            "triggerPrice":"17001",
            "triggerBy":"MarkPrice",
            "timeInForce":"GTC",
            "takeProfit":"",
            "stopLoss":"",
            "reduce_only":false,
            "closeOnTrigger":false
        },
        {
            "symbol":"BTCPERP",
            "side":"Buy",
            "positionIdx":0,
            "orderType":"Limit",
            "qty":"0.01",
            "price":"17000",
            "triggerDirection":2,
            "triggerPrice":"17001",
            "triggerBy":"MarkPrice",
            "timeInForce":"GTC",
            "takeProfit":"",
            "stopLoss":"",
            "reduce_only":false,
            "closeOnTrigger":false
        }
    ]
}`
