package openapi

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"

	geo2 "code.bydev.io/fbu/gateway/gway.git/geo"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/service/geoip"
	"bgw/pkg/test"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"

	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/tj/assert"
)

func TestCheckPermission(t *testing.T) {
	a := assert.New(t)

	permission := "[[\"OptionsTrade\",false]]"

	o := &openapi{}
	err := o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_WRITE"})
	a.NoError(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_READ"})
	a.NoError(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_READ_WRITE"})
	a.NoError(err)

	permission = "[[\"OptionsTrade\",true]]"
	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_WRITE"})
	a.Error(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_READ"})
	a.NoError(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Group: "RESOURCE_GROUP_OPTIONS_TRADE", Permission: "PERMISSION_READ_WRITE"})
	a.NoError(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{Groups: []string{"RESOURCE_GROUP_OPTIONS_TRADE"}, Permission: "PERMISSION_READ_WRITE"})
	a.NoError(err)

	err = o.checkPermission(&types.Ctx{}, permission, metadata.ACL{AllGroup: true, Permission: "PERMISSION_READ_WRITE"})
	a.NoError(err)
}

func TestIPCheck(t *testing.T) {
	a := assert.New(t)
	o := &openapi{}

	member := &user.MemberLogin{
		Id:          1,
		MemberId:    43567,
		LoginName:   "5FdeE4CnNztmXLC9HE",
		Type:        4,
		LoginSecret: "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh",
		Status:      2,
		ExtInfo: &user.MemberLoginExt{
			ExpiredTimeE0: 1736029405,
			ApiKeyType:    1,
			Ips:           "[\"*\"]",
			Permissions:   "[[\"Position\",false],[\"BlockTrade\",false],[\"SpotTrade\",false]]",
			Note:          "aa",
		},
	}
	ctx, md := test.NewReqCtx()
	err := o.checkIp(ctx, md, true, member, "14.67.45.78")
	a.NoError(err)

	member.ExtInfo.Ips = "[\"14.67.45.78\"]"
	err = o.checkIp(ctx, md, true, member, "14.67.45.78")
	a.NoError(err)

	err = o.checkIp(ctx, md, true, member, "14.67.45.74")
	a.NoError(err)

	err = o.checkIp(ctx, md, false, member, "14.67.45.74")
	a.Error(err)

	member.ExtInfo.ApiKeyType = 2
	err = o.checkIp(&types.Ctx{}, md, false, member, "14.67.45.74")
	a.Equal(berror.ErrInvalidIP, berror.ErrInvalidIP)

	o.nacosLoader = mockNacosLoader{ip: "*", f: false}
	err = o.checkIp(&types.Ctx{}, md, false, member, "14.67.45.74")
	a.Equal(berror.ErrInvalidIP, berror.ErrInvalidIP)

	o.nacosLoader = mockNacosLoader{ip: "*", f: true}
	err = o.checkIp(&types.Ctx{}, md, false, member, "14.67.45.74")
	a.NoError(err)
}

func TestCheckBannedCountries(t *testing.T) {
	gmetric.Init("TestCheckBannedCountries")
	// query member tag
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	defer ctrl.Finish()

	geo := geo2.NewMockGeoManager(ctrl)

	err := aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: true,
	}, "123")
	assert.NoError(t, err)

	p := gomonkey.ApplyFuncReturn(geoip.CheckIPWhitelist, true)
	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.NoError(t, err)
	p.Reset()
	p.ApplyFuncReturn(geoip.CheckIPWhitelist, false)
	p.ApplyFuncReturn(geoip.NewGeoManager, geo, errors.New("xxxx"))
	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.EqualError(t, err, "openapi NewGeoManager error: xxxx")

	p.Reset()

	p.ApplyFuncReturn(geoip.CheckIPWhitelist, false)
	p.ApplyFuncReturn(geoip.NewGeoManager, geo, nil)
	geo.EXPECT().QueryCityAndCountry(gomock.Any(), gomock.Eq("123")).Return(nil, errors.New("xxx"))
	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.NoError(t, err)

	p.Reset()

	p.ApplyFuncReturn(geoip.CheckIPWhitelist, false)
	p.ApplyFuncReturn(geoip.NewGeoManager, geo, nil)
	mgd := &mockGeoData{c: geo2.NewMockCountry(ctrl)}
	mgd.c.EXPECT().GetISO().Return("")
	geo.EXPECT().QueryCityAndCountry(gomock.Any(), gomock.Eq("123")).Return(mgd, nil)

	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.NoError(t, err)

	p.Reset()

	p.ApplyFuncReturn(geoip.CheckIPWhitelist, false)
	p.ApplyFuncReturn(geoip.NewGeoManager, geo, nil)
	mgd = &mockGeoData{c: geo2.NewMockCountry(ctrl)}
	mgd.c.EXPECT().GetISO().Return("88gt02")
	geo.EXPECT().QueryCityAndCountry(gomock.Any(), gomock.Eq("123")).Return(mgd, nil)
	aa.bc = "88gt02"
	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.EqualError(t, err, berror.ErrCountryBanned.Error())

	p.Reset()

	p.ApplyFuncReturn(geoip.CheckIPWhitelist, false)
	p.ApplyFuncReturn(geoip.NewGeoManager, geo, nil)
	mgd = &mockGeoData{c: geo2.NewMockCountry(ctrl)}
	mgd.c.EXPECT().GetISO().Return("882212gt02")
	geo.EXPECT().QueryCityAndCountry(gomock.Any(), gomock.Eq("123")).Return(mgd, nil)
	aa.bc = "88gt02"
	err = aa.checkBannedCountries(rctx, &openapiRule{
		skipIpCheck: false,
	}, "123")
	assert.NoError(t, err)

	p.Reset()
	ctrl.Finish()
}

func TestAntiReplay(t *testing.T) {
	gmetric.Init("TestAntiReplay")
	// query member tag
	rctx, _ := test.NewReqCtx()
	aa, ctrl, _ := makeOpenApi(t)
	defer ctrl.Finish()

	a, err := aa.antiReplay(rctx, "22x", "123")
	assert.Equal(t, int64(0), a)
	assert.EqualError(t, err, "openapi params error! req_timestamp invalid")

	a, err = aa.antiReplay(rctx, "100000", "123xx")
	assert.Equal(t, int64(0), a)
	assert.EqualError(t, err, "openapi params error! recv_window invalid")

	a, err = aa.antiReplay(rctx, "100000", "123")
	assert.Equal(t, int64(0), a)
	assert.Error(t, err)

	n := time.Now()

	rt := n.UnixMilli()

	a, err = aa.antiReplay(rctx, strconv.FormatInt(rt, 10), "100000")
	assert.Equal(t, (rt+100000)*1e6, a)
	assert.NoError(t, err)
}

func TestOpenapi_getCheckers(t *testing.T) {
	convey.Convey("test get checkers", t, func() {
		op := &openapi{}
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		_, err := op.getCheckers(ctx, md, false, false)
		convey.So(err, convey.ShouldNotBeNil)

		md.WssFlag = true
		_, err = op.getCheckers(ctx, md, false, false)
		convey.So(err, convey.ShouldNotBeNil)

		ctx.Request.Header.Set(constant.HeaderAPIKey, "mock_key")
		_, err = op.getCheckers(ctx, md, false, false)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestOpenapi_getCheckers2(t *testing.T) {
	convey.Convey("test getCheckers 2", t, func() {
		op := &openapi{}
		ctx := &types.Ctx{}
		ctx.Request.Header.Set(constant.HeaderAPIKey, "mock_key")

		patch := gomonkey.ApplyFunc(newV3Checker, func(ctx *types.Ctx, apiKey, ip string, allowGuest, isWss bool) (ret [2]Checker, err error) {
			return [2]Checker{
				&v3Checker{apiKey: apiKey, apiSignature: "apiSignature", remoteIp: "ip"},
				&v3Checker{apiKey: apiKey, apiSignature: "apiSignature", remoteIp: "ip"},
			}, nil
		})
		defer patch.Reset()

		// allowGuest
		_, err := op.getCheckers(ctx, &metadata.Metadata{}, true, true)
		convey.So(err, convey.ShouldBeNil)

		patch1 := gomonkey.ApplyFunc((*openapi).antiReplay, func(openapi2 *openapi, ctx *types.Ctx, apiTimestamp, window string) (int64, error) {
			return 1234, nil
		})
		defer patch1.Reset()

		_, err = op.getCheckers(ctx, &metadata.Metadata{}, false, true)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestOpenapi_checkPermission(t *testing.T) {
	convey.Convey("test checkPermission", t, func() {
		op := &openapi{}
		ctx := &types.Ctx{}

		err := op.checkPermission(ctx, "permissions", metadata.ACL{Permission: constant.ResourceGroupInvalid})
		convey.So(err, convey.ShouldNotBeNil)

		err = op.checkPermission(ctx, "OptionsTrade", metadata.ACL{Permission: "RESOURCE_GROUP_OPTIONS_TRADE"})
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestOpenapi_checkDBPermission(t *testing.T) {
	convey.Convey("test checkDbPermission", t, func() {
		op := &openapi{}
		err := op.checkDbPermission(context.Background(), metadata.ACL{}, [][]interface{}{{"123"}}, 1)
		convey.So(err, convey.ShouldNotBeNil)

		err = op.checkDbPermission(context.Background(), metadata.ACL{}, [][]interface{}{{123, "234"}}, 1)
		convey.So(err, convey.ShouldNotBeNil)

		err = op.checkDbPermission(context.Background(), metadata.ACL{}, [][]interface{}{{123, "234"}}, 2)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestOpenapi_checkReadOnly(t *testing.T) {
	convey.Convey("test checkReadOnly", t, func() {
		op := &openapi{}
		err := op.checkReadOnly(context.Background(), []interface{}{true, "123"}, "per")
		convey.So(err, convey.ShouldNotBeNil)
	})
}

type mockGeoData struct {
	c *geo2.MockCountry
}

func (m *mockGeoData) HasCountryInfo() bool {
	return true
}

func (m *mockGeoData) HasCityInfo() bool {
	return true
}

func (m *mockGeoData) GetCountry() geo2.Country {
	return m.c
}

func (m *mockGeoData) GetCity() geo2.City {
	return nil
}

type mockNacosLoader struct {
	f  bool
	ip string
}

func (m mockNacosLoader) GetIpWhiteList(_ context.Context, _ *user.MemberLoginExt) (string, bool) {
	return m.ip, m.f
}
