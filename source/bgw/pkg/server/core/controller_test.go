package core

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/groute"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/discovery"
	_ "bgw/pkg/server/cluster/selector/random"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/tradingroute"
)

type mockRouter struct {
	addr        string
	endpointErr error
}

func (m mockRouter) Namespace() string {
	return "123"
}

func (m mockRouter) IsAioUser(ctx context.Context, userId int64) (bool, error) {
	return true, nil
}

func (m mockRouter) GetEndpoint(ctx context.Context, in *tradingroute.GetRouteRequest) (tradingroute.Endpoint, error) {
	return tradingroute.Endpoint{Address: m.addr}, m.endpointErr
}

func (m mockRouter) ClearInstances() map[string]string {
	return nil
}

func (m mockRouter) ClearRoutings() {
	return
}

func (m mockRouter) ClearRoutingByUser(userId int64, scope string) {
	return
}

func TestGetGroup(t *testing.T) {
	ctr := controller{}
	g := ctr.getGroup(&MethodConfig{
		GroupRouteMode: allToDefault,
		service: &ServiceConfig{
			Group: allToDefault,
		}}, false)
	assert.Equal(t, allToDefault, g)

	g = ctr.getGroup(&MethodConfig{GroupRouteMode: allToDefault,
		service: &ServiceConfig{
			Group: defaultOnly,
		}}, false)
	assert.Equal(t, defaultOnly, g)

	g = ctr.getGroup(&MethodConfig{GroupRouteMode: demoOnly, service: &ServiceConfig{
		Group: demoOnly,
	}}, false)
	assert.Equal(t, demoAccountGroup, g)

	g = ctr.getGroup(&MethodConfig{GroupRouteMode: "123", service: &ServiceConfig{
		Group: "xxx",
	}}, true)
	assert.Equal(t, demoAccountGroup, g)

	g = ctr.getGroup(&MethodConfig{GroupRouteMode: "123", service: &ServiceConfig{
		Group: "xxx",
	}}, false)
	assert.Equal(t, "xxx", g)
}

func TestDemoAccountTradingInvoke(t *testing.T) {

	t.Run("empty address", func(t *testing.T) {
		md := metadata.NewMetadata()
		md.Route = metadata.RouteKey{
			AppName: "test",
		}
		ctr := controller{}
		err := ctr.tradingInvoke(&fasthttp.RequestCtx{},
			mockRouter{
				addr: "",
			},
			nil,
			md)
		if err.Error() != "empty routing address" {
			t.Failed()
		}
	})

	t.Run("endpoint error", func(t *testing.T) {
		md := metadata.NewMetadata()
		md.Route = metadata.RouteKey{
			AppName: "test",
		}
		ctr := controller{}
		err := ctr.tradingInvoke(&fasthttp.RequestCtx{},
			mockRouter{
				addr:        "",
				endpointErr: berror.ErrParams,
			},
			nil,
			md)
		if err != berror.ErrParams {
			t.Failed()
		}
	})
	// todo mock invoker
	t.Run("call", func(t *testing.T) {
		defer func() {
			if msg := recover(); msg != nil {
			}
		}()
		md := metadata.NewMetadata()
		md.Route = metadata.RouteKey{
			AppName: "test",
		}
		ctr := controller{}
		err := ctr.tradingInvoke(&fasthttp.RequestCtx{},
			mockRouter{
				addr: "11111",
			},
			&MethodConfig{
				service: &ServiceConfig{
					App: &AppConfig{App: "123"},
				},
			},
			md)
		if err != berror.ErrParams {
			t.Failed()
		}
	})

}

func TestGetController(t *testing.T) {
	Convey("test GetController", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
		defer patch.Reset()
		c := GetController(context.Background()).(*controller)
		So(c, ShouldNotBeNil)

		r := c.GetRouteManager()
		So(r, ShouldNotBeNil)

		err := c.Init()
		So(err, ShouldBeNil)

		c.routeManager = &mockRouteManger{}
		h, err := c.GetHandler(context.Background(), nil)
		So(h, ShouldNotBeNil)
		So(err, ShouldBeNil)

		err = c.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &configChangeEvent{
			BaseEvent: &observer.BaseEvent{},
		}
		ac := &AppConfig{}
		ac.Services = append(ac.Services, &ServiceConfig{})
		e.Source = ac
		err = c.OnEvent(e)
		So(err, ShouldNotBeNil)

		ty := c.GetEventType()
		So(ty, ShouldNotBeNil)
		p := c.GetPriority()
		So(p, ShouldEqual, -1)

		mc := &MethodConfig{}
		sc := &ServiceConfig{}
		sc.App = &AppConfig{}
		mc.SetService(sc)
		_, err = c.getRouteChain(mc)
		So(err, ShouldNotBeNil)

		_, err = c.getInvoker(mc)
		So(err, ShouldBeNil)
	})
}

type mockRouteManger struct{}

func (m *mockRouteManger) FindRoute(ctx context.Context, provider RouteDataProvider) (*Route, error) {
	return &Route{
		Handler: func(ctx2 *types.Ctx) error { return nil },
	}, nil
}

func (m *mockRouteManger) FindRoutes(method string, path string) *groute.Routes {
	return &groute.Routes{}
}

func (m *mockRouteManger) Load(appConf *AppConfig) error {
	return nil
}

func (m *mockRouteManger) Routes() []*Route {
	return make([]*Route, 0)
}

func TestController_GetIncoker(t *testing.T) {
	Convey("test getinvoker", t, func() {
		ctr := &controller{
			serviceRegistry: discovery.NewServiceRegistry(context.Background()),
		}
		mc := &MethodConfig{
			service: &ServiceConfig{},
		}
		mc.Service().Registry = "routing://"
		handler, err := ctr.getInvoker(mc)
		So(err, ShouldBeNil)
		So(handler, ShouldNotBeNil)

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.IsDemoUID = true

		patch := gomonkey.ApplyFunc((*controller).tradingInvoke, func(c *controller, ctx *types.Ctx, router tradingroute.Routing, mc *MethodConfig, md *metadata.Metadata) error {
			return nil
		})
		defer patch.Reset()
		err = handler(ctx)
		So(err, ShouldBeNil)

		md.IsDemoUID = false
		err = handler(ctx)
		So(err, ShouldBeNil)

		mc.Service().Registry = "reg"
		handler, err = ctr.getInvoker(mc)
		So(err, ShouldBeNil)
		So(handler, ShouldNotBeNil)

		err = handler(ctx)
		So(err, ShouldNotBeNil)

		md.IsDemoUID = true
		err = handler(ctx)
		So(err, ShouldNotBeNil)
	})
}
