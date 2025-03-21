package ban

import (
	"bgw/pkg/common/berror"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestTradeCheckSpot(t *testing.T) {
	convey.Convey("TestTradeCheckSpot", t, func() {
		convey.Convey("static ban", func() {
			err := tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  SPOTType,
				TagName:  TradeTag,
				TagValue: "spot_all",
			})
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserLoginBanned)
			err = tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  TradeType,
				TagName:  TradeTag,
				TagValue: AllTrade,
			})
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserLoginBanned)
			err = tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  DBUType,
				TagName:  TradeTag,
				TagValue: SPOTAllKO,
			})
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserLoginBanned)
		})
		convey.Convey("ban not match", func() {
			err := tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  "xxxx",
				TagName:  TradeTag,
				TagValue: "spot_all",
			})
			convey.So(err, convey.ShouldBeNil)
			err = tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  SPOTType,
				TagName:  "sss",
				TagValue: "spot_all",
			})
			convey.So(err, convey.ShouldBeNil)
		})
		convey.Convey("parseBizInfo return nil", func() {
			err := tradeCheckSpot(nil, &ban.UserStatus_BanItem{
				BizType:  SPOTType,
				TagName:  TradeTag,
				TagValue: "kjkjk",
			})
			convey.So(err, convey.ShouldBeNil)
		})
	})
}
