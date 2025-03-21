package http

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/tj/assert"

	"bgw/pkg/common/berror"
	"bgw/pkg/test"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/types"
	"bgw/pkg/server/core"
	"bgw/pkg/server/filter"
)

func TestServer_Init(t *testing.T) {
	Convey("test server init", t, func() {
		sr := &Server{
			controller: &errCtrll{},
			console:    &webConsole{},
		}

		err := sr.Init()
		So(err, ShouldNotBeNil)

		sr = &Server{
			controller: &mockController{},
			console:    &webConsole{},
		}

		patch := gomonkey.ApplyFunc((*webConsole).init, func(*webConsole) error { return errors.New("mock err") })
		defer patch.Reset()
		err = sr.Init()
		So(err, ShouldNotBeNil)
	})
}

func TestServer_RequestHandler2(t *testing.T) {
	Convey("test RequestHandler2", t, func() {
		sr := &Server{}
		patch := gomonkey.ApplyFunc((*Server).genericHandler, func(server *Server) types.Handler {
			return func(ctx *fasthttp.RequestCtx) error {
				return errors.New("mock err")
			}
		})
		defer patch.Reset()

		handler := sr.RequestHandler()
		handler(&fasthttp.RequestCtx{})
	})
}

func TestHandle(t *testing.T) {
	gmetric.Init("TestHandle")
	s := &Server{
		controller: &mockController{
			getHandlerErr: errors.New("ssss"),
		},
	}
	rctx, _ := test.NewReqCtx()
	err := s.handle(rctx)
	assert.EqualError(t, err, "ssss")
	assert.Equal(t, "ssss", string(rctx.Response.Header.StatusMessage()))
	assert.Equal(t, http.StatusUnauthorized, rctx.Response.StatusCode())

	s = &Server{
		controller: &mockController{
			getHandlerErr: nil,
			handler:       nil,
		},
	}
	rctx, _ = test.NewReqCtx()
	err = s.handle(rctx)
	assert.Equal(t, berror.ErrRouteNotFound, err)

	s = &Server{
		controller: &mockController{
			getHandlerErr: nil,
			handler: func(rctx *fasthttp.RequestCtx) error {
				return errors.New("cvvcc")
			},
		},
	}
	rctx, _ = test.NewReqCtx()
	err = s.handle(rctx)
	assert.EqualError(t, err, "cvvcc")

	s = &Server{
		controller: &mockController{
			getHandlerErr: nil,
			handler: func(rctx *fasthttp.RequestCtx) error {
				return nil
			},
		},
	}
	rctx, _ = test.NewReqCtx()
	err = s.handle(rctx)
	assert.NoError(t, err)
}

func TestNew(t *testing.T) {
	Convey("test new server", t, func() {
		s := New()
		s.controller = &mockController{
			handler: func(rctx *fasthttp.RequestCtx) error {
				return nil
			},
		}
		h := s.Health()
		So(h, ShouldBeFalse)
		state := s.State()
		So(state, ShouldNotBeNil)
		eps := s.Endpoints()
		So(eps, ShouldBeNil)
	})
}

func TestServer_RequestHandler(t *testing.T) {
	Convey("test RequestHandler", t, func() {
		s := New()
		s.controller = &mockController{
			handler: func(rctx *fasthttp.RequestCtx) error {
				return nil
			},
		}

		Convey("test serviceTimeHandler", func() {
			ctx := &types.Ctx{}
			err := s.serviceTimeHandler(ctx)
			So(err, ShouldBeNil)
			So(len(ctx.Response.Body()), ShouldNotEqual, 0)
		})

		Convey("test route and handler", func() {
			ctx := &types.Ctx{}
			err := s.Route(ctx)
			So(err, ShouldBeNil)

			ctx2 := &types.Ctx{}
			url := &fasthttp.URI{}
			url.SetPath("/v5/market/time")
			ctx2.Request.SetURI(url)

			err = s.Route(ctx2)
			So(err, ShouldBeNil)
		})

		Convey("test genericHandler", func() {
			patch := gomonkey.ApplyFunc(filter.GlobalChain, func() *filter.Chain { return filter.NewChain() })
			defer patch.Reset()

			handler := s.RequestHandler()
			ctx := &types.Ctx{}
			handler(ctx)

			err := s.Start()
			So(err, ShouldBeNil)
			health := s.Health()
			So(health, ShouldBeTrue)

			err = s.Stop()
			So(err, ShouldBeNil)
			health = s.Health()
			So(health, ShouldBeFalse)
		})
	})
}

func Test_errHandler(t *testing.T) {
	Convey("test errHandler", t, func() {
		ctx := &fasthttp.RequestCtx{}
		err := errors.New("mock err")
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()
		errHandler(ctx, err)
	})
}

// func TestServer_Stop(t *testing.T) {
// 	Convey("test server stop", t, func() {
// 		patch := gomonkey.ApplyFunc((*viper.Viper).GetBool, func(*viper.Viper, string) bool { return false })
// 		defer patch.Reset()
//
// 		s := &Server{
// 			server: &fasthttp.Server{},
// 		}
// 		err := s.Stop()
// 		So(err, ShouldBeNil)
// 	})
// }

type mockController struct {
	getHandlerErr error
	handler       types.Handler
}

func (m *mockController) Init() error {
	return nil
}

func (m *mockController) GetRouteManager() core.RouteManager {
	return &mockRoutMgr{}
}

func (m *mockController) GetHandler(ctx context.Context, p core.RouteDataProvider) (types.Handler, error) {
	return m.handler, m.getHandlerErr
}

func (m *mockController) String() string {
	return "mock controller"
}

type errCtrll struct {
}

func (m *errCtrll) Init() error {
	return errors.New("mock err")
}

func (m *errCtrll) GetRouteManager() core.RouteManager {
	return &mockRoutMgr{}
}

func (m *errCtrll) GetHandler(ctx context.Context, p core.RouteDataProvider) (types.Handler, error) {
	return nil, nil
}

func (m *errCtrll) String() string {
	return "mock controller"
}
