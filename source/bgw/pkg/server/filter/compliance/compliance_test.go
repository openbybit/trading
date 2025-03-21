package compliance

import (
	"context"
	"log"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/geo"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/geoip"
)

func TestNew(t *testing.T) {
	Convey("test new", t, func() {
		Init()
		c := New()
		n := c.GetName()
		So(n, ShouldEqual, filter.ComplianceWallFilterKey)
	})
}

func TestComplianceWall_Init(t *testing.T) {
	Convey("test compliance wall init", t, func() {

		patch0 := gomonkey.ApplyFunc(galert.Error, func(ctx context.Context, message string, opts ...galert.Option) {})
		defer patch0.Reset()

		args := []string{"route", "--scene=SBU_OPENAPI_orderCreate", "--multiScenes={\"futures\":\"FBU_OPENAPI_v5PlaceOrder\",\"option\":\"OBU_OPENAPI_v3BatchPlaceOrder\"}",
			"--kycInfo=true", "--spotLeveragedToken={\"Scene\":\"SBU_OPENAPI_etp_orderCreate\",\"Category\":\"\",\"SymbolField\":\"symbol\"}", "--product=SPOT",
			"--batchItemsKey=request.reduceOnly", "--uaeLeverageCheck=futures", "--uaeSymbolCheck={\"Category\":[\"futures\",\"spot\"],\"SymbolField\":\"symbol\"}",
			"--reduceOnlyKey=reduceOnly", "--batchUaeSymbolCheck={\"Category\":[\"futures\",\"spot\"],\"SymbolField\":\"symbol\"}"}

		c := &complianceWall{}

		patch := gomonkey.ApplyFunc(geoip.NewGeoManager, func() (geo.GeoManager, error) { return nil, mockErr })
		defer patch.Reset()
		patch1 := gomonkey.ApplyFunc(initComplianceService, func() error { return nil })
		defer patch1.Reset()
		patch2 := gomonkey.ApplyFunc(buildListen, func() error { return nil })
		defer patch2.Reset()

		err := c.Init(context.Background(), args...)
		So(err, ShouldBeNil)

		args = append(args, "--wrongFlags=000")
		err = c.Init(context.Background(), args...)
		So(err, ShouldNotBeNil)
	})
}

func TestComplianceWall_Do(t *testing.T) {
	Convey("test compliance wall Do", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {
			log.Println("123")
		})
		defer patch.Reset()
		patch1 := gomonkey.ApplyFunc((*complianceWall).getGeoIP, func(*complianceWall, context.Context, string) (string, string, error) {
			return "", "", mockErr
		})
		defer patch1.Reset()

		patch2 := gomonkey.ApplyFunc(gmetric.ObserveDefaultLatencySince, func(t time.Time, typ, label string) {})
		defer patch2.Reset()

		c := &complianceWall{}
		args := []string{"route", "--scene=SBU_OPENAPI_orderCreate", "--multiScenes={\"futures\":\"FBU_OPENAPI_v5PlaceOrder\",\"option\":\"OBU_OPENAPI_v3BatchPlaceOrder\"}",
			"--kycInfo=true", "--spotLeveragedToken={\"Scene\":\"SBU_OPENAPI_etp_orderCreate\",\"Category\":\"\",\"SymbolField\":\"symbol\"}", "--product=SPOT",
			"--uaeLeverageCheck=futures", "--uaeSymbolCheck={\"Category\":[\"futures\",\"spot\"],\"SymbolField\":\"symbol\"}",
			"--reduceOnlyKey=reduceOnly", "--batchUaeSymbolCheck={\"Category\":[\"futures\",\"spot\"],\"SymbolField\":\"symbol\"}"}
		_ = c.Init(context.Background(), args...)
		next := func(ctx *types.Ctx) error {
			return nil
		}
		h := c.Do(next)

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.IsDemoUID = true
		err := h(ctx)
		So(err, ShouldBeNil)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().CheckStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			int64(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, false, mockErr)
		res := &mockRes{}
		mockWall.EXPECT().CheckStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			int64(2), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, false, nil)
		mockWall.EXPECT().CheckStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			int64(3), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, true, nil)
		mockWall.EXPECT().GetUserInfo(gomock.Any(), gomock.Any()).Return(gcompliance.UserInfo{}, nil).AnyTimes()
		mockWall.EXPECT().GetSiteConfig(gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, mockErr).AnyTimes()

		gw = mockWall

		md.IsDemoUID = false
		md.UID = 1
		err = h(ctx)
		So(err, ShouldBeNil)

		md.UID = 2
		err = h(ctx)
		So(err, ShouldBeNil)

		md.UID = 3
		ctx.Request.SetBody(make([]byte, 0))
		err = h(ctx)
		So(err, ShouldNotBeNil)
	})
}

func TestComplianceWall_getGeoIP(t *testing.T) {
	Convey("test get geo ip", t, func() {
		ctrl := gomock.NewController(t)
		mockGeoMgr := geo.NewMockGeoManager(ctrl)
		mockGeoMgr.EXPECT().QueryCityAndCountry(gomock.Any(), "127.0.0.1").Return(nil, mockErr).AnyTimes()

		c := &complianceWall{}

		patch := gomonkey.ApplyFunc(geoip.NewGeoManager, func() (geo.GeoManager, error) { return mockGeoMgr, nil })
		_, _, err := c.getGeoIP(context.Background(), "127.0.0.1")
		So(err, ShouldEqual, mockErr)
		patch.Reset()

		patch1 := gomonkey.ApplyFunc(geoip.NewGeoManager, func() (geo.GeoManager, error) { return nil, mockErr })
		_, _, err = c.getGeoIP(context.Background(), "127.0.0.1")
		So(err, ShouldNotBeNil)
		patch1.Reset()

	})
}

func TestListener_OnEvent(t *testing.T) {
	Convey("test listener on event", t, func() {
		l := &listener{}

		err := l.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &observer.DefaultEvent{}
		err = l.OnEvent(e)
		So(err, ShouldBeNil)

		patch := gomonkey.ApplyFunc(yaml.Unmarshal, func([]byte, interface{}) error { return nil })
		defer patch.Reset()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().SetCityConfig(gomock.Any(), gomock.Any())
		gw = mockWall

		e = &observer.DefaultEvent{
			Value: "123",
		}
		err = l.OnEvent(e)
		So(err, ShouldBeNil)

		ty := l.GetEventType()
		So(ty, ShouldBeNil)

		p := l.GetPriority()
		So(p, ShouldEqual, 0)
	})
}

func Test_buildListen(t *testing.T) {
	Convey("test build listen", t, func() {
		patch := gomonkey.ApplyFunc((*listener).OnEvent, func(*listener, observer.Event) error { return nil })
		defer patch.Reset()
		_ = buildListen(context.Background())
	})
}
