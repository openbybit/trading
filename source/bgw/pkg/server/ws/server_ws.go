package ws

import (
	"bytes"
	"context"
	"net/http"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/discov/nacos"
	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	rnacos "bgw/pkg/registry/nacos"
	"bgw/pkg/server/core"
)

const userToken = "usertoken"

var bUserToken = []byte(userToken)

type wsServer struct {
	upgrader   *Upgrader
	srv        *fasthttp.Server
	controller core.Controller
	register   *nacos.Register
}

func (s *wsServer) FillStatus(st *Status) {
	if s.controller != nil {
		st.Core = cast.UnsafeStringToBytes(s.controller.String())
	}
}

func (s *wsServer) Start() error {
	appConf := getAppConf()
	if appConf.EnableController {
		s.controller = core.GetController(context.TODO())
		if err := s.controller.Init(); err != nil {
			glog.Error(context.TODO(), "Init controller error", glog.String("error", err.Error()))
			return err
		}
	}

	wsConf := getStaticConf().WS

	addr := toListenAddress(wsConf.ListenPort)
	glog.Infof(context.TODO(), "[bgws]start websocket, addr=%v", addr)

	// 1.init upgrader
	s.upgrader = &websocket.FastHTTPUpgrader{
		ReadBufferSize:    wsConf.ReadBufferSize,
		WriteBufferSize:   wsConf.WriteBufferSize,
		WriteBufferPool:   &sync.Pool{},
		EnableCompression: wsConf.Compression,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
	}
	// 2.init routes
	server := &fasthttp.Server{
		ReadTimeout:        wsConf.ReadTimeout,
		WriteTimeout:       wsConf.WriteTimeout,
		IdleTimeout:        wsConf.IdleTimeout,
		ReadBufferSize:     wsConf.ReadBufferSize,
		WriteBufferSize:    wsConf.WriteBufferSize,
		MaxRequestBodySize: wsConf.MaxRequestBodySize,
		CloseOnShutdown:    true,
		Handler:            s.RequestHandler(),
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			switch err.(type) {
			case *fasthttp.ErrSmallBuffer:
				WSErrorInc("ws_server", "smallbuffer")
			default:
				WSErrorInc("ws_server", "other")
			}
			glog.Error(ctx, "[bgws] fasthttp serve fail", glog.NamedError("error", err))
		},
	}

	if err := gapp.Serve(server, "tcp", addr, true); err != nil {
		return err
	}

	s.srv = server

	// 服务注册
	r, err := rnacos.BuildRegister(serviceName, wsConf.ListenPort, config.ServiceRegistry{Enable: wsConf.EnableRegistry, ServiceName: wsConf.ServiceName})
	if err != nil {
		glog.Error(context.TODO(), "[BGWS] buildRegister error", glog.NamedError("err", err))
		galert.Error(context.TODO(), "[BGWS] buildRegister error", galert.WithField("err", err))
		return nil
	}
	s.register = r
	if s.register != nil {
		glog.Info(context.Background(), "[BGWS] Service receive start signal, begin Register", glog.String("ip", nets.GetLocalIP()), glog.String("addr", addr))
		if err := s.register.Register(); err != nil {
			glog.Error(context.TODO(), "[BGWS] Register error", glog.NamedError("err", err))
			galert.Error(context.TODO(), "[BGWS] Register error", galert.WithField("err", err))
		}
	}

	return nil
}

func (s *wsServer) Unregister() {
	if s.register == nil {
		return
	}

	glog.Info(context.TODO(), "[BGWS] service receive stop signal, begin stop register", glog.String("ip", nets.GetLocalIP()))
	err := s.register.Stop()
	if err != nil {
		glog.Error(context.TODO(), "[BGWS] register.Stop() error", glog.NamedError("err", err))
	}
}

func (s *wsServer) Stop() {
	if s.srv == nil {
		return
	}

	glog.Info(context.Background(), "start to stop websocket server")
	_ = s.srv.Shutdown()
	s.srv = nil
}

func (s *wsServer) RequestHandler() RequestHandler {
	r := router.New()

	paths := getStaticConf().WS.Routes
	for _, p := range paths {
		path, version := parsePathAndVersion(p)
		glog.Info(context.Background(), "ws server register new path", glog.String("path", p), glog.String("version", version.String()))
		if version != versionNone {
			r.GET(path, s.upgrade(version))
		}
	}

	return r.Handler
}

func (s *wsServer) upgrade(vt versionType) RequestHandler {
	return func(ctx *types.Ctx) {
		if s.srv == nil {
			ctx.Response.Header.SetStatusCode(http.StatusInternalServerError)
			ctx.Response.SetBodyString("server not ready")
			return
		}

		const private = "/private"
		var uid int64
		var err error
		path := string(ctx.Path())
		if path == private && !getDynamicConf().DisableAuthorizedOnConnected {
			var token []byte
			var haveToken bool
			peeker := func(key, value []byte) {
				if bytes.EqualFold(bUserToken, key) {
					haveToken = true
					token = value
				}
			}
			ctx.Request.Header.VisitAll(peeker)
			if !haveToken {
				ctx.QueryArgs().VisitAll(peeker)
			}
			if haveToken {
				WSCounterInc("ws_server", "verify_token")
				if len(token) == 0 {
					WSCounterInc("ws_server", "empty_token")
					ctx.Error("upgrade error, no token", http.StatusUnauthorized)
					return
				}
				originUrl := string(ctx.Referer()) + path
				uid, err = verifyToken(context.Background(), string(token), originUrl)
				if err != nil {
					ctx.Error("upgrade error: "+err.Error(), http.StatusUnauthorized)
					return
				}
			}
		}

		err = s.upgrader.Upgrade(ctx, func(conn *WSConn) {
			s.doUpgrade(ctx, conn, vt, uid)
		})

		if err != nil {
			glog.Info(ctx, "ws_client upgrade fail",
				glog.String("err", err.Error()),
				glog.String("ip", bhttp.GetRemoteIP(ctx)),
				glog.String("path", string(ctx.Path())),
			)
			WSCounterInc("ws_server", "upgrade_fail")
			ctx.Response.Header.SetStatusCode(http.StatusInternalServerError)
			ctx.Response.SetBodyString("upgrade error: " + err.Error())
			return
		}
	}
}

func (s *wsServer) doUpgrade(ctx *types.Ctx, conn *WSConn, vt versionType, uid int64) {
	params := make(map[string]string)
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		if bytes.EqualFold(key, bUserToken) {
			return
		}
		params[string(key)] = string(value)
	})

	if getAppConf().ReplaceStreamPlatform {
		const optionPlatformStream = "1"
		params[urlParamKeyPlatformUnderline] = optionPlatformStream
	}

	ip := bhttp.GetRemoteIP(ctx)
	cli := NewClient(&ClientConfig{
		IP:          ip,
		Path:        string(ctx.Path()),
		Host:        string(ctx.URI().Host()),
		UserAgent:   string(ctx.UserAgent()),
		Referer:     string(ctx.Referer()),
		BrokerID:    string(ctx.Request.Header.Peek(constant.BrokerID)),
		XOriginFrom: string(ctx.Request.Header.Peek(constant.XOriginFrom)),
		Params:      params,
	})
	sess := newSession(conn, cli, vt)

	if err := gSessionMgr.AddSession(sess); err != nil {
		glog.Info(ctx, "add session fail", glog.String("error", err.Error()))
		return
	}

	if uid > 0 {
		if err := bindUser(uid, sess); err != nil {
			WSCounterInc("ws_server", "bind_user_fail")
			return
		}
	}

	args := ctx.QueryArgs()
	platform := string(args.Peek(urlParamKeyPlatformUnderline))
	if len(platform) == 0 {
		platform = string(args.Peek(urlParamKeyPlatform))
	}
	platform = verifyPlatform(platform)

	source := string(args.Peek(urlParamKeySource))
	source = verifySource(source)

	version := string(args.Peek(urlParamKeyVersion))
	if len(version) == 0 {
		version = string(args.Peek(urlParamKeyVersionShort))
	}
	version = verifyVersion(version)

	wsUpgradeInc(string(ctx.Path()), platform, source, version)

	glog.Info(ctx, "new session",
		glog.String("id", sess.ID()),
		glog.String("ip", ip),
		glog.String("protocol", vt.String()),
		glog.String("host", string(ctx.Host())),
		glog.String("path", string(ctx.Path())),
		glog.String("query", ctx.QueryArgs().String()),
		glog.String("platform", platform),
		glog.String("version", version),
		glog.String("source", source),
		glog.Int64("uid", uid),
	)

	sess.Run()
}
