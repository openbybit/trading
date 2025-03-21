package compliance

import (
	"testing"

	com "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
)

func TestUaeLeverageCheck(t *testing.T) {
	Convey("test uae leverage check", t, func() {

		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()

		ctrl := gomock.NewController(t)
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(0), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, mockErr)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, nil)
		cfg := &com.SitesConfigItemConfig{}
		cfg.MaxLeverage = 111
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(2), gomock.Any(), gomock.Any(), gomock.Any()).Return("", cfg, nil).AnyTimes()
		gw = mockWall

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.AppName = "futures"
		err := uaeLeverageCheck(ctx, 0, 0, siteUAE, []string{"futures", "123"}, md)
		So(err, ShouldBeNil)

		err = uaeLeverageCheck(ctx, 0, 1, siteUAE, []string{"futures", "123"}, md)
		So(err, ShouldBeNil)

		err = uaeLeverageCheck(ctx, 0, 2, siteUAE, []string{"futures", "123"}, md)
		So(err, ShouldBeNil)

		ctx.Request.SetBody([]byte("{\"buyLeverage\": 345, \"sellLeverage\": 567}"))
		err = uaeLeverageCheck(ctx, 0, 2, siteUAE, []string{"futures", "123"}, md)
		So(err, ShouldBeNil)
	})
}

func TestUaeSymbolCheck(t *testing.T) {
	Convey("test uae symbol check", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.AppName = "futures"
		usc := &UaeSymbolCheck{}

		usc.Category = []string{"spot"}
		err := uaeSymbolCheck(ctx, 0, 0, siteUAE, usc, md)
		So(err, ShouldBeNil)

		ctrl := gomock.NewController(t)
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(0), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, mockErr)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, nil)
		cfg := &com.SitesConfigItemConfig{}
		cfg.MaxLeverage = 111
		cfg.CoinWhiteList = []string{"BTC", "UCDT"}
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(2), gomock.Any(), gomock.Any(), gomock.Any()).Return("", cfg, nil).AnyTimes()
		gw = mockWall

		usc.Category = []string{"futures", "spot"}
		err = uaeSymbolCheck(ctx, 0, 0, siteUAE, usc, md)
		So(err, ShouldBeNil)

		err = uaeSymbolCheck(ctx, 0, 1, siteUAE, usc, md)
		So(err, ShouldBeNil)

		err = uaeSymbolCheck(ctx, 0, 2, siteUAE, usc, md)
		So(err, ShouldNotBeNil)
	})
}

func TestBatchUaeSymbolCheck(t *testing.T) {
	Convey("test batchUaeSymbolCheck", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()

		ctx := &types.Ctx{}
		ctx.Request.SetBody([]byte(batchBody))
		md := metadata.MDFromContext(ctx)
		md.Route.AppName = "futures"
		usc := &UaeSymbolCheck{}

		usc.Category = []string{"spot"}
		batchUaeSymbolCheck(ctx, 0, 0, siteUAE, usc, md)

		ctrl := gomock.NewController(t)
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(0), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, mockErr)
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(1), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, nil)
		cfg := &com.SitesConfigItemConfig{}
		cfg.MaxLeverage = 111
		cfg.CoinWhiteList = []string{"BTC", "UCDT"}
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(), int64(2), gomock.Any(), gomock.Any(), gomock.Any()).Return("", cfg, nil).AnyTimes()
		gw = mockWall

		usc.Category = []string{"futures", "spot"}
		batchUaeSymbolCheck(ctx, 0, 0, siteUAE, usc, md)
		batchUaeSymbolCheck(ctx, 0, 1, siteUAE, usc, md)
		batchUaeSymbolCheck(ctx, 0, 2, siteUAE, usc, md)
	})
}

func TestGetCoin(t *testing.T) {
	Convey("test coin", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetBody([]byte("{\"symbol\": \"BTCUSDT\"}"))
		_, _ = getCoin(ctx, "futures", "symbol")
		_, _ = getCoin(ctx, "spot", "symbol")
	})
}

func TestProductMap(t *testing.T) {
	Convey("test product map", t, func() {
		_ = productMap("spot")
		_ = productMap("futures")
		_ = productMap("option")
		_ = productMap("other")
	})
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
