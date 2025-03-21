package ban

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/service/symbolconfig"
	"code.bydev.io/fbu/gateway/gway.git/gsymbol"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
)

func TestTradeCheckOptions(t *testing.T) {
	convey.Convey("TestTradeCheckOptions", t, func() {
		convey.Convey("static ban", func() {
			o := &Options{}
			ro, err := tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: "usdc_options_lu",
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeTrue)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  UTAType,
				TagName:  TradeTag,
				TagValue: LightenUp,
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeTrue)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  TradeType,
				TagName:  TradeTag,
				TagValue: LightenUp,
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeTrue)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: "usdc_options_all",
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserUsdtAllBanned)
			convey.So(ro, convey.ShouldBeFalse)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  UTAType,
				TagName:  TradeTag,
				TagValue: AllTrade,
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserUsdtAllBanned)
			convey.So(ro, convey.ShouldBeFalse)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  TradeType,
				TagName:  TradeTag,
				TagValue: AllTrade,
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserUsdtAllBanned)
			convey.So(ro, convey.ShouldBeFalse)
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DBUType,
				TagName:  TradeTag,
				TagValue: OptionsAllKO,
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserUsdtAllBanned)
			convey.So(ro, convey.ShouldBeFalse)

		})
		convey.Convey("not match", func() {
			o := &Options{}
			ro, err := tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DBUType,
				TagName:  TradeTag,
				TagValue: "1212",
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeFalse)
		})
		convey.Convey("coin ban", func() {
			// symbolconfig.GetOptionManager() nil
			p := gomonkey.ApplyFuncReturn(symbolconfig.GetOptionManager, nil)
			defer p.Reset()
			o := &Options{
				symbol: "",
			}
			ro, err := tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: optionsCoinBanPrefix + "BTC",
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrInvalidRequest)
			convey.So(ro, convey.ShouldBeFalse)
			p.Reset()

			// "symbol not found"
			o = &Options{
				symbol: "asas",
			}
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: optionsCoinBanPrefix + "BTC",
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeFalse)

			// coin not match
			o = &Options{
				symbol: "BTC-OPXXX",
			}
			gom := gsymbol.GetOptionManager()
			p.ApplyPrivateMethod(reflect.TypeOf(gom), "GetByName", func(name string) *gsymbol.OptionConfig {
				return &gsymbol.OptionConfig{
					BaseCoin:   "BTC",
					SymbolName: "BTC-OPXXX",
				}
			})
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: optionsCoinBanPrefix + "ETH",
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeFalse)

			// coin all ban
			o = &Options{
				symbol: "BTC-29DEC23-10000-C",
			}
			p.ApplyPrivateMethod(reflect.TypeOf(gom), "GetByName", func(name string) *gsymbol.OptionConfig {
				return &gsymbol.OptionConfig{
					BaseCoin:   "BTC",
					SymbolName: "BTC-29DEC23-10000-C",
				}
			})
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: optionsCoinBanPrefix + "_BTC",
			}, o)
			convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserUsdtAllBanned)
			convey.So(ro, convey.ShouldBeFalse)

			// coin lu ban
			o = &Options{
				symbol: "BTC-29DEC23-10000-C",
			}
			p.ApplyPrivateMethod(reflect.TypeOf(gom), "GetByName", func(name string) *gsymbol.OptionConfig {
				return &gsymbol.OptionConfig{
					BaseCoin:   "BTC",
					SymbolName: "BTC-29DEC23-10000-C",
				}
			})
			ro, err = tradeCheckOptions(nil, &ban.UserStatus_BanItem{
				BizType:  DERIVATIVESType,
				TagName:  TradeTag,
				TagValue: optionsCoinBanPrefix + "_BTC" + "_lu",
			}, o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ro, convey.ShouldBeTrue)
		})
	})
}
