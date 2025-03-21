package auth

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"git.bybit.com/svc/go/pkg/bconst"
	"git.bybit.com/svc/mod/pkg/bproto"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	gmetadata "bgw/pkg/server/metadata"
	"bgw/pkg/service/ban"
	"bgw/pkg/service/dynconfig"
	"bgw/pkg/service/masque"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/service/user"
	"bgw/pkg/test"
	"bgw/pkg/test/mock"

	oauthv1 "code.bydev.io/cht/backend-bj/user-service/buf-user-gen.git/pkg/bybit/oauth/v1"

	"github.com/tj/assert"
)

func TestGetRule(t *testing.T) {
	Init()
	a := &auth{
		rules: sync.Map{},
	}

	r := a.getRule(gmetadata.RouteKey{})
	assert.Nil(t, r)

	a.rules.Store(gmetadata.RouteKey{}.String(), &authRule{})
	r = a.getRule(gmetadata.RouteKey{})
	assert.NotNil(t, r)
}

func TestCheckOauth(t *testing.T) {
	rctx, md := test.NewReqCtx()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	os := mock.NewMockOauthIface(ctrl)
	a := &auth{
		os:    os,
		rules: sync.Map{},
	}
	resp, err := a.checkOAuth(rctx, &authRule{}, md, "")
	assert.Nil(t, resp)
	assert.Equal(t, berror.ErrAuthVerifyFailed, err)

	os.EXPECT().OAuth(gomock.Any(), gomock.Eq("22")).Return(nil, errors.New("xxx"))
	resp, err = a.checkOAuth(rctx, &authRule{}, md, "22")
	assert.Nil(t, resp)
	assert.Equal(t, berror.ErrAuthVerifyFailed, err)

	os.EXPECT().OAuth(gomock.Any(), gomock.Eq("22")).Return(nil, nil)
	resp, err = a.checkOAuth(rctx, &authRule{}, md, "22")
	assert.Nil(t, resp)
	assert.Equal(t, berror.ErrAuthVerifyFailed, err)
	os.EXPECT().OAuth(gomock.Any(), gomock.Eq("22")).Return(&oauthv1.OAuthResponse{
		Error:    &oauthv1.Error{ErrorCode: 0},
		MemberId: 1212,
	}, nil)
	resp, err = a.checkOAuth(rctx, &authRule{}, md, "22")
	assert.NotNil(t, resp)
	assert.NoError(t, err)
}

func TestGetMemberID(t *testing.T) {
	rctx, md := test.NewReqCtx()
	u, b, err := GetMemberID(rctx, "")
	assert.Equal(t, false, b)
	assert.Equal(t, int64(0), u)
	assert.Equal(t, berror.ErrAuthVerifyFailed, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := mock.NewMockMasqueIface(ctrl)

	p := gomonkey.ApplyFuncReturn(masque.GetMasqueService, nil, errors.New("xxx"))
	u, b, err = GetMemberID(rctx, "121")
	assert.Equal(t, false, b)
	assert.Equal(t, int64(0), u)
	assert.EqualError(t, err, "xxx")
	p.Reset()

	p = gomonkey.ApplyFuncReturn(masque.GetMasqueService, ms, nil)
	ms.EXPECT().MasqueTokenInvoke(gomock.Any(), gomock.Eq(""), gomock.Eq("121"), gomock.Eq(""), gomock.Eq(masque.WeakAuth)).Return(nil, errors.New("sss"))
	u, b, err = GetMemberID(rctx, "121")
	assert.Equal(t, false, b)
	assert.Equal(t, int64(0), u)
	assert.EqualError(t, err, "sss")

	ms.EXPECT().MasqueTokenInvoke(gomock.Any(), gomock.Eq(""), gomock.Eq("121"), gomock.Eq(""), gomock.Eq(masque.WeakAuth)).Return(&masque.AuthResponse{
		UserId:  123,
		ExtInfo: map[string]string{parentUID: "222", subMemberTypeKey: "xxx"},
	}, nil)
	u, b, err = GetMemberID(rctx, "121")
	assert.Equal(t, false, b)
	assert.Equal(t, "xxx", md.MemberRelation)
	assert.Equal(t, false, md.IsDemoUID)
	assert.Equal(t, int64(123), u)
	assert.Equal(t, int64(123), md.UID)
	assert.NoError(t, err)
	p.Reset()
}

func TestInit(t *testing.T) {
	a := &auth{
		rules: sync.Map{},
	}
	err := a.Init(context.Background())
	assert.NoError(t, err)
	_, ok := a.rules.Load("11")
	assert.Equal(t, false, ok)

	p := gomonkey.ApplyFuncReturn(limiterFlagParse, nil, errors.New("xxx"))
	err = a.Init(context.Background(), "11")
	assert.EqualError(t, err, "xxx")
	_, ok = a.rules.Load("11")
	assert.Equal(t, false, ok)
	p.Reset()

	p = gomonkey.ApplyFuncReturn(limiterFlagParse, nil, nil).ApplyFuncReturn(dynconfig.GetBrokerIdLoader, nil, errors.New("xxxx"))
	err = a.Init(context.Background(), "11")
	assert.EqualError(t, err, "xxxx")
	_, ok = a.rules.Load("11")
	assert.Equal(t, true, ok)
	p.Reset()
}

func TestGetName(t *testing.T) {
	gmetric.Init("sss4")
	a := &auth{}
	assert.Equal(t, filter.AuthFilterKey, a.GetName())
}

func TestFlagParse(t *testing.T) {
	gmetric.Init("sss3")
	a := assert.New(t)
	args := []string{
		"HelloServer.ABC",
		"--bizType=1",
		"--allowGuest=true",
		"--suiInfo=true",
	}
	_, err := limiterFlagParse(context.Background(), args)
	a.NoError(err)
}

func TestHashToken(t *testing.T) {
	gmetric.Init("sss5")
	a := &auth{}
	token := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzA1NzU4MzUsInVzZXJfaWQiOjE5OTQ0NSwibm9uY2UiOiI2NThlOWUwZCJ9.W-4BljgmSW2XX38tOmyOYRfOajSOhCRdOjDAmrZ1ahHI3qqIQviPwdm_jqYrQIOB2ddhVRnRNZhifcCKovujJA"
	_, _ = a.hashToken(token)
}

func TestRouteInvalid(t *testing.T) {
	gmetric.Init("TestRouteInvalid")
	// route is invalid
	aa, ctrl, _, _ := makeAuth(t)
	rctx, md := test.NewReqCtx()
	md.Route = gmetadata.RouteKey{}
	err := aa.Do(nil)(rctx)
	ctrl.Finish()
	assert.Equal(t, "auth user route error", err.Error())
}

func TestRouteRuleNil(t *testing.T) {
	gmetric.Init("TestRouteRuleNil")
	// rule is nil
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	defer ctrl.Finish()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "invalid auth rule", err.Error())
}

func TestCheckLoginErr(t *testing.T) {
	gmetric.Init("TestCheckLoginErr")
	// check login err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return nil, berror.ErrAuthVerifyFailed
		})
	defer p.Reset()
	defer ctrl.Finish()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, berror.ErrAuthVerifyFailed, err)
}

func TestAuthExtInfo(t *testing.T) {
	gmetric.Init("TestAuthExtInfo")
	// AuthExtInfo true & HasNextToken true
	rctx, md := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func(aa *auth) error {
		return errors.New("xxx")
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:       123,
				NameSpace:    "xxx",
				HasNextToken: true,
				ExtInfo: map[string]string{
					parentUID:        "321",
					subMemberTypeKey: demoSubMember,
				},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		})
	defer ctrl.Finish()
	defer p.Reset()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "321", md.ParentUID)
	assert.Equal(t, "xxx", md.UserNameSpace)
	assert.Equal(t, demoSubMember, md.AuthExtInfo[subMemberTypeKey])
	assert.Equal(t, int64(123), md.UID)

	assert.Equal(t, "sss", *md.Intermediate.SecureToken)
	assert.Equal(t, "bbb", *md.Intermediate.WeakToken)

	assert.Error(t, err)
}

func TestAuthExtInfo2(t *testing.T) {
	gmetric.Init("TestAuthExtInfo2")
	// AuthExtInfo false & HasNextToken false & handleAccountID err
	rctx, md := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return errors.New("xxx")
	})
	defer p.Reset()
	defer ctrl.Finish()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "", md.ParentUID)
	assert.Equal(t, "xxx", md.UserNameSpace)
	assert.Equal(t, "", md.AuthExtInfo[subMemberTypeKey])
	assert.Equal(t, int64(123), md.UID)

	assert.Equal(t, (*string)(nil), md.Intermediate.SecureToken)
	assert.Equal(t, (*string)(nil), md.Intermediate.WeakToken)

	assert.Error(t, err)
}

func TestUnifiedTradingCheck_Main(t *testing.T) {
	// unifiedTradingCheck err
	gmetric.Init("TestUnifiedTradingCheck_Main")
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	defer ctrl.Finish()

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "unifiedTradingCheck", func() error {
		return errors.New("xxx")
	})
	defer p.Reset()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "xxx", err.Error())
}

func TestQueryMemberTag(t *testing.T) {
	gmetric.Init("TestQueryMemberTag")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
	defer ctrl.Finish()
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			suiInfo: true,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "unifiedTradingCheck", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "tradeCheck", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleSuiInfo", func(ctx context.Context,
		md *gmetadata.Metadata) {
	})
	m.EXPECT().
		QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Eq(user.UserSiteIDTag)).
		Return("100", nil)
	defer p.Reset()
	_ = aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.Equal(t, "100", md.UserSiteID)
}

func TestQueryMemberTagErr(t *testing.T) {
	gmetric.Init("TestQueryMemberTagErr")
	// query member tag err
	rctx, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
	defer ctrl.Finish()
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			suiInfo: true,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{"ExtInfoSiteID": "123"},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "unifiedTradingCheck", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "tradeCheck", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleSuiInfo", func(ctx context.Context,
		md *gmetadata.Metadata) {
	})
	m.EXPECT().
		QueryMemberTag(gomock.Any(), gomock.Eq(int64(123)), gomock.Eq(user.UserSiteIDTag)).
		Return("", errors.New("xxxx"))
	defer p.Reset()
	_ = aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.Equal(t, "123", md.UserSiteID)
}

func TestTradeCheckErr(t *testing.T) {
	gmetric.Init("TestTradeCheckErr")
	// trade check err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	defer ctrl.Finish()
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "unifiedTradingCheck", func() error {
		return nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "tradeCheck", func() error {
		return errors.New("xxx")
	})
	defer p.Reset()
	err := aa.Do(nil)(rctx)
	assert.Equal(t, "xxx", err.Error())
}

func TestCp(t *testing.T) {
	gmetric.Init("TestCp")
	// copytrade service error & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			copyTrade: true,
			copyTradeInfo: &user.CopyTradeInfo{
				AllowGuest: true,
			},
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return errors.New("xxx")
	}).ApplyFunc(user.GetCopyTradeService, func() (user.CopyTradeIface, error) {
		return nil, errors.New("error")
	})
	defer p.Reset()
	defer ctrl.Finish()
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)
	assert.NoError(t, err)
}

func TestCp1(t *testing.T) {
	gmetric.Init("TestCp1")
	// copytrade GetCopyTradeData error & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	cpt := mock.NewMockCopyTradeIface(ctrl)
	cpt.EXPECT().GetCopyTradeData(gomock.Any(), gomock.Any()).Return(nil, errors.New("xxx"))

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			copyTrade: true,
			copyTradeInfo: &user.CopyTradeInfo{
				AllowGuest: true,
			},
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return errors.New("xxx2")
	}).ApplyFunc(user.GetCopyTradeService, func() (user.CopyTradeIface, error) {
		return cpt, nil
	})
	defer p.Reset()
	defer ctrl.Finish()
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.NoError(t, err)
}

func TestCp2(t *testing.T) {
	gmetric.Init("TestCp2")
	// copytrade GetCopyTradeData error & copyTradeInfo nil & handleAccountID err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	cpt := mock.NewMockCopyTradeIface(ctrl)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			copyTrade:     true,
			copyTradeInfo: nil,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return errors.New("xxx3")
	}).ApplyFunc(user.GetCopyTradeService, func() (user.CopyTradeIface, error) {
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
	// copytrade GetCopyTradeData no err & handle accountid err
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _, _ := makeAuth(t)
	cpt := mock.NewMockCopyTradeIface(ctrl)
	cpt.EXPECT().GetCopyTradeData(gomock.Any(), gomock.Any()).Return(&user.CopyTrade{}, nil)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getRule", func(aa *auth) *authRule {
		return &authRule{
			copyTrade:     true,
			copyTradeInfo: nil,
		}
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "checkLogin",
		func(aa *auth) (resp *masque.AuthResponse, err error) {
			return &masque.AuthResponse{
				UserId:    123,
				NameSpace: "xxx",
				ExtInfo:   map[string]string{},
				NextToken: "sss",
				WeakToken: "bbb",
			}, nil
		}).ApplyFunc(user.GetCopyTradeService, func() (user.CopyTradeIface, error) {
		return cpt, nil
	}).ApplyPrivateMethod(reflect.TypeOf(aa), "handleAccountID", func() error {
		return errors.New("xxx2")
	})
	defer ctrl.Finish()
	defer p.Reset()
	err := aa.Do(func(rctx *fasthttp.RequestCtx) error {
		return nil
	})(rctx)

	assert.EqualError(t, err, "xxx2")
	assert.IsType(t, &user.CopyTrade{}, rctx.Value("copytrade"))
}

func TestCheckLogin(t *testing.T) {
	gmetric.Init("sss2")
	t.Run("token is nil", func(t *testing.T) {
		a, ctrl, _, _ := makeAuth(t)
		defer ctrl.Finish()
		token := ""
		rctx, md := test.NewReqCtx()
		_, err := a.checkLogin(rctx, &authRule{}, md, token)
		assert.Equal(t, berror.ErrAuthVerifyFailed, err)
		resp, err := a.checkLogin(rctx, &authRule{allowGuest: true}, md, token)
		assert.NoError(t, err)
		assert.IsType(t, new(masque.AuthResponse), resp)
	})
	t.Run("token masq err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockMasqueIface(ctrl)
		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(nil, errors.New("xxx"))
		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.WeakAuth)).
			Return(nil, errors.New("xxx"))
		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.RefreshToken)).
			Return(nil, errors.New("xxx"))

		a := &auth{
			ms: m,
		}
		token := "123"
		rctx, md := test.NewReqCtx()
		md.Extension.Platform = "pc"
		md.Extension.Referer = "pc"
		md.Path = "pccc"
		rctx.SetUserValue(constant.METADATA_CTX, md)

		_, err := a.checkLogin(rctx, &authRule{}, md, token)
		assert.Equal(t, "xxx", err.Error())
		_, err = a.checkLogin(rctx, &authRule{
			refreshToken: true,
		}, md, token)
		assert.Equal(t, "xxx", err.Error())
		_, err = a.checkLogin(rctx, &authRule{
			weakAuth: true,
		}, md, token)
		assert.Equal(t, "xxx", err.Error())
	})
	t.Run("token masq resp err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockMasqueIface(ctrl)
		e := bproto.NewError(100, "xxxxx")
		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error: e,
			}, nil)

		a := &auth{
			ms: m,
		}
		token := "123"
		rctx, md := test.NewReqCtx()
		md.Extension.Platform = "pc"
		md.Extension.Referer = "pc"
		md.Path = "pccc"
		tt := "secureToken"
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		rctx.SetUserValue(constant.METADATA_CTX, md)

		_, err := a.checkLogin(rctx, &authRule{}, md, token)
		assert.IsType(t, berror.UpStreamErr{}, err)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error: bproto.NewError(bconst.RpcErrorCodeFailed, "xxxxx"),
			}, nil)
		_, err = a.checkLogin(rctx, &authRule{}, md, token)
		assert.Equal(t, berror.ErrAuthVerifyFailed, err)
		assert.Equal(t, "", *md.Intermediate.SecureToken)
		assert.Equal(t, "", *md.Intermediate.WeakToken)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error: bproto.NewError(bconst.RpcErrorCodeFailed, "xxxxx"),
			}, nil)
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		resp, err := a.checkLogin(rctx, &authRule{
			allowGuest: true,
		}, md, token)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "secureToken", *md.Intermediate.SecureToken)
		assert.Equal(t, "secureToken", *md.Intermediate.WeakToken)
	})
	t.Run("token masq resp BrokerID > 0", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockMasqueIface(ctrl)
		a := &auth{
			ms: m,
		}
		token := "123"
		rctx, md := test.NewReqCtx()
		md.Extension.Platform = "pc"
		md.Extension.Referer = "pc"
		md.Path = "pccc"
		md.BrokerID = 120
		tt := "secureToken"
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		rctx.SetUserValue(constant.METADATA_CTX, md)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error:    nil,
				BrokerId: 120,
			}, nil)
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		resp, err := a.checkLogin(rctx, &authRule{}, md, token)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "secureToken", *md.Intermediate.SecureToken)
		assert.Equal(t, "secureToken", *md.Intermediate.WeakToken)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error:    nil,
				BrokerId: 121,
			}, nil)
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		resp, err = a.checkLogin(rctx, &authRule{}, md, token)
		assert.Equal(t, berror.ErrAuthVerifyFailed, err)
		assert.Nil(t, resp)
		assert.Equal(t, "", *md.Intermediate.SecureToken)
		assert.Equal(t, "", *md.Intermediate.WeakToken)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error:    nil,
				BrokerId: 121,
			}, nil)
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		resp, err = a.checkLogin(rctx, &authRule{
			allowGuest: true,
		}, md, token)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "secureToken", *md.Intermediate.SecureToken)
		assert.Equal(t, "secureToken", *md.Intermediate.WeakToken)
	})
	t.Run("token masq resp BrokerID <= 0", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMockMasqueIface(ctrl)
		a := &auth{
			ms: m,
		}
		token := "123"
		rctx, md := test.NewReqCtx()
		md.Extension.Platform = "pc"
		md.Extension.Referer = "pc"
		md.Path = "pccc"
		md.BrokerID = 0
		tt := "secureToken"
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		rctx.SetUserValue(constant.METADATA_CTX, md)

		m.EXPECT().
			MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
				gomock.Eq("123"), gomock.Eq("pcpccc"),
				gomock.Eq(masque.Auth)).
			Return(&masque.AuthResponse{
				Error:    nil,
				BrokerId: 120,
			}, nil)
		md.Intermediate.SecureToken = &tt
		md.Intermediate.WeakToken = &tt
		resp, err := a.checkLogin(rctx, &authRule{}, md, token)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "secureToken", *md.Intermediate.SecureToken)
		assert.Equal(t, "secureToken", *md.Intermediate.WeakToken)

		t.Run("GetBrokerIdLoader err", func(t *testing.T) {
			m.EXPECT().
				MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
					gomock.Eq("123"), gomock.Eq("pcpccc"),
					gomock.Eq(masque.Auth)).
				Return(&masque.AuthResponse{
					Error:    nil,
					BrokerId: 0,
				}, nil)
			p := gomonkey.ApplyFunc(dynconfig.GetBrokerIdLoader, func(ctx context.Context) (*dynconfig.BrokerIdLoader, error) {
				return nil, errors.New("GetBrokerIdLoader err")
			})
			defer p.Reset()
			md.BrokerID = 0
			_, err = a.checkLogin(rctx, &authRule{}, md, token)
			assert.NoError(t, err)
		})
		t.Run("GetBrokerIdLoader deny true", func(t *testing.T) {
			m.EXPECT().
				MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
					gomock.Eq("123"), gomock.Eq("pcpccc"),
					gomock.Eq(masque.Auth)).
				Return(&masque.AuthResponse{
					Error:    nil,
					BrokerId: 0,
				}, nil)
			bl := &dynconfig.BrokerIdLoader{}
			p := gomonkey.ApplyFunc(dynconfig.GetBrokerIdLoader, func(ctx context.Context) (*dynconfig.BrokerIdLoader, error) {
				return bl, nil
			}).ApplyPrivateMethod(reflect.TypeOf(bl), "IsDeny", func(originFrom string, userBrokerId int, originSite string, userSite string) (bool, error) {
				return true, nil
			})
			defer p.Reset()
			md.BrokerID = 0
			resp, err = a.checkLogin(rctx, &authRule{}, md, token)
			assert.EqualError(t, err, berror.ErrAuthVerifyFailed.Error())
			assert.Equal(t, "", *md.Intermediate.SecureToken)
			assert.Equal(t, "", *md.Intermediate.WeakToken)
			assert.Nil(t, resp)
		})
		t.Run("GetBrokerIdLoader deny false", func(t *testing.T) {
			m.EXPECT().
				MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
					gomock.Eq("123"), gomock.Eq("pcpccc"),
					gomock.Eq(masque.Auth)).
				Return(&masque.AuthResponse{
					Error:    nil,
					BrokerId: 0,
				}, nil)
			bl := &dynconfig.BrokerIdLoader{}
			p := gomonkey.ApplyFunc(dynconfig.GetBrokerIdLoader, func(ctx context.Context) (*dynconfig.BrokerIdLoader, error) {
				return bl, nil
			}).ApplyPrivateMethod(reflect.TypeOf(bl), "IsDeny", func(originFrom string, userBrokerId int, originSite string, userSite string) (bool, error) {
				return false, nil
			})
			defer p.Reset()
			md.BrokerID = 0
			resp, err = a.checkLogin(rctx, &authRule{}, md, token)
			assert.NoError(t, err)
			assert.Equal(t, "", *md.Intermediate.SecureToken)
			assert.Equal(t, "", *md.Intermediate.WeakToken)
			assert.Equal(t, int32(0), resp.BrokerId)
		})
		t.Run("GetBrokerIdLoader deny err", func(t *testing.T) {
			m.EXPECT().
				MasqueTokenInvoke(gomock.Any(), gomock.Eq("pc"),
					gomock.Eq("123"), gomock.Eq("pcpccc"),
					gomock.Eq(masque.Auth)).
				Return(&masque.AuthResponse{
					Error:    nil,
					BrokerId: 0,
				}, nil)
			bl := &dynconfig.BrokerIdLoader{}
			p := gomonkey.ApplyFunc(dynconfig.GetBrokerIdLoader, func(ctx context.Context) (*dynconfig.BrokerIdLoader, error) {
				return bl, nil
			}).ApplyPrivateMethod(reflect.TypeOf(bl), "IsDeny", func(originFrom string, userBrokerId int, originSite string, userSite string) (bool, error) {
				return false, errors.New("xxx")
			})
			defer p.Reset()
			md.BrokerID = 0
			resp, err = a.checkLogin(rctx, &authRule{}, md, token)
			assert.NoError(t, err)
			assert.Equal(t, int32(0), resp.BrokerId)
		})
	})
}

func TestGetAid(t *testing.T) {
	gmetric.Init("TestGetAid")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, as, _ := makeAuth(t)
	defer ctrl.Finish()

	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Eq(user.UnifiedTradingTag)).Return("", errors.New("xxx"))
	err := aa.getAid(rctx, md, "hh", &authRule{
		unifiedTrading: true,
	})
	assert.EqualError(t, err, "xxx")

	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Eq(user.UnifiedTradingTag)).Return("user tag", nil)
	as.EXPECT().GetUnifiedTradingAccountID(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("xxx2"))
	err = aa.getAid(rctx, md, "hh", &authRule{
		unifiedTrading: true,
	})
	assert.EqualError(t, err, "xxx2")
	assert.Equal(t, "user tag", md.UaTag)

	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Eq(user.UnifiedTradingTag)).Return("user tag", nil)
	as.EXPECT().GetUnifiedTradingAccountID(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
	err = aa.getAid(rctx, md, "hh", &authRule{
		unifiedTrading: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "user tag", md.UaTag)
	assert.Equal(t, true, md.UnifiedTrading)
	assert.Equal(t, int64(10), md.UaID)
	assert.Equal(t, int64(10), md.AccountID)

	md.UaTag = ""
	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Eq(user.UnifiedTradingTag)).Return("", errors.New("xxx3"))
	err = aa.getAid(rctx, md, "hh", &authRule{
		utaStatus: true,
	})
	md.UnifiedTrading = false
	md.UaID = 0
	md.AccountID = 0

	assert.EqualError(t, err, "xxx3")
	assert.Equal(t, "", md.UaTag)
	assert.Equal(t, false, md.UnifiedTrading)
	assert.Equal(t, int64(0), md.UaID)
	assert.Equal(t, int64(0), md.AccountID)

	md.UaTag = ""
	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(md.UID), gomock.Eq(user.UnifiedTradingTag)).Return("uta", nil)
	as.EXPECT().GetAccountID(gomock.Any(), gomock.Eq(md.UID), gomock.Eq("hh"), gomock.Any()).Return(int64(0), errors.New("ccc"))
	err = aa.getAid(rctx, md, "hh", &authRule{
		utaStatus: true,
	})
	assert.EqualError(t, err, "ccc")
	assert.Equal(t, "uta", md.UaTag)
	assert.Equal(t, false, md.UnifiedTrading)
	assert.Equal(t, int64(0), md.UaID)
	assert.Equal(t, int64(0), md.AccountID)

	md.UaTag = ""
	as.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(md.UID), gomock.Eq(user.UnifiedTradingTag)).Return("uta", nil)
	as.EXPECT().GetAccountID(gomock.Any(), gomock.Eq(md.UID), gomock.Eq("hh"), gomock.Any()).Return(int64(20), nil)
	err = aa.getAid(rctx, md, "hh", &authRule{
		utaStatus: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "uta", md.UaTag)
	assert.Equal(t, false, md.UnifiedTrading)
	assert.Equal(t, int64(0), md.UaID)
	assert.Equal(t, int64(20), md.AccountID)

	as.EXPECT().GetUnifiedMarginAccountID(gomock.Any(), gomock.Eq(md.UID), gomock.Eq(1)).Return(int64(0), errors.New("ddd"))
	err = aa.getAid(rctx, md, "hh", &authRule{
		unified: true,
		bizType: 1,
	})
	assert.EqualError(t, err, "ddd")

	as.EXPECT().GetUnifiedMarginAccountID(gomock.Any(), gomock.Eq(md.UID), gomock.Eq(1)).Return(int64(10), nil)
	err = aa.getAid(rctx, md, "hh", &authRule{
		unified: true,
		bizType: 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, true, md.UnifiedMargin)
	assert.Equal(t, int64(10), md.AccountID)
	assert.Equal(t, int64(10), md.UnifiedID)

	rctx.Request.SetRequestURI(string(UnifiedPrivateURLPrefix))
	as.EXPECT().GetUnifiedMarginAccountID(gomock.Any(), gomock.Eq(md.UID), gomock.Eq(1)).Return(int64(0), nil)
	err = aa.getAid(rctx, md, "hh", &authRule{
		unified: true,
		bizType: 1,
	})
	assert.Equal(t, berror.ErrUnifiedMarginAccess, err)
}

func TestTradeCheck(t *testing.T) {
	convey.Convey("TestTradeCheck", t, func() {
		gmetric.Init("TestTradeCheck")
		// query member tag
		rctx, md := test.NewReqCtx()
		aa, ctrl, _, _ := makeAuth(t)
		defer ctrl.Finish()

		md.UID = 100
		// query member tag err
		as := mock.NewMockAccountIface(ctrl)
		aa.as = as

		// get ban svc return err
		p := gomonkey.ApplyFuncReturn(ban.GetBanService, nil, errors.New("xxx"))
		defer p.Reset()
		err := aa.tradeCheck(rctx, &authRule{tradeCheck: true}, md)
		convey.So(err.Error(), convey.ShouldEqual, "xxx")

		banmock := ban.NewMockBanServiceIface(ctrl)
		p.ApplyFuncReturn(ban.GetBanService, banmock, nil)

		// CheckStatus return err
		banmock.EXPECT().CheckStatus(gomock.Any(), gomock.Eq(int64(100))).Return(nil, errors.New("xxx"))
		err = aa.tradeCheck(rctx, &authRule{tradeCheck: true}, md)
		convey.So(err.Error(), convey.ShouldEqual, "xxx")

		// tradeCheck == true
		banmock.EXPECT().CheckStatus(gomock.Any(), gomock.Eq(int64(100))).Return(nil, nil).AnyTimes()

		p.ApplyFuncReturn(ban.TradeCheckSingleSymbol, errors.New("xxx"))
		err = aa.tradeCheck(rctx, &authRule{tradeCheck: true, tradeCheckCfg: &tradeCheckCfg{SymbolField: ""}}, md)
		convey.So(err.Error(), convey.ShouldEqual, "xxx")

		// batch tradeCheck, app not in batchTradeCheck
		p.ApplyFuncReturn(ban.TradeCheckBatchSymbol, "", nil)
		md.Route.AppName = "asas"
		md.BatchBan = ""
		err = aa.tradeCheck(rctx, &authRule{
			batchTradeCheck:    map[string]struct{}{"options": {}},
			batchTradeCheckCfg: map[string]*tradeCheckCfg{"options": {SymbolField: ""}},
		}, md)
		convey.So(err, convey.ShouldBeNil)

		// batch tradeCheck
		p.ApplyFuncReturn(ban.TradeCheckBatchSymbol, "ss", nil)
		md.Route.AppName = "options"
		md.BatchBan = ""
		err = aa.tradeCheck(rctx, &authRule{
			batchTradeCheck:    map[string]struct{}{"options": {}},
			batchTradeCheckCfg: map[string]*tradeCheckCfg{"options": {SymbolField: "xx"}},
		}, md)
		convey.So(err, convey.ShouldBeNil)
		convey.So(md.BatchBan, convey.ShouldEqual, "ss")

	})
}

func TestHandleAccountID(t *testing.T) {
	gmetric.Init("TestHandleAccountID")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
	defer ctrl.Finish()

	md.UID = 100
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *gmetadata.Metadata, appName string, rule *authRule) error {
		return errors.New("getAid failed")
	})

	err := aa.handleAccountID(rctx, md, "test", &authRule{})
	assert.EqualError(t, err, "getAid failed")
	p.Reset()

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *gmetadata.Metadata, appName string, rule *authRule) error {
		return nil
	})
	m.EXPECT().GetBizAccountIDByApps(gomock.Any(), gomock.Eq(int64(100)),
		gomock.Eq(11), gomock.Eq("123"), gomock.Eq("321")).
		Return([]int64{100, 200}, []error{nil, nil})
	err = aa.handleAccountID(rctx, md, "test", &authRule{
		bizType:  11,
		aidQuery: []string{"123", "321"},
	})
	assert.NoError(t, err)
	p.Reset()

	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(aa), "getAid", func(c *types.Ctx, md *gmetadata.Metadata, appName string, rule *authRule) error {
		return nil
	})
	m.EXPECT().GetBizAccountIDByApps(gomock.Any(), gomock.Eq(int64(100)),
		gomock.Eq(11), gomock.Eq("123"), gomock.Eq("321")).
		Return([]int64{0, 0}, []error{errors.New("www"), nil})
	err = aa.handleAccountID(rctx, md, "test", &authRule{
		bizType:  11,
		aidQuery: []string{"123", "321"},
	})
	assert.EqualError(t, err, "www")
	p.Reset()
}

func TestUnifiedTradingCheck(t *testing.T) {
	gmetric.Init("TestUnifiedTradingCheck")
	// query member tag
	rctx, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
	defer ctrl.Finish()

	// uid < 0
	md.UID = -1
	err := aa.unifiedTradingCheck(rctx, &authRule{}, md)
	assert.NoError(t, err)

	// !rule.utaProcessBan && rule.unifiedTradingCheck == ""
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		utaProcessBan:       false,
		unifiedTradingCheck: "",
	}, md)
	assert.NoError(t, err)

	// QueryMemberTag err
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user.UnifiedTradingTag)).
		Return("", errors.New("ee"))
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		utaProcessBan: true,
	}, md)
	assert.NoError(t, err)

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateSuccess, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		unifiedTradingCheck: utaBan,
	}, md)
	assert.EqualError(t, err, "uta banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateFail, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		unifiedTradingCheck: comBan,
	}, md)
	assert.EqualError(t, err, "common banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateProcess, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		utaProcessBan: true,
	}, md)
	assert.EqualError(t, err, "uta process banned")

	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateFail, nil)
	md.UID = 100
	err = aa.unifiedTradingCheck(rctx, &authRule{
		utaProcessBan: true,
	}, md)
	assert.NoError(t, err)
}

func TestHandleSuiInfo(t *testing.T) {
	gmetric.Init("TestHandleSuiInfo")
	// query member tag
	_, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
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

func TestHandlerMemberTags(t *testing.T) {
	gmetric.Init("TestHandlerMemberTags")
	// query member tag
	_, md := test.NewReqCtx()
	aa, ctrl, m, _ := makeAuth(t)
	defer ctrl.Finish()

	md.UID = 100
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("123")).
		Return("", errors.New("ee"))
	m.EXPECT().QueryMemberTag(gomock.Any(), gomock.Eq(int64(100)), gomock.Eq("456")).
		Return("789", nil)
	aa.handlerMemberTags(context.Background(), []string{"123", "456"}, md)
	assert.Equal(t, "789", md.MemberTags["456"])
	assert.Equal(t, user.MemberTagFailed, md.MemberTags["123"])

}

func TestParseFlagTradeCheck(t *testing.T) {
	gmetric.Init("TestParseFlagTradeCheck")
	a := assert.New(t)
	_, err := limiterFlagParse(context.Background(), []string{"auth", "--tradeCheck=true", "--tradeCheckCfg=0"})
	a.Equal("json: cannot unmarshal number into Go value of type auth.tradeCheckCfg", err.Error())
	r, err := limiterFlagParse(context.Background(), []string{"auth", "--tradeCheck=true", "--tradeCheckCfg={\"symbolField\":\"12\"}"})
	a.NoError(err)
	a.Equal("12", r.tradeCheckCfg.SymbolField)
	// init symbol config err
	p := gomonkey.ApplyFuncReturn(symbolconfig.InitSymbolConfig, errors.New("xxx"))
	defer p.Reset()
	_, err = limiterFlagParse(context.Background(), []string{"auth", "--tradeCheck=true"})
	a.Equal("xxx", err.Error())
}

func TestParseFlagBatchTradeCheck(t *testing.T) {
	gmetric.Init("TestParseFlagBatchTradeCheck")
	a := assert.New(t)
	_, err := limiterFlagParse(context.Background(), []string{"auth", "--batchTradeCheck=options", "--batchTradeCheckCfg=0"})
	a.Equal("json: cannot unmarshal number into Go value of type map[string]*auth.tradeCheckCfg", err.Error())
	r, err := limiterFlagParse(context.Background(), []string{"auth", "--batchTradeCheck=options", "--batchTradeCheckCfg={\"options\":{\"symbolField\":\"12\"}}"})
	a.NoError(err)
	a.Equal("12", r.batchTradeCheckCfg["options"].SymbolField)
}

func Test_limiterFlagParse(t *testing.T) {
	convey.Convey("test limiterFlagParse", t, func() {
		args := []string{
			"route",
			"--wrongArg=1",
		}
		_, err := limiterFlagParse(context.Background(), args)
		convey.So(err, convey.ShouldNotBeNil)

		args = []string{
			"HelloServer.ABC",
			"--allowGuest=true",
			"--suiInfo=true",
			"--copyTradeInfo=123",
			"--bizType=0",
		}
		_, err = limiterFlagParse(context.Background(), args)
		convey.So(err, convey.ShouldNotBeNil)

		args = []string{
			"HelloServer.ABC",
			"--allowGuest=true",
			"--suiInfo=true",
			"--weakAuth=true",
			"--refreshToken=true",
		}
		_, err = limiterFlagParse(context.Background(), args)
		convey.So(err, convey.ShouldNotBeNil)

		args = []string{
			"HelloServer.ABC",
			"--allowGuest=true",
			"--accountIDQuery=future",
			"--memberTags=tag1",
		}
		_, err = limiterFlagParse(context.Background(), args)
		convey.So(err, convey.ShouldBeNil)
	})
}

func Test_oauth(t *testing.T) {
	convey.Convey("test oauth", t, func() {
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		route := metadata.RouteKey{
			AppName:     "future",
			ModuleName:  "module",
			ServiceName: "service",
			Registry:    "reg",
			MethodName:  "method",
			HttpMethod:  "post",
		}
		md.Route = route
		a := &auth{}
		as, _ := user.NewAccountService()
		a.as = as
		a.rules.Store(route.String(), &authRule{oauth: true})
		next := func(*types.Ctx) error {
			return nil
		}

		handler := a.Do(next)
		patch := gomonkey.ApplyFunc((*auth).checkOAuth, func(a *auth, ctx *types.Ctx, rule *authRule, md *metadata.Metadata, token string) (*oauthv1.OAuthResponse, error) {
			if token == "failedtoken" {
				return nil, errors.New("mock err")
			}

			return &oauthv1.OAuthResponse{}, nil
		})
		defer patch.Reset()

		err := handler(ctx)
		convey.So(err, convey.ShouldBeNil)

		ctx.Request.Header.Set("authorization", "failedtoken")
		err = handler(ctx)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func makeAuth(t *testing.T) (*auth, *gomock.Controller, *mock.MockAccountIface, *mock.MockMasqueIface) {
	aa := &auth{}
	ctrl := gomock.NewController(t)
	m := mock.NewMockAccountIface(ctrl)
	aa.as = m
	mm := mock.NewMockMasqueIface(ctrl)
	aa.ms = mm
	return aa, ctrl, m, mm
}

type mockOAuthService1 struct {
	// Define your struct that will mock the OAuth service
}

func (receiver mockOAuthService1) OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error) {
	return &oauthv1.OAuthResponse{Error: &oauthv1.Error{ErrorCode: 0}, MemberId: 1}, nil
}

type mockOAuthService2 struct {
	// Define your struct that will mock the OAuth service
}

func (receiver mockOAuthService2) OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error) {
	return &oauthv1.OAuthResponse{Error: &oauthv1.Error{ErrorCode: 1}}, nil
}

type mockOAuthService3 struct {
	// Define your struct that will mock the OAuth service
}

func (receiver mockOAuthService3) OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error) {
	return nil, nil
}

type mockOAuthService4 struct {
	// Define your struct that will mock the OAuth service
}

func (receiver mockOAuthService4) OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error) {
	return nil, errors.New("error")
}

// Implement the necessary method(s) for your mock OAuth service here
// You can have it return different things based on the inputs to simulate different scenarios
//
// func TestCheckOAuth(t *testing.T) {
//	gmetric.Init("TestCheckOAuth")
//	ctx, md := test.NewReqCtx()
//	rule := &authRule{}
//	token := "token"
//
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	a := &auth{
//		os: &mockOAuthService1{},
//		as: mock.NewMockAccountIface(ctrl),
//	}
//
//	// Test scenario 1: Token is empty and guest access is allowed
//	rule.oauth = true
//	resp, err := a.checkOAuth(ctx, rule, md, token)
//	assert.NoError(t, err)
//	assert.NotNil(t, resp)
//
//	ctx.Request.Header.Set(oauthToken, "a")
//	a.checkLogin(ctx, rule, md, token)
//
//	a = &auth{
//		os: &mockOAuthService2{},
//	}
//
//	// Test scenario 1: Token is empty and guest access is allowed
//	resp, err = a.checkOAuth(ctx, rule, md, token)
//	assert.Error(t, err)
//	assert.Nil(t, resp)
//
//	rule.oauth = true
//	rule.allowGuest = true
//	resp, err = a.checkOAuth(ctx, rule, md, "")
//	assert.NotNil(t, resp)
//
//	rule.oauth = true
//	rule.allowGuest = false
//	resp, err = a.checkOAuth(ctx, rule, md, "")
//	assert.Error(t, err)
//	assert.Nil(t, resp)
//
//	rule.oauth = true
//	rule.allowGuest = false
//	resp2, err := a.checkLogin(ctx, rule, md, "")
//	assert.Error(t, err)
//	assert.Nil(t, resp2)
//
// }
//
// func TestCheckOAuth2(t *testing.T) {
//	ctx, md := test.NewReqCtx()
//	rule := &authRule{}
//	token := "token"
//
//	a := &auth{
//		os: &mockOAuthService1{},
//	}
//
//	// Test scenario 1: Token is empty and guest access is allowed
//	rule.oauth = true
//	resp, err := a.checkOAuth(ctx, rule, md, token)
//	assert.NoError(t, err)
//	assert.NotNil(t, resp)
//
//	a = &auth{
//		os: &mockOAuthService2{},
//	}
//
//	rule.oauth = true
//	rule.allowGuest = false
//	resp2, err := a.checkLogin(ctx, rule, md, "")
//	assert.Error(t, err)
//	assert.Nil(t, resp2)
//
// }
//
// func TestCheckOAuth3(t *testing.T) {
//	ctx, md := test.NewReqCtx()
//	rule := &authRule{}
//	token := "token"
//
//	a := &auth{
//		os: &mockOAuthService3{},
//	}
//
//	rule.oauth = true
//	rule.allowGuest = false
//	resp, err := a.checkLogin(ctx, rule, md, token)
//	assert.Error(t, err)
//	assert.Nil(t, resp)
//
// }
//
// func TestCheckOAuth4(t *testing.T) {
//	ctx, md := test.NewReqCtx()
//	rule := &authRule{}
//	token := "token"
//
//	a := &auth{
//		os: &mockOAuthService4{},
//	}
//
//	rule.oauth = true
//	rule.allowGuest = false
//	resp, err := a.checkLogin(ctx, rule, md, token)
//	assert.Error(t, err)
//	assert.Nil(t, resp)
//
//	md = gmetadata.NewMetadata()
//	md.Route = gmetadata.RouteKey{
//		AppName:     "test",
//		ModuleName:  "ccc",
//		ServiceName: "aaa",
//		Registry:    "ddd",
//		MethodName:  "xxx",
//		HttpMethod:  "get",
//	}
//	ctx.SetUserValue(constant.METADATA_CTX, md)
// }
