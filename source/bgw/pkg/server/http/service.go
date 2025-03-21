package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/registry/nacos"
	"bgw/pkg/server/core"
	"bgw/pkg/server/filter"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	bn "code.bydev.io/frameworks/byone/core/discov/nacos"
	"github.com/valyala/fasthttp"
)

var (
	pathPublicTime = container.NewSet("/v3/public/time", "/v5/market/time")
)

// Server is http service
type Server struct {
	health     int32
	conf       *config.ServerConfig
	console    *webConsole
	controller core.Controller
	server     *fasthttp.Server
	adminMgr   adminMgr
	register   *bn.Register
}

// New create new http service
func New() *Server {
	ctx := context.Background()
	conf := config.GetHTTPServerConfig()
	return &Server{
		conf:       conf,
		console:    newWebConsole(ctx),
		controller: core.GetController(ctx),
	}
}

func (s *Server) Health() bool {
	return atomic.LoadInt32(&s.health) == 1
}

func (s *Server) State() interface{} {
	type state struct {
		Health bool
		Core   json.RawMessage
	}
	st := state{
		Health: atomic.LoadInt32(&s.health) == 1,
	}
	st.Core = cast.UnsafeStringToBytes(s.controller.String())
	return &st
}

func (s *Server) Endpoints() []gapp.Endpoint {
	return nil
}

// Init service engine
func (s *Server) Init() (err error) {
	if err = s.controller.Init(); err != nil {
		return
	}

	if err = s.console.init(); err != nil {
		return
	}

	s.adminMgr.init(s.controller.GetRouteManager())

	glog.Info(context.Background(), "service info", glog.String("namespace", config.GetNamespace()), glog.String("group", config.GetGroup()))

	return
}

// RequestHandler http handlers
func (s *Server) RequestHandler() types.RequestHandler {
	h := s.genericHandler()
	return func(ctx *fasthttp.RequestCtx) {
		if err := h(ctx); err != nil {
			glog.Errorf(ctx, "handler fail, path: %s, err:%v", cast.UnsafeBytesToString(ctx.Path()), err)
		}
	}
}

// genericHandler generic grpc handler
// construct must filters chain, and invoke handler
// handle errors for invoker
func (s *Server) genericHandler() types.Handler {
	chain := filter.GlobalChain()

	f := func(c *types.Ctx) (err error) {
		return s.Route(c)
	}

	return chain.Finally(f)
}

// Route is http route
func (s *Server) Route(ctx *types.Ctx) (err error) {
	if pathPublicTime.Contains(string(ctx.Path())) {
		return s.serviceTimeHandler(ctx)
	}
	return s.handle(ctx)
}

func (s *Server) handle(ctx *types.Ctx) error {
	provider := core.NewCtxRouteDataProvider(ctx, getUserID, getAccountTypeByUID)
	handler, err := s.controller.GetHandler(ctx, provider)
	if err != nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.Response.Header.SetStatusMessage([]byte(err.Error()))

		glog.Info(ctx, "handler error", glog.String("method", string(ctx.Method())), glog.String("path", string(ctx.Path())), glog.String("err", err.Error()))
		return err
	}

	if handler == nil {
		ctx.SetStatusCode(http.StatusNotFound)
		glog.Debug(ctx, "route not found", glog.String("method", string(ctx.Method())), glog.String("path", string(ctx.Path())))
		return berror.ErrRouteNotFound
	}

	if err := handler(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) serviceTimeHandler(ctx *types.Ctx) error {
	type ServerTimeResponse struct {
		Code    int64  `json:"retCode"`
		Message string `json:"retMsg"`
		Result  struct {
			TimeSecond string `json:"timeSecond"`
			TimeNano   string `json:"timeNano"`
		} `json:"result"`
		RetExtInfo json.RawMessage `json:"retExtInfo"` // for extension info to customer
		Time       int64           `json:"time"`
	}

	now := time.Now()
	resp := ServerTimeResponse{
		Message:    "OK",
		RetExtInfo: []byte("{}"),
		Time:       now.UnixMilli(),
	}
	resp.Result.TimeSecond = cast.ToString(now.Unix())
	resp.Result.TimeNano = cast.ToString(now.UnixNano())
	data, err := util.JsonMarshal(resp)
	if err != nil {
		glog.Info(ctx, "server time marshal error", glog.String("error", err.Error()))
		return berror.ErrDefault
	}
	ctx.SetBody(data)
	return nil
}

// Start starts service hook, it will be called after init, it will be called by bat
// it will set service health status to true after 3 seconds
func (s *Server) Start() error {
	if err := s.Init(); err != nil {
		return err
	}
	conf := config.GetHTTPServerConfig()
	server := &fasthttp.Server{
		ReadTimeout:        conf.GetReadTimeout(),
		WriteTimeout:       conf.GetWriteTimeout(),
		IdleTimeout:        conf.GetIdleTimeout(),
		ReadBufferSize:     conf.GetReadBufferSize(),
		WriteBufferSize:    conf.GetWriteBufferSize(),
		MaxRequestBodySize: conf.GetMaxRequestBodySize(),
		CloseOnShutdown:    true,
		Handler:            s.RequestHandler(),
		ErrorHandler:       errHandler,
	}
	if err := gapp.Serve(server, "tcp", conf.GetAddr(), true); err != nil {
		return err
	}
	s.server = server

	atomic.StoreInt32(&s.health, 1)

	r, err := nacos.BuildRegister("bgw", conf.GetPort(), conf.ServiceRegistry)
	if err != nil {
		glog.Error(context.TODO(), "http buildRegister error", glog.NamedError("err", err))
		galert.Error(context.TODO(), "http buildRegister error", galert.WithField("err", err))
		return nil
	}
	s.register = r
	if s.register != nil {
		glog.Info(context.Background(), "Service receive start signal, begin Register", glog.String("ip", nets.GetLocalIP()), glog.String("addr", conf.GetAddr()))
		if err := s.register.Register(); err != nil {
			glog.Error(context.TODO(), "http Register error", glog.NamedError("err", err))
			galert.Error(context.TODO(), "http Register error", galert.WithField("err", err))
		}
	}
	glog.Info(context.Background(), "service health", glog.String("ip", nets.GetLocalIP()), glog.String("addr", conf.GetAddr()))

	return nil
}

// Stop is a service stop hook, it will be called before stop, it will be called by bat
// it will set service health to false, then block 20 seconds
func (s *Server) Stop() error {
	atomic.StoreInt32(&s.health, 0)
	if s.register != nil {
		glog.Info(context.TODO(), "http service receive stop signal, begin stop register", glog.String("ip", nets.GetLocalIP()))
		err := s.register.Stop()
		if err != nil {
			glog.Error(context.TODO(), "http register.Stop() error", glog.NamedError("err", err))
		}
	}
	if config.AppCfg().NoHealthBlock {
		glog.Info(context.TODO(), "http service skip health block")
		return nil
	}
	glog.Info(context.TODO(), "http service will block 20s, wait elb liveliness game over", glog.String("ip", nets.GetLocalIP()))
	time.Sleep(20 * time.Second)
	if s.server != nil {
		_ = s.server.Shutdown()
		s.server = nil
	}
	glog.Info(context.TODO(), "http service wait over", glog.String("ip", nets.GetLocalIP()))
	return nil
}

func errHandler(ctx *fasthttp.RequestCtx, err error) {
	gmetric.IncDefaultError("server", "fasthttp")
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	glog.Info(ctx, "handle error",
		glog.String("path", fmt.Sprintf("%s-%s", ctx.Method(), ctx.Path())),
		glog.String("querystring", cast.UnsafeBytesToString(ctx.URI().QueryString())),
		glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())),
		glog.String("content-type", string(ctx.Request.Header.ContentType())),
		glog.String("err", errStr),
	)
}
