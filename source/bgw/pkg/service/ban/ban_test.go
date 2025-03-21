package ban

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/coocood/freecache"
	"google.golang.org/grpc"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/sarama"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
	jsoniter "github.com/json-iterator/go"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
)

func TestInit(t *testing.T) {
	convey.Convey("test init", t, func() {
		SetBanService(nil)
		cfg := Config{}
		err := Init(cfg)
		convey.So(err, convey.ShouldNotBeNil)

		cfg.RpcConf = config.Global.BanServicePrivate
		cfg.KafkaConf = config.Global.KafkaCli
		patch1 := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch1.Reset()

		bs, err := GetBanService()
		convey.So(bs, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldBeNil)

		patch := gomonkey.ApplyFunc(zrpc.NewClient, func(c zrpc.RpcClientConf, options ...zrpc.ClientOption) (zrpc.Client, error) {
			return &mockCli{}, nil
		})
		defer patch.Reset()
		err = Init(cfg)
		convey.So(err, convey.ShouldBeNil)
	})
}

type mockCli struct{}

func (m *mockCli) Conn() grpc.ClientConnInterface {
	return nil
}

func TestBanService_GetMemberStatus2(t *testing.T) {
	convey.Convey("test GetMemberStatus2", t, func() {
		bs := &banService{
			cache: freecache.NewCache(1000),
		}

		gomonkey.ApplyFunc(ban.NewBanInternalClient, func(connInterface grpc.ClientConnInterface) ban.BanInternalClient {
			return &mockCli{}
		})

		_, err := bs.GetMemberStatus(context.Background(), 10)
		convey.So(err, convey.ShouldNotBeNil)

		_, err = bs.GetMemberStatus(context.Background(), 1)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestBanService_CheckStatus(t *testing.T) {
	convey.Convey("test CheckStatus", t, func() {
		bs := &banService{}

		patch := gomonkey.ApplyFunc((*banService).GetMemberStatus, func(b *banService, c context.Context, u int64) (*UserStatusWrap, error) {
			if u == 1 {
				return &UserStatusWrap{LoginBanType: int32(BantypeLogin)}, nil
			}
			if u == 2 {
				return &UserStatusWrap{LoginBanType: 5}, nil
			}
			return nil, errors.New("mock err")
		})
		defer patch.Reset()

		_, err := bs.CheckStatus(context.Background(), 1)
		convey.So(err, convey.ShouldNotBeNil)

		_, err = bs.CheckStatus(context.Background(), 2)
		convey.So(err, convey.ShouldBeNil)

		_, err = bs.CheckStatus(context.Background(), 32)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestGetErrFromBanItem(t *testing.T) {
	convey.Convey("TestGetErrFromBanItem", t, func() {
		err := getErrFromBanItem(nil, nil)
		convey.So(err, convey.ShouldBeNil)

		err = getErrFromBanItem(&ban.UserStatus_BanItem{}, berror.ErrDefault)
		convey.So(err, convey.ShouldEqual, berror.ErrDefault)

		err = getErrFromBanItem(&ban.UserStatus_BanItem{
			ErrorCode:  18001,
			ReasonText: "errorTest",
		}, berror.ErrDefault)

		convey.So(err, convey.ShouldEqual, berror.NewBizErr(18001, "errorTest"))
	})
}

func TestGetErrFromBanItemMap(t *testing.T) {
	convey.Convey("TestGetErrFromBanItemMap", t, func() {
		err := getErrFromBanItemMap(BantypeUnspecified, berror.ErrDefault, nil)
		convey.So(err, convey.ShouldEqual, berror.ErrDefault)

		banItemMap := map[BanType]*ban.UserStatus_BanItem{
			BantypeUsdcPerpetualAllKo: {
				ErrorCode:  18001,
				ReasonText: "errorTest",
			},
		}
		err = getErrFromBanItemMap(BantypeUsdcPerpetualAllKo, berror.ErrDefault, banItemMap)
		convey.So(err, convey.ShouldEqual, berror.NewBizErr(18001, "errorTest"))

		banItemMap = map[BanType]*ban.UserStatus_BanItem{
			BantypeUsdcFutureAllKo: {
				ErrorCode:  18002,
				ReasonText: "errorTest",
			},
		}
		err = getErrFromBanItemMap(BantypeUsdcFutureAllKo, berror.ErrDefault, banItemMap)
		convey.So(err, convey.ShouldEqual, berror.NewBizErr(18002, "errorTest"))

		err = getErrFromBanItemMap(BantypeSpotAllKo, berror.ErrDefault, banItemMap)
		convey.So(err, convey.ShouldEqual, berror.ErrDefault)
	})
}

func TestBan(t *testing.T) {
	gmetric.Init("test")
	galert.SetDefault(mockAlert{})
	t.Run("handleChtMemberBannedMessage", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		mm := &gkafka.Message{}

		// fixme assert？
		bbs.handleChtMemberBannedMessage(context.Background(), mm)

		cm := &ChtBannedMessage{
			Uids: []int{1, 2, 3},
		}
		bb, _ := util.JsonMarshal(cm)
		mm.Value = bb
		bbs.handleChtMemberBannedMessage(context.Background(), mm)

	})
	t.Run("banOnErr", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		bbs.banOnErr(&sarama.ConsumerError{Err: errors.New("212")})
	})
	t.Run("futuresTradeCheck", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		br, err := bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "usdt_all",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.Equal(t, berror.ErrOpenAPIUserUsdtAllBanned, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "symbol_BTCUSDT",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.Equal(t, berror.ErrOpenAPIUserAllBanned, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "symbol_BTCUSDT_lu",
				},
			},
		}, 11, "symbol_BTCUSDT_lu", true)
		assert.NoError(t, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "symbol",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.NoError(t, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "upgrade",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.Equal(t, berror.ErrTradeCheckUTAProcessBanned, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "login",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.NoError(t, err)
		assert.Equal(t, false, br)
		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "lighten_up",
				},
			},
		}, 11, "BTCUSDTx", true)
		assert.NoError(t, err)
		assert.Equal(t, true, br)

		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  DBUType,
					TagName:  TradeTag,
					TagValue: "usdc_all_ko",
				},
			},
		}, 11, "BTCUSDT", true)
		assert.NoError(t, err)
		assert.Equal(t, false, br)

		br, err = bbs.futuresTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  DBUType,
					TagName:  TradeTag,
					TagValue: USDCPERPETUALALLKO,
				},
			},
		}, 11, "BTCPERP", true)
		assert.Equal(t, berror.ErrOpenAPIUserUsdtAllBanned, err)
		assert.Equal(t, false, br)
	})
	t.Run("spotTradeCheck", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		err = bbs.spotTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  TradeType,
					TagName:  TradeTag,
					TagValue: AllTrade,
				},
			},
		})
		assert.Equal(t, berror.ErrOpenAPIUserLoginBanned, err)
		err = bbs.spotTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  FbuType,
					TagName:  TradeTag,
					TagValue: "usdt_all",
				},
			},
		})
		assert.Nil(t, err)

		err = bbs.spotTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  DBUType,
					TagName:  TradeTag,
					TagValue: "spot_all_ko",
				},
			},
		})
		assert.Equal(t, berror.ErrOpenAPIUserLoginBanned, err)

		err = bbs.spotTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  DBUType,
					TagName:  TradeTag,
					TagValue: "spot_all",
				},
			},
		})
		assert.EqualError(t, berror.ErrOpenAPIUserLoginBanned, "User has been banned.")
	})
	t.Run("optionTradeCheck", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		o := &Options{}
		ro, err := bbs.optionTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  UTAType,
					TagName:  TradeTag,
					TagValue: AllTrade,
				},
			},
		}, o)
		assert.Equal(t, berror.ErrOpenAPIUserUsdtAllBanned, err)
		assert.Equal(t, false, ro)
		ro, err = bbs.optionTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  UTAType,
					TagName:  TradeTag,
					TagValue: LightenUp,
				},
			},
		}, o)
		assert.Nil(t, err)
		assert.Equal(t, true, ro)

		ro, err = bbs.optionTradeCheck(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  DBUType,
					TagName:  TradeTag,
					TagValue: OptionsAllKO,
				},
			},
		}, o)
		assert.Equal(t, berror.ErrOpenAPIUserUsdtAllBanned, err)
		assert.Equal(t, false, ro)
	})
	t.Run("parseBanStatus", func(t *testing.T) {
		bs, err := GetBanService()
		assert.NoError(t, err)
		bbs := bs.(*banService)
		bss, bt := bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  LoginType,
					TagName:  LoginTag,
					TagValue: AllTrade,
				},
			},
		}, 11)
		assert.Equal(t, BantypeAllTrade, bt)
		assert.Equal(t, banStatus{loginBanStatus: int32(MemberLoginStatus_LOGIN_BAN), tradeBanStatus: 1, withdrawStatus: 1}, bss)
		bss, bt = bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  LoginType,
					TagName:  LoginTag,
					TagValue: "1212",
				},
			},
		}, 11)
		assert.Equal(t, BantypeUnspecified, bt)
		assert.Equal(t, banStatus{loginBanStatus: int32(MemberLoginStatus_LOGIN_UNKNOWN), tradeBanStatus: 1, withdrawStatus: 1}, bss)
		bss, bt = bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  WithdrawType,
					TagName:  WithdrawTag,
					TagValue: AllTrade,
				},
			},
		}, 11)
		assert.Equal(t, BantypeUnspecified, bt)
		assert.Equal(t, banStatus{loginBanStatus: 1, tradeBanStatus: 1, withdrawStatus: int32(MemberWithdrawStatus_WITHDRAW_BAN)}, bss)
		bss, bt = bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  WithdrawType,
					TagName:  WithdrawTag,
					TagValue: "121212",
				},
			},
		}, 11)
		assert.Equal(t, BantypeUnspecified, bt)
		assert.Equal(t, banStatus{loginBanStatus: 1, tradeBanStatus: 1, withdrawStatus: int32(MemberWithdrawStatus_WITHDRAW_UNKNOWN)}, bss)
		bss, bt = bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  TradeType,
					TagName:  TradeTag,
					TagValue: AllTrade,
				},
			},
		}, 11)
		assert.Equal(t, BantypeUnspecified, bt)
		assert.Equal(t, banStatus{loginBanStatus: 1, tradeBanStatus: int32(MemberTradeStatus_TRADE_BAN), withdrawStatus: 1}, bss)
		bss, bt = bbs.parseBanStatus(context.Background(), &UserStatus{
			BanItems: []*ban.UserStatus_BanItem{
				{
					BizType:  TradeType,
					TagName:  TradeTag,
					TagValue: "1212",
				},
			},
		}, 11)
		assert.Equal(t, BantypeUnspecified, bt)
		assert.Equal(t, banStatus{loginBanStatus: 1, tradeBanStatus: int32(MemberTradeStatus_TRADE_UNKNOWN), withdrawStatus: 1}, bss)
	})
}

func TestBanService_GetMemberStatus(t *testing.T) {
	convey.Convey("TestBanService_GetMemberStatus", t, func() {
		svc, err := GetBanService()
		convey.So(err, convey.ShouldBeNil)
		convey.So(svc, convey.ShouldNotBeNil)

		convey.Convey("TestBanService_GetMemberStatus ", func() {
			// mock and hit cache
			key := "111_banned_info"
			needCacheMsg := &userStatusInternal{
				LoginStatus:    1,
				WithdrawStatus: 1,
				TradeStatus:    1,
				LoginBanType:   1,
				UserState:      []byte{},
			}
			data, err := jsoniter.Marshal(needCacheMsg)
			convey.So(err, convey.ShouldBeNil)
			err = defaultBanService.(*banService).cache.Set([]byte(key), data, 60)
			convey.So(err, convey.ShouldBeNil)

			status, err := defaultBanService.GetMemberStatus(context.Background(), 111)
			convey.So(err, convey.ShouldBeNil)
			convey.So(status, convey.ShouldNotBeNil)
			convey.So(status.LoginStatus, convey.ShouldEqual, 1)
		})

	})
}

func TestBanService_VerifyTrade(t *testing.T) {

	convey.Convey("TestBanService_VerifyTrade", t, func() {
		_, _ = GetBanService()

		convey.Convey("TestBanService_VerifyTrade with wrong app", func() {
			trade, err := defaultBanService.VerifyTrade(context.Background(), 111, "1", &UserStatusWrap{
				LoginStatus:    1,
				WithdrawStatus: 1,
				TradeStatus:    1,
				LoginBanType:   1,
				UserState:      &UserStatus{BanItems: []*ban.UserStatus_BanItem{}},
			}, WithSymbol("BTCUSDT"), WithSiteAPI(true))
			convey.So(err, convey.ShouldBeNil)
			convey.So(trade, convey.ShouldBeFalse)
		})

		convey.Convey("TestBanService_VerifyTrade with future", func() {
			trade, err := defaultBanService.VerifyTrade(context.Background(), 111, constant.AppTypeFUTURES, &UserStatusWrap{
				LoginStatus:    1,
				WithdrawStatus: 1,
				TradeStatus:    1,
				LoginBanType:   1,
				UserState:      &UserStatus{BanItems: []*ban.UserStatus_BanItem{}},
			}, WithSymbol("BTCUSDT"), WithSiteAPI(true))
			convey.So(err, convey.ShouldBeNil)
			convey.So(trade, convey.ShouldBeFalse)
		})

		convey.Convey("TestBanService_VerifyTrade with spot", func() {
			trade, err := defaultBanService.VerifyTrade(context.Background(), 111, constant.AppTypeSPOT, &UserStatusWrap{
				LoginStatus:    1,
				WithdrawStatus: 1,
				TradeStatus:    1,
				LoginBanType:   1,
				UserState:      &UserStatus{BanItems: []*ban.UserStatus_BanItem{}},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(trade, convey.ShouldBeFalse)
		})

		convey.Convey("TestBanService_VerifyTrade with option", func() {
			trade, err := defaultBanService.VerifyTrade(context.Background(), 111, constant.AppTypeOPTION, &UserStatusWrap{
				LoginStatus:    1,
				WithdrawStatus: 1,
				TradeStatus:    1,
				LoginBanType:   1,
				UserState:      &UserStatus{BanItems: []*ban.UserStatus_BanItem{}},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(trade, convey.ShouldBeFalse)
		})

	})

}

func TestDiagnosis(t *testing.T) {
	convey.Convey("Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		defer p.Reset()
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)

		dig := diagnose{}
		convey.So(dig.Key(), convey.ShouldEqual, "ban_service_private")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp["kafka"], convey.ShouldEqual, result)
		convey.So(resp["grpc"], convey.ShouldEqual, result)
		convey.So(err, convey.ShouldBeNil)
	})
}

type mockAlert struct {
}

func (m mockAlert) Alert(_ context.Context, _ galert.Level, _ string, _ ...galert.Option) {
}

type mockBanCli struct{}

// 获取封禁区域
func (m *mockCli) GetBanAreas(ctx context.Context, in *ban.GetBanAreasRequest, opts ...grpc.CallOption) (*ban.GetBanAreasResponse, error) {
	return nil, nil
}

// 封禁
func (m *mockCli) EnableBan(ctx context.Context, in *ban.EnableBanRequest, opts ...grpc.CallOption) (*ban.EnableBanResponse, error) {
	return nil, nil
}

// 解禁
func (m *mockCli) DisableBan(ctx context.Context, in *ban.DisableBanRequest, opts ...grpc.CallOption) (*ban.DisableBanResponse, error) {
	return nil, nil
}

// 查询用户封禁状态
func (m *mockCli) QueryBanStatus(ctx context.Context, in *ban.QueryBanStatusRequest, opts ...grpc.CallOption) (*ban.QueryBanStatusResponse, error) {
	return nil, nil
}

// 查询所有封禁的业务项
func (m *mockCli) QueryBanBizItems(ctx context.Context, in *ban.QueryBanBizItemsRequest, opts ...grpc.CallOption) (*ban.QueryBanBizItemsResponse, error) {
	return nil, nil
}

// 查询封禁的业务项标签状态
func (m *mockCli) QueryBanSceneStatus(ctx context.Context, in *ban.QueryBanSceneStatusRequest, opts ...grpc.CallOption) (*ban.QueryBanSceneStatusResponse, error) {
	return nil, nil
}

// 根据时间查询有状态更新的用户的封禁状态
func (m *mockCli) BatchQueryRenewStatus(ctx context.Context, in *ban.BatchQueryRenewStatusRequest, opts ...grpc.CallOption) (*ban.BatchQueryRenewStatusResponse, error) {
	return nil, nil
}

// 批量查询用户封禁状态
func (m *mockCli) BatchQueryBanStatus(ctx context.Context, in *ban.BatchQueryBanStatusRequest, opts ...grpc.CallOption) (*ban.BatchQueryBanStatusResponse, error) {

	if in.Uids[0] == 1 {
		return &ban.BatchQueryBanStatusResponse{
			UserStatusItems: []*ban.UserStatus{
				{Uid: 1},
			},
		}, nil
	}
	return nil, errors.New("mock err")
}
