package http

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"

	"bgw/pkg/common/constant"
	"bgw/pkg/service/tradingroute"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/groute"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/server/core"
)

type mockRoutMgr struct{}

func (m *mockRoutMgr) FindRoute(ctx context.Context, provider core.RouteDataProvider) (*core.Route, error) {
	return nil, nil
}

func (m *mockRoutMgr) FindRoutes(method string, path string) *groute.Routes {
	return &groute.Routes{}
}

func (m *mockRoutMgr) Load(appConf *core.AppConfig) error {
	return nil
}

func (m *mockRoutMgr) Routes() []*groute.Route {
	r1 := &core.Route{
		AppKey: "uta",
		Method: "GET",
		Path:   "\"order/create\"",
	}
	r2 := &core.Route{
		AppKey: "openapi",
		Method: "POST",
		Path:   "\"order/create\"",
	}
	return []*core.Route{r1, r2}
}

func TestOnClearTradingRoute(t *testing.T) {
	gmetric.Init("TestOnClearTradingRoute")
	am := &adminMgr{}
	tr := tradingroute.GetRouting()
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(tr), "ClearRoutingByUser", func(userId int64, scope string) {}).
		ApplyPrivateMethod(reflect.TypeOf(tr), "ClearRoutings", func() {}).
		ApplyPrivateMethod(reflect.TypeOf(tr), "ClearInstances", func() map[string]string {
			return map[string]string{"sss": "xxx"}
		})
	defer p.Reset()
	r, err := am.onClearTradingRoute(gapp.AdminArgs{
		Options: map[string]string{"mode": "xxx"},
	})
	assert.Nil(t, r)
	assert.EqualError(t, err, "not support: xxx")

	r, err = am.onClearTradingRoute(gapp.AdminArgs{
		Options: map[string]string{"mode": "all"},
	})
	assert.NotNil(t, r)
	assert.Equal(t, 1, len(r.(map[string]string)))
	assert.Equal(t, "xxx", r.(map[string]string)["sss"])
	assert.NoError(t, err)

	r, err = am.onClearTradingRoute(gapp.AdminArgs{
		Options: map[string]string{"mode": "instances"},
	})
	assert.NotNil(t, r)
	assert.Equal(t, 1, len(r.(map[string]string)))
	assert.Equal(t, "xxx", r.(map[string]string)["sss"])
	assert.NoError(t, err)

	r, err = am.onClearTradingRoute(gapp.AdminArgs{
		Options: map[string]string{"mode": "routings"},
	})
	assert.Nil(t, r)
	assert.NoError(t, err)

	r, err = am.onClearTradingRoute(gapp.AdminArgs{
		Options: map[string]string{"mode": "user_routing"},
	})
	assert.Nil(t, r)
	assert.NoError(t, err)
}

func TestAdminMgr(t *testing.T) {
	Convey("test admin mgr", t, func() {

		am := &adminMgr{}
		_, err := am.onPing(gapp.AdminArgs{})
		So(err, ShouldBeNil)

		_, err = am.onGetRoute(gapp.AdminArgs{})
		So(err, ShouldNotBeNil)
		am.init(&mockRoutMgr{})
		_, err = am.onGetRoute(gapp.AdminArgs{})
		So(err, ShouldBeNil)

		args := gapp.AdminArgs{
			Options: map[string]string{"path": "order/create"},
		}
		_, err = am.onGetRoute(args)
		So(err, ShouldBeNil)

		args = gapp.AdminArgs{
			Options: map[string]string{"path": "order/create", "method": "GET"},
		}
		_, err = am.onGetRoute(args)
		So(err, ShouldBeNil)

		args = gapp.AdminArgs{
			Options: map[string]string{"path": "order/create", "method": "GET", "app": "openapi"},
		}
		_, err = am.onGetRoute(args)
		So(err, ShouldBeNil)

		_, err = am.onGetTradingRoute(gapp.AdminArgs{})
		So(err, ShouldNotBeNil)

		_, err = am.onClearTradingRoute(gapp.AdminArgs{})
		So(err, ShouldNotBeNil)

		_, err = am.onGetAccountType(gapp.AdminArgs{})
		So(err, ShouldNotBeNil)

		patch := gomonkey.ApplyFunc(getAccountTypeByUID, func(ctx context.Context, uid int64) (constant.AccountType, error) {
			if uid < 10 {
				return constant.AccountTypeUnknown, errors.New("mock err")
			}
			return constant.AccountTypeNormal, nil
		})
		defer patch.Reset()

		_, err = am.onGetAccountType(gapp.AdminArgs{Options: map[string]string{"uid": "9"}})
		So(err, ShouldNotBeNil)

		_, err = am.onGetAccountType(gapp.AdminArgs{Options: map[string]string{"uid": "19"}})
		So(err, ShouldBeNil)
	})
}
