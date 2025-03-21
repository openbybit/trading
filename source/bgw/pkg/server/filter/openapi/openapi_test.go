package openapi

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gsmp"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service/ban"
	ropenapi "bgw/pkg/service/openapi"
	"bgw/pkg/service/smp"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/service/tradingroute"
	user2 "bgw/pkg/service/user"
	"bgw/pkg/test"
	"bgw/pkg/test/mock"

	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	sUser "bgw/pkg/service/user"
)

func TestVerify(t *testing.T) {
	gmetric.Init("TestVerify")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	banSvc := ban.NewMockBanServiceIface(ctrl)
	checker := mock.NewMockChecker(ctrl)
	checker.EXPECT().GetAPIKey().Return("xxxx").AnyTimes()

	osvc := mock.NewMockOpenAPIServiceIface(ctrl)
	p := gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, osvc, errors.New("xxx"))
	f := &openapi{
		as: &mockAs{
			aids:              []int64{1, 2, 3},
			queryMemberTagERR: errors.New("ddd"),
		},
	}

	rctx, _ := test.NewReqCtx()
	md := metadata.NewMetadata()
	resp, err := f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{})
	assert.Equal(t, int64(0), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.EqualError(t, err, "xxx")
	p.Reset()
	p = gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, osvc, nil)
	defer p.Reset()
	f = &openapi{
		as: &mockAs{
			aids:              []int64{1, 2, 3},
			queryMemberTagERR: errors.New("ddd"),
		},
	}

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(nil, errors.New("sssd"))
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: true,
	})
	assert.Equal(t, int64(0), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.NoError(t, err)

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(nil, errors.New("sssd"))
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: false,
	})
	assert.Equal(t, int64(0), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.EqualError(t, err, "sssd")

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(&user.MemberLogin{
		MemberId: 100,
		BrokerId: 200,
	}, nil)
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: true,
	})
	assert.Equal(t, int64(100), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, int64(100), md.UID)
	assert.Equal(t, int32(200), md.BrokerID)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.NoError(t, err)

	banP := gomonkey.ApplyFuncReturn(ban.GetBanService, banSvc, errors.New("x123"))

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(&user.MemberLogin{
		MemberId: 100,
		BrokerId: 200,
	}, nil)
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: false,
	})
	assert.Equal(t, int64(100), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, int64(100), md.GetMemberID())
	assert.Equal(t, int32(200), md.BrokerID)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.EqualError(t, err, "x123")

	banP.Reset()

	banP = gomonkey.ApplyFuncReturn(ban.GetBanService, banSvc, nil)

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"

	banSvc.EXPECT().CheckStatus(gomock.Any(), gomock.Eq(int64(100))).Return(nil, errors.New("321"))
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(&user.MemberLogin{
		MemberId: 100,
		BrokerId: 200,
	}, nil)
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: false,
	})
	assert.Equal(t, int64(100), resp.memberId)
	assert.Equal(t, int32(0), resp.loginStatus)
	assert.Equal(t, int32(0), resp.tradeStatus)
	assert.Equal(t, int32(0), resp.withdrawStatus)
	assert.Equal(t, int64(100), md.GetMemberID())
	assert.Equal(t, int32(200), md.BrokerID)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.EqualError(t, err, "321")

	rctx, _ = test.NewReqCtx()
	md = metadata.NewMetadata()
	md.Extension.XOriginFrom = "asas"

	checker.EXPECT().VerifySign(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	checker.EXPECT().GetClientIP().Return("asas").Times(2)
	banSvc.EXPECT().CheckStatus(gomock.Any(), gomock.Eq(int64(100))).Return(&ban.UserStatusWrap{
		LoginStatus:    1,
		WithdrawStatus: 1,
		TradeStatus:    1,
		LoginBanType:   1,
	}, nil)
	osvc.EXPECT().VerifyAPIKey(gomock.Any(), gomock.Eq("xxxx"), gomock.Eq("asas")).Return(&user.MemberLogin{
		MemberId: 100,
		BrokerId: 200,
		ExtInfo: &user.MemberLoginExt{
			AppName: "as",
		},
	}, nil)
	f = &openapi{}
	ppp := gomonkey.ApplyPrivateMethod(reflect.TypeOf(f), "tradeCheck", func(ctx *types.Ctx, tradeCheck bool, batchTradeCheck map[string]struct{}, md *metadata.Metadata, banSvc ban.BanServiceIface, memberStatus *ban.UserStatusWrap) error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(f), "checkBannedCountries", func(ctx *types.Ctx, rule *openapiRule, ip string) error { return nil }).
		ApplyPrivateMethod(reflect.TypeOf(f), "checkIp", func(ctx *types.Ctx, skipIpCheck bool, member *ropenapi.MemberLogin, ip string) (err error) {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(f), "checkPermission", func(ctx *types.Ctx, permissions string, acl metadata.ACL) error { return nil })
	defer ppp.Reset()
	resp, err = f.verify(rctx, checker, md, metadata.RouteKey{}, &openapiRule{
		allowGuest: false,
	})
	assert.Equal(t, int64(100), resp.memberId)
	assert.Equal(t, int32(1), resp.loginStatus)
	assert.Equal(t, int32(1), resp.tradeStatus)
	assert.Equal(t, int32(1), resp.withdrawStatus)
	assert.Equal(t, int64(100), md.GetMemberID())
	assert.Equal(t, int32(200), md.BrokerID)
	assert.Equal(t, "xxxx", md.APIKey)
	assert.NoError(t, err)

	banP.Reset()
}
func BenchmarkV2GetAPI_Origin_Origin(b *testing.B) {
	a := assert.New(b)
	ctx := &types.Ctx{}
	uri := &fasthttp.URI{}
	uri.SetQueryString("a=b&f=g,h,y&sign=0d3b619867c7aa920875df3a1e0ed80c58b386fa8e3c2c2f0ae9e78ec261c98b&api_key=5FdeE4CnNztmXLC9HE&timestamp=1665760702678&recv_window=30000000")
	ctx.Request.SetURI(uri)
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v2, err := newV2Checker(ctx, "127.0.0.1", false, false)
		a.NoError(err)
		err = v2[0].VerifySign(ctx, sign.TypeHmac, secret)
		a.NoError(err)
	}
}

func BenchmarkV2GetAPI_Encode_Encode(b *testing.B) {
	a := assert.New(b)
	ctx := &types.Ctx{}
	uri := &fasthttp.URI{}
	uri.SetQueryString("f=g%2Ch%2Cy&sign=6bbb02cc1ad3ee276175d67133eee51a93dacae9b4a751ee46555db6da3313cb&api_key=5FdeE4CnNztmXLC9HE&timestamp=1665761075765&recv_window=30000000&a=b")
	ctx.Request.SetURI(uri)
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v2, err := newV2Checker(ctx, "127.0.0.1", false, false)
		a.NoError(err)
		err = v2[0].VerifySign(ctx, sign.TypeHmac, secret)
		a.NoError(err)
	}
}

func BenchmarkV2GetAPI_Origin_Encode(b *testing.B) {
	a := assert.New(b)
	ctx := &types.Ctx{}
	uri := &fasthttp.URI{}
	uri.SetQueryString("recv_window=30000000&a=b&f=g%2Ch%2Cy&sign=c99b8f204d300f223a0e469c1efeed7c4e9b7ff90d91ba8eb959b45679a92106&api_key=5FdeE4CnNztmXLC9HE&timestamp=1665761222284")
	ctx.Request.SetURI(uri)
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v2, err := newV2Checker(ctx, "127.0.0.1", false, false)
		a.NoError(err)
		err = v2[0].VerifySign(ctx, sign.TypeHmac, secret)
		a.NoError(err)
	}
}

func TestLimitFlagParse(t *testing.T) {
	gmetric.Init("TestLimitFlagParse")
	a := assert.New(t)
	rule, err := limiterFlagParse(context.Background(), []string{"--skipAID=true", "--copyTrade=true", `--copyTradeInfo={"allowGuest":true}`})
	a.NotNil(rule.copyTradeInfo)
	a.NoError(err)
	rule, err = limiterFlagParse(context.Background(), []string{"--skipAID true", "--copyTrade true", `--copyTradeInfo={"allowGuest":true}`})
	a.Error(err)
	rule, err = limiterFlagParse(context.Background(), []string{"--skipAID=true", "--bizType=0"})
	a.NoError(err)
	a.Equal(1, rule.bizType)
	rule, err = limiterFlagParse(context.Background(), []string{"--skipAID=true", "--bizType=0", "--copyTradeInfo=asas"})
	a.Error(err)
	a.Equal(0, rule.bizType)
}

func TestParseFlagTradeCheck(t *testing.T) {
	gmetric.Init("TestParseFlagTradeCheck")
	a := assert.New(t)
	_, err := limiterFlagParse(context.Background(), []string{"openapi", "--tradeCheck=true", "--tradeCheckCfg=0"})
	a.Equal("json: cannot unmarshal number into Go value of type openapi.tradeCheckCfg", err.Error())
	r, err := limiterFlagParse(context.Background(), []string{"openapi", "--tradeCheck=true", "--tradeCheckCfg={\"symbolField\":\"12\"}"})
	a.NoError(err)
	a.Equal("12", r.tradeCheckCfg.SymbolField)
	// init symbol config err
	p := gomonkey.ApplyFuncReturn(symbolconfig.InitSymbolConfig, errors.New("xxx"))
	defer p.Reset()
	_, err = limiterFlagParse(context.Background(), []string{"openapi", "--tradeCheck=true"})
	a.Equal("xxx", err.Error())
}

func TestParseFlagBatchTradeCheck(t *testing.T) {
	gmetric.Init("TestParseFlagBatchTradeCheck")
	a := assert.New(t)
	_, err := limiterFlagParse(context.Background(), []string{"openapi", "--batchTradeCheck=options", "--batchTradeCheckCfg=0"})
	a.Equal("json: cannot unmarshal number into Go value of type map[string]*openapi.tradeCheckCfg", err.Error())
	r, err := limiterFlagParse(context.Background(), []string{"openapi", "--batchTradeCheck=options", "--batchTradeCheckCfg={\"options\":{\"symbolField\":\"12\"}}"})
	a.NoError(err)
	a.Equal("12", r.batchTradeCheckCfg["options"].SymbolField)
}

func TestVerifySymbol(t *testing.T) {
	var symbolLimit *user.SymbolLimit

	convey.Convey("verifySymbol", t, func() {
		convey.Convey("symbolLimit is nil", func() {
			convey.So(symbolLimit, convey.ShouldEqual, (*user.SymbolLimit)(nil))
		})

		symbolLimit = &user.SymbolLimit{
			Spot: map[string]string{
				"BTCUSDT": "",
			},
		}

		ctx := new(types.Ctx)
		metadata.MDFromContext(ctx).Route.AppName = constant.AppTypeSPOT

		f := openapi{}
		convey.Convey("symbol empty", func() {
			ctx.SetUserValue(constant.BgwRequestParsed, map[string]interface{}{symbolconfig.Symbol: ""})
			err := f.verifySymbol(ctx, symbolLimit)
			convey.So(err, convey.ShouldEqual, berror.ErrSymbolLimited)
		})

		convey.Convey("spot symbol not limit", func() {
			ctx.SetUserValue(constant.BgwRequestParsed, map[string]interface{}{symbolconfig.Symbol: "BTCUSDT"})
			err := f.verifySymbol(ctx, symbolLimit)
			convey.So(err, convey.ShouldEqual, nil)
		})

		convey.Convey("spot symbol limited", func() {
			ctx.SetUserValue(constant.BgwRequestParsed, map[string]interface{}{symbolconfig.Symbol: "ETHUSDT"})
			err := f.verifySymbol(ctx, symbolLimit)
			convey.So(err.Error(), convey.ShouldEqual, berror.ErrSymbolLimited.Error())
		})
	})
}

func TestOpenapi(t *testing.T) {
	t.Run("getRule", func(t *testing.T) {
		rctx, _ := test.NewReqCtx()
		rctx.Request.Header.Set(constant.HeaderAPIKey, "1212")
		f := openapi{}
		rule := f.getRule(metadata.RouteKey{})
		assert.Nil(t, rule)

		or := &openapiRule{}
		r := metadata.RouteKey{}
		f.rules.Store(r.String(), or)
		rule = f.getRule(r)
		assert.NotNil(t, rule)
		assert.Equal(t, rule, or)
	})
	t.Run("setPlatform", func(t *testing.T) {
		rctx, _ := test.NewReqCtx()
		rctx.Request.Header.Set(constant.HeaderAPIKey, "1212")
		f := openapi{}
		md := metadata.NewMetadata()
		f.setPlatform(md)
		assert.Equal(t, "api", md.Extension.OpFrom)
		assert.Equal(t, "openapi", md.Extension.Platform)
		assert.Equal(t, int32(6), md.Extension.EPlatform)
		assert.Equal(t, "openapi", md.Extension.OpPlatform)
		assert.Equal(t, int32(3), md.Extension.EOpPlatform)
	})

}

func TestGetAid(t *testing.T) {
	t.Run("getAid unifiedTrading", func(t *testing.T) {
		f := openapi{
			as: &mockAs{
				aids:              []int64{1, 2, 3},
				queryMemberTagERR: errors.New("ddd"),
			},
		}
		rctx, _ := test.NewReqCtx()
		md := metadata.NewMetadata()
		err := f.getAid(rctx, md, "123", &openapiRule{
			unifiedTrading: true,
		})
		assert.Equal(t, "ddd", err.Error())
		assert.Equal(t, int64(0), md.UaID)
		assert.Equal(t, int64(0), md.AccountID)
		assert.Equal(t, false, md.UnifiedTrading)
		assert.Equal(t, false, md.UnifiedMargin)
		f = openapi{
			as: &mockAs{
				aids:                         []int64{1, 2, 3},
				memberTag:                    "ttt",
				getUnifiedMarginAccountIDERR: errors.New("ddd"),
			},
		}
		md = metadata.NewMetadata()
		err = f.getAid(rctx, md, "123", &openapiRule{
			unifiedTrading: true,
		})
		assert.Equal(t, "ddd", err.Error())
		assert.Equal(t, "ttt", md.UaTag)
		assert.Equal(t, int64(0), md.UaID)
		assert.Equal(t, int64(0), md.AccountID)
		assert.Equal(t, false, md.UnifiedTrading)
		assert.Equal(t, false, md.UnifiedMargin)

		f = openapi{
			as: &mockAs{
				aids:      []int64{1, 2, 3},
				memberTag: "ttt",
			},
		}
		md = metadata.NewMetadata()
		err = f.getAid(rctx, md, "123", &openapiRule{
			unifiedTrading: true,
		})
		assert.NoError(t, err)
		assert.Equal(t, "ttt", md.UaTag)
		assert.Equal(t, int64(1), md.UaID)
		assert.Equal(t, int64(1), md.AccountID)
		assert.Equal(t, true, md.UnifiedTrading)
		assert.Equal(t, false, md.UnifiedMargin)
	})

	t.Run("getAid unified", func(t *testing.T) {
		f := openapi{
			as: &mockAs{
				aids:              []int64{1, 2, 3},
				queryMemberTagERR: errors.New("ddd"),
			},
		}
		rctx, _ := test.NewReqCtx()
		md := metadata.NewMetadata()
		f = openapi{
			as: &mockAs{
				aids:                         []int64{1, 2, 3},
				getUnifiedMarginAccountIDERR: errors.New("ddd"),
			},
		}
		md = metadata.NewMetadata()
		err := f.getAid(rctx, md, "123", &openapiRule{
			unified: true,
		})
		assert.Equal(t, "ddd", err.Error())
		assert.Equal(t, "", md.UaTag)
		assert.Equal(t, int64(0), md.UnifiedID)
		assert.Equal(t, int64(0), md.AccountID)
		assert.Equal(t, false, md.UnifiedMargin)
		assert.Equal(t, false, md.UnifiedTrading)

		f = openapi{
			as: &mockAs{
				aids: []int64{1, 2, 3},
			},
		}
		md = metadata.NewMetadata()
		err = f.getAid(rctx, md, "123", &openapiRule{
			unified: true,
		})
		assert.NoError(t, err)
		assert.Equal(t, "", md.UaTag)
		assert.Equal(t, int64(1), md.UnifiedID)
		assert.Equal(t, int64(1), md.AccountID)
		assert.Equal(t, true, md.UnifiedMargin)
		assert.Equal(t, false, md.UnifiedTrading)

		f = openapi{
			as: &mockAs{
				aids: []int64{0},
			},
		}
		rctx.Request.SetRequestURIBytes(UnifiedPrivateURLPrefix)
		md = metadata.NewMetadata()
		err = f.getAid(rctx, md, "123", &openapiRule{
			unified: true,
		})
		assert.Equal(t, berror.ErrUnifiedMarginAccess, err)
	})

	t.Run("getAid default", func(t *testing.T) {
		f := openapi{
			as: &mockAs{
				aids:              []int64{1, 2, 3},
				queryMemberTagERR: errors.New("ddd"),
			},
		}
		rctx, _ := test.NewReqCtx()
		md := metadata.NewMetadata()
		f = openapi{
			as: &mockAs{
				aids:            []int64{1, 2, 3},
				getAccountIDErr: errors.New("ddd"),
			},
		}
		md = metadata.NewMetadata()
		err := f.getAid(rctx, md, "123", &openapiRule{})
		assert.Equal(t, "ddd", err.Error())
		assert.Equal(t, "", md.UaTag)
		assert.Equal(t, int64(0), md.UnifiedID)
		assert.Equal(t, int64(0), md.AccountID)
		assert.Equal(t, false, md.UnifiedMargin)
		assert.Equal(t, false, md.UnifiedTrading)
		f = openapi{
			as: &mockAs{
				aids: []int64{1, 2, 3},
			},
		}
		md = metadata.NewMetadata()
		err = f.getAid(rctx, md, "123", &openapiRule{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), md.AccountID)
		assert.Equal(t, "", md.UaTag)
		assert.Equal(t, int64(0), md.UnifiedID)
		assert.Equal(t, false, md.UnifiedMargin)
		assert.Equal(t, false, md.UnifiedTrading)
	})
}

func TestOpenapi_Do_1(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_1")
	// route is invalid
	aa, ctrl, _ := makeOpenApi(t)
	rctx, md := test.NewReqCtx()
	md.Route = metadata.RouteKey{}
	err := aa.Do(nil)(rctx)
	ctrl.Finish()
	assert.Equal(t, "openapi invalid route", err.Error())
}

func TestOpenapi_Do_2(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_2")
	// rule is nil
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "invalid openapi rule, test.ccc.aaa.ddd.xxx.get.false", err.Error())
	ctrl.Finish()
}

func TestOpenapi_Do_3(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_3")
	// getCheckers is err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return &openapiRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{}, errors.New("getCheckers err")
	})

	err := aa.Do(nil)(rctx)
	assert.EqualError(t, err, "getCheckers err")
	ctrl.Finish()
	p.Reset()
}

func TestOpenapi_Do_4(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_4")
	// rule.allowGuest && checkers[0].GetAPIKey() == ""
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return &openapiRule{
			allowGuest: true,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: ""}}, nil
	})

	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return errors.New("next")
	})(rctx)
	assert.EqualError(t, err, "next")
	ctrl.Finish()
	p.Reset()
}

func TestOpenapi_Do_5(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_5")
	// route.ACL.Group == constant.ResourceGroupBlockTrade err
	rctx, md := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return &openapiRule{
			allowGuest: false,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verifyBlockTrade", func(ctx *types.Ctx, v3s [2]Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (verifyResp, error) {
		return verifyResp{memberId: 100}, errors.New("verifyBlockTrade err")
	})
	md.Route.ACL.Group = constant.ResourceGroupBlockTrade
	err := aa.Do(nil)(rctx)
	assert.EqualError(t, err, "verifyBlockTrade err")
	assert.Equal(t, int64(100), md.UID)
	ctrl.Finish()
	p.Reset()
}

func TestOpenapi_Do_6(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_6")
	// route.ACL.Group == constant.ResourceGroupBlockTrade & allowGuest: true
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return &openapiRule{
			allowGuest: true,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verifyBlockTrade", func(ctx *types.Ctx, v3s [2]Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (verifyResp, error) {
		return verifyResp{memberId: 100}, nil
	})
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", errors.New("xxxx"))

	md.Route.ACL.Group = constant.ResourceGroupBlockTrade
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return errors.New("next")
	})(rctx)
	assert.EqualError(t, err, "next")
	assert.Equal(t, int64(100), md.UID)
	assert.Equal(t, "333", md.UserSiteID)
	ctrl.Finish()
	p.Reset()
}

func TestOpenapi_Do_7(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_7")
	// route.ACL.Group != constant.ResourceGroupBlockTrade & verify err
	rctx, md := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	rule := &openapiRule{
		allowGuest: true,
	}
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, errors.New("verify err")
	})
	md.Route.ACL.Group = constant.ResourceGroupAll
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return errors.New("next")
	})(rctx)
	assert.EqualError(t, err, "next")
	assert.Equal(t, int64(100), md.UID)

	rule.allowGuest = false
	md.Route.ACL.Group = constant.ResourceGroupAll
	err = aa.Do(nil)(rctx)
	assert.EqualError(t, err, "verify err")
	assert.Equal(t, int64(100), md.UID)
	ctrl.Finish()
	p.Reset()
}

func TestOpenapi_Do_8(t *testing.T) {
	gmetric.Init("TestOpenapi_Do_8")
	// handleAccountID & unifiedTradingCheck & getSmpGroup
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	router := mock.NewMockRouting(ctrl)
	rule := &openapiRule{
		allowGuest: false,
		suiInfo:    true,
		aioFlag:    true,
	}
	// TODO mock err
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "unifiedTradingCheck", func(c *types.Ctx, rule *openapiRule, md *metadata.Metadata) error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getSmpGroup", func(ctx *types.Ctx, smpGroup bool, md *metadata.Metadata) error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleSuiInfo", func(ctx context.Context, md *metadata.Metadata) {
	}).ApplyFunc(tradingroute.GetRouting, func() tradingroute.Routing {
		return router
	})
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", nil)

	router.EXPECT().IsAioUser(gomock.Any(), gomock.Eq(int64(100))).Return(true, nil)
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return errors.New("next")
	})(rctx)
	assert.EqualError(t, err, "next")
	assert.Equal(t, int64(100), md.UID)
	ctrl.Finish()
	p.Reset()
}

func TestCp(t *testing.T) {
	gmetric.Init("TestCp")
	// copytrade service error & handleAccountID err
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	rule := &openapiRule{
		allowGuest: false,
		copyTrade:  true,
		copyTradeInfo: &user2.CopyTradeInfo{
			AllowGuest: true,
		},
	}
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return errors.New("handleAccountID err")
	}).ApplyFunc(user2.GetCopyTradeService, func() (user2.CopyTradeIface, error) {
		return nil, errors.New("error")
	})
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", nil)

	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return errors.New("next")
	})(rctx)
	assert.EqualError(t, err, "next")
	assert.Equal(t, int64(100), md.UID)
	ctrl.Finish()
	p.Reset()
}

func TestCp1(t *testing.T) {
	gmetric.Init("TestCp1")
	// copytrade GetCopyTradeData error & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)

	rule := &openapiRule{
		allowGuest: false,
		copyTrade:  true,
		copyTradeInfo: &user2.CopyTradeInfo{
			AllowGuest: true,
		},
	}
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", nil)

	cpt := mock.NewMockCopyTradeIface(ctrl)
	cpt.EXPECT().GetCopyTradeData(gomock.Any(), gomock.Any()).Return(nil, errors.New("xxx"))
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return errors.New("handleAccountID err")
	}).ApplyFunc(user2.GetCopyTradeService, func() (user2.CopyTradeIface, error) {
		return cpt, nil
	})

	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.NoError(t, err)

	p.Reset()
	ctrl.Finish()
}

func TestCp2(t *testing.T) {
	gmetric.Init("TestCp2")
	// copytrade GetCopyTradeData error & copyTradeInfo nil & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", nil)

	rule := &openapiRule{
		allowGuest:    false,
		copyTrade:     true,
		copyTradeInfo: nil,
	}
	cpt := mock.NewMockCopyTradeIface(ctrl)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return errors.New("handleAccountID err")
	}).ApplyFunc(user2.GetCopyTradeService, func() (user2.CopyTradeIface, error) {
		return cpt, nil
	})
	cpt.EXPECT().GetCopyTradeData(gomock.Any(), gomock.Any()).Return(nil, errors.New("xxx"))
	defer ctrl.Finish()
	defer p.Reset()
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)
	assert.EqualError(t, err, "xxx")
}

func TestCp3(t *testing.T) {
	gmetric.Init("TestCp3")
	// copytrade GetCopyTradeData no err & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("site-id")).Return("333", nil)

	rule := &openapiRule{
		allowGuest:    false,
		copyTrade:     true,
		copyTradeInfo: nil,
	}
	cpt := mock.NewMockCopyTradeIface(ctrl)
	cpt.EXPECT().GetCopyTradeData(gomock.Any(), gomock.Any()).Return(&user2.CopyTrade{}, nil)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(route metadata.RouteKey) *openapiRule {
		return rule
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "getCheckers", func(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
		return [2]Checker{&v3Checker{apiKey: "xxxxx"}}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return errors.New("handleAccountID err")
	}).ApplyFunc(user2.GetCopyTradeService, func() (user2.CopyTradeIface, error) {
		return cpt, nil
	})
	defer ctrl.Finish()
	defer p.Reset()
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.EqualError(t, err, "handleAccountID err")
	assert.IsType(t, &user2.CopyTrade{}, rctx.Value("copytrade"))
}

func TestVerifyBlockTrade(t *testing.T) {
	gmetric.Init("TestVerifyBlockTrade")
	// rule is nil
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	router := mock.NewMockRouting(ctrl)
	defer ctrl.Finish()

	c1 := mock.NewMockChecker(ctrl)
	c2 := mock.NewMockChecker(ctrl)
	checker := [2]Checker{c1, c2}

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "verify", func(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
		return verifyResp{memberId: 100}, nil
	}).ApplyFunc(tradingroute.GetRouting, func() tradingroute.Routing {
		return router
	})
	m.EXPECT().GetBizAccountIDByApps(gomock.Any(), gomock.Eq(int64(100)),
		gomock.Eq(1), gomock.Eq(constant.AppTypeFUTURES), gomock.Eq(constant.AppTypeOPTION), gomock.Eq(constant.AppTypeSPOT)).
		Return([]int64{500, 600, 700}, []error{nil, nil, nil}).Times(2)
	m.EXPECT().GetUnifiedTradingAccountID(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(1)).Return(int64(200), nil).Times(2)
	m.EXPECT().GetUnifiedMarginAccountID(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(1)).Return(int64(300), nil).Times(2)
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).Return(user2.UnifiedStateProcess, nil).Times(2)
	router.EXPECT().IsAioUser(gomock.Any(), gomock.Eq(int64(100))).Return(true, nil).Times(2)
	defer p.Reset()
	defer ctrl.Finish()

	resp, err := aa.verifyBlockTrade(rctx, checker, md, md.Route, &openapiRule{
		bizType: 1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	bt := rctx.UserValue("blocktrade").(*bizmetedata.BlockTrade)
	assert.Equal(t, int64(100), bt.MakerMemberId)
	assert.Equal(t, int32(0), bt.MakerLoginStatus)
	assert.Equal(t, int32(2), bt.MakerTradeStatus)
	assert.Equal(t, int32(0), bt.MakerWithdrawStatus)
	assert.Equal(t, true, bt.MakerAIOFlag)

	assert.Equal(t, int64(500), bt.MakerFuturesAccountId)
	assert.Equal(t, int64(600), bt.MakerOptionAccountId)
	assert.Equal(t, int64(700), bt.MakerSpotAccountId)
	assert.Equal(t, int64(300), bt.MakerUnifiedAccountId)
	assert.Equal(t, int64(200), bt.MakerUnifiedTradingID)
	assert.Equal(t, int32(2), bt.MakerTradeStatus)
}

func Test(t *testing.T) {
	gmetric.Init("TestRouteRuleNil")
	// rule is nil
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	defer ctrl.Finish()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "invalid openapi rule, test.ccc.aaa.ddd.xxx.get.false", err.Error())
}

func TestUnifiedTradingCheck(t *testing.T) {
	gmetric.Init("TestUnifiedTradingCheck")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	defer ctrl.Finish()

	// uid < 0
	md.UID = -1
	err := aa.unifiedTradingCheck(rctx, &openapiRule{}, md)
	assert.NoError(t, err)

	// !rule.utaProcessBan && rule.unifiedTradingCheck == ""
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		utaProcessBan:       false,
		unifiedTradingCheck: "",
	}, md)
	assert.NoError(t, err)

	// QueryMemberTag err
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).
		Return("", errors.New("ee"))
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		utaProcessBan: true,
	}, md)
	assert.NoError(t, err)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).
		Return(user2.UnifiedStateSuccess, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		unifiedTradingCheck: utaBan,
	}, md)
	assert.EqualError(t, err, "uta banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).
		Return(user2.UnifiedStateFail, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		unifiedTradingCheck: comBan,
	}, md)
	assert.EqualError(t, err, "common banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).
		Return(user2.UnifiedStateProcess, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		utaProcessBan: true,
	}, md)
	assert.EqualError(t, err, "uta process banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user2.UnifiedTradingTag)).
		Return(user2.UnifiedStateFail, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &openapiRule{
		utaProcessBan: true,
	}, md)
	assert.NoError(t, err)
}

func TestTradeCheck(t *testing.T) {
	convey.Convey("TestTradeCheck", t, func() {
		// query member tag
		rctx, md := test.NewReqCtx()
		aa, ctrl, _ := makeOpenApi(t)
		defer ctrl.Finish()
		md.UID = 100

		status := &ban.UserStatusWrap{}
		as := mock.NewMockAccountIface(ctrl)
		aa.as = as

		p := gomonkey.ApplyFuncReturn(ban.TradeCheckSingleSymbol, nil)
		defer p.Reset()
		// tradeCheck=true
		err := aa.tradeCheck(rctx, &openapiRule{
			tradeCheck:      true,
			tradeCheckCfg:   &tradeCheckCfg{},
			batchTradeCheck: map[string]struct{}{},
		}, md, status)
		convey.So(err, convey.ShouldBeNil)

		p.ApplyFuncReturn(ban.TradeCheckBatchSymbol, "xxx", nil)
		// len(rule.batchTradeCheck) > 0
		md.Route.AppName = "options"
		err = aa.tradeCheck(rctx, &openapiRule{
			batchTradeCheck:    map[string]struct{}{"options": {}},
			batchTradeCheckCfg: map[string]*tradeCheckCfg{"options": {SymbolField: "122"}},
		}, md, status)
		convey.So(err, convey.ShouldBeNil)
		convey.So(md.BatchBan, convey.ShouldEqual, "xxx")

		// len(rule.batchTradeCheck) > 0 && ok==false
		md.Route.AppName = "options22"
		md.BatchBan = ""
		err = aa.tradeCheck(rctx, &openapiRule{
			batchTradeCheck:    map[string]struct{}{"options": {}},
			batchTradeCheckCfg: map[string]*tradeCheckCfg{"options": {SymbolField: "122"}},
		}, md, status)
		convey.So(err, convey.ShouldBeNil)
		convey.So(md.BatchBan, convey.ShouldEqual, "")
	})
}

func TestHandlerMemberTags(t *testing.T) {
	gmetric.Init("TestHandlerMemberTags")
	// query member tag
	_, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	defer ctrl.Finish()

	assert.Equal(t, filter.OpenAPIFilterKey, aa.GetName())

	md.UID = 100
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("123")).
		Return("", errors.New("ee"))
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("456")).
		Return("789", nil)
	aa.handlerMemberTags(context.Background(), []string{"123", "456"}, md)
	assert.Equal(t, "789", md.MemberTags["456"])
	assert.Equal(t, sUser.MemberTagFailed, md.MemberTags["123"])

}

func TestHandleAccountID(t *testing.T) {
	gmetric.Init("TestHandleAccountID")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	defer ctrl.Finish()

	md.UID = 100
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return errors.New("getAid failed")
	})
	err := aa.handleAccountID(rctx, md, "test", &openapiRule{})
	p.Reset()
	assert.EqualError(t, err, "getAid failed")

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return nil
	})
	m.EXPECT().GetBizAccountIDByApps(gomock.Any(), gomock.Eq(int64(100)),
		gomock.Eq(11), gomock.Eq("123"), gomock.Eq("321")).
		Return([]int64{100, 200}, []error{nil, nil})
	err = aa.handleAccountID(rctx, md, "test", &openapiRule{
		bizType:  11,
		aidQuery: []string{"123", "321"},
	})
	assert.NoError(t, err)

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
		return nil
	})
	m.EXPECT().GetBizAccountIDByApps(gomock.Any(), gomock.Eq(int64(100)),
		gomock.Eq(11), gomock.Eq("123"), gomock.Eq("321")).
		Return([]int64{0, 0}, []error{errors.New("www"), nil})
	err = aa.handleAccountID(rctx, md, "test", &openapiRule{
		bizType:  11,
		aidQuery: []string{"123", "321"},
	})
	assert.EqualError(t, err, "www")

	p.Reset()
}

func TestHandleSuiInfo(t *testing.T) {
	gmetric.Init("TestHandleSuiInfo")
	// query member tag
	_, md := test.NewReqCtx()
	aa, ctrl, m := makeOpenApi(t)
	defer ctrl.Finish()

	// uid < 0
	md.UID = -1
	aa.handleSuiInfo(context.Background(), md)
	assert.Equal(t, "", md.SuiKyc)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiKycTag)).
		Return("", errors.New("ee"))

	md.UID = 100
	aa.handleSuiInfo(context.Background(), md)
	assert.Equal(t, "", md.SuiKyc)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiKycTag)).
		Return("123", errors.New("ee"))

	md.UID = 100
	aa.handleSuiInfo(context.Background(), md)
	assert.Equal(t, "", md.SuiKyc)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiKycTag)).
		Return("123", nil)
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiProtoTag)).
		Return("", errors.New("ee"))

	md.UID = 100
	aa.handleSuiInfo(context.Background(), md)
	assert.Equal(t, "123", md.SuiKyc)
	assert.Equal(t, "", md.SuiProto)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiKycTag)).
		Return("123", nil)
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(suiProtoTag)).
		Return("321", nil)

	md.UID = 100
	aa.handleSuiInfo(context.Background(), md)
	assert.Equal(t, "123", md.SuiKyc)
	assert.Equal(t, "321", md.SuiProto)

}

func TestGetSmpGroup(t *testing.T) {
	gmetric.Init("TestGetSmpGroup")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	defer ctrl.Finish()

	err := aa.getSmpGroup(rctx, false, md)
	assert.NoError(t, err)

	md.UID = 1000
	p := gomonkey.ApplyFuncReturn(smp.GetGrouper, nil, errors.New("xxxxx"))
	err = aa.getSmpGroup(rctx, true, md)
	assert.NoError(t, err)

	p.Reset()

	smpg := mock.NewMockGrouper(ctrl)
	md.UID = 1000
	p.ApplyFuncReturn(smp.GetGrouper, smpg, nil)
	smpg.EXPECT().GetGroup(gomock.Any(), gomock.Eq(int64(1000))).Return(int32(123), errors.New("xxxxx"))
	err = aa.getSmpGroup(rctx, true, md)
	assert.NoError(t, err)
	assert.Equal(t, int32(123), md.SmpGroup)
	p.Reset()
	ctrl.Finish()
}

func Test_limiterFlagParse(t *testing.T) {
	convey.Convey("test limiterFlagParse", t, func() {
		_, err := limiterFlagParse(context.Background(), []string{"args", "--emptyArgs=123"})
		convey.So(err, convey.ShouldNotBeNil)

		patch := gomonkey.ApplyFunc(smp.GetGrouper, func(ctx context.Context) (gsmp.Grouper, error) {
			return nil, errors.New("mock err")
		})
		defer patch.Reset()
		_, err = limiterFlagParse(context.Background(), []string{"args", "--bizType=0", "--accountIDQuery=futures,spot",
			"--smpGroup=true", "--suiInfo=true", "--memberTags=tag1,tag2", "--batchTradeCheck=futures,spot"})
		convey.So(err, convey.ShouldBeNil)
	})
}

func Test_Init(t *testing.T) {
	convey.Convey("test openapi init", t, func() {
		patch := gomonkey.ApplyFunc(user2.NewAccountService, func() (user2.AccountIface, error) {
			return nil, nil
		})
		patch.Reset()
		patch2 := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch2.Reset()

		Init()
		op := &openapi{}
		err := op.Init(context.Background())
		convey.So(err, convey.ShouldBeNil)

		args := []string{"args0"}
		err = op.Init(context.Background(), args...)
		convey.So(err, convey.ShouldBeNil)

		args = []string{"args0", "--empty=1"}
		err = op.Init(context.Background(), args...)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

type mockAs struct {
	aids                         []int64
	memberTag                    string
	getAccountIDErr              error
	queryMemberTagERR            error
	getUnifiedMarginAccountIDERR error
	getBizAccountIDByAppsERR     []error
}

func (m *mockAs) GetAccountID(ctx context.Context, uid int64, accountType string, bizType int) (int64, error) {
	return m.aids[0], m.getAccountIDErr
}

func (m *mockAs) GetBizAccountIDByApps(_ context.Context, _ int64, _ int, _ ...string) (aids []int64, errs []error) {
	return m.aids, m.getBizAccountIDByAppsERR
}

func (m *mockAs) QueryMemberTag(_ context.Context, _ int64, _ string) (string, error) {
	return m.memberTag, m.queryMemberTagERR
}

func (m *mockAs) GetUnifiedMarginAccountID(_ context.Context, _ int64, _ int) (aid int64, err error) {
	return m.aids[0], m.getUnifiedMarginAccountIDERR
}

func (m *mockAs) GetUnifiedTradingAccountID(_ context.Context, _ int64, _ int) (aid int64, err error) {
	return m.aids[0], m.getUnifiedMarginAccountIDERR
}

func makeOpenApi(t *testing.T) (*openapi, *gomock.Controller, *mock.MockAccountIface) {
	aa := &openapi{}
	ctrl := gomock.NewController(t)
	m := mock.NewMockAccountIface(ctrl)
	aa.as = m
	return aa, ctrl, m
}

type mockBan struct{}

func (m *mockBan) GetMemberStatus(ctx context.Context, uid int64) (*ban.UserStatusWrap, error) {
	return nil, errors.New("mock err")
}

func (m *mockBan) CheckStatus(ctx context.Context, uid int64) (*ban.UserStatusWrap, error) {
	return nil, errors.New("mock err")
}

func (m *mockBan) VerifyTrade(ctx context.Context, uid int64, app string, status *ban.UserStatusWrap, opts ...ban.Option) (bool, error) {
	if uid == 1 {
		return false, berror.NewBizErr(123, "")
	}

	if uid == 2 {
		return false, nil
	}

	return true, nil
}
