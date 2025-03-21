package ws

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/core/logx"
	jaeger "github.com/uber/jaeger-client-go/config"
)

func Run() error {
	logx.DisableStat()
	getConfigMgr().LoadStaticConfig()

	conf := getStaticConf()

	// setup metrics
	gmetric.Init(serviceName)
	// setup jaeger
	_, _, _ = jaeger.JaegerInit()

	// setup logger
	lconf := conf.Log.convert(bgwsLogName)
	glog.SetLogger(glog.New(&lconf))

	// setup alert
	fields := []*galert.Field{
		galert.BasicField("env", fmt.Sprintf("%s:%s", env.EnvName(), env.ProjectEnvName())),
		galert.CurrentTimeField("utc", ""),
		galert.BasicField("ip", nets.GetLocalIP()),
	}

	x := galert.New(&galert.Config{
		Webhook: conf.Alert.Path,
		Fields:  fields,
	})
	galert.SetDefault(x)

	// load dynamic config
	getConfigMgr().LoadDynamicConfig()

	glog.Infof(context.Background(), "init config, static=%v", toJsonString(conf))
	glog.Infof(context.Background(), "init config, dynamic=%v", toJsonString(getDynamicConf()))
	glog.Infof(context.Background(), "init config, sdk=%v", toJsonString(getConfigMgr().GetSdkConf()))

	svc := New()

	return gapp.Run(
		gapp.WithAddr(toListenAddress(getAppConf().DevPort)),
		gapp.WithDefaultEndpoints(),
		gapp.WithHealth(toHealthFunc(svc)),
		gapp.WithEndpoints(toStateEndpoint(svc)),
		gapp.WithLifecycles(toLifecycle(svc)),
	)
}

func toLifecycle(svc *Server) gapp.LifecycleFunc {
	return func(ctx context.Context, event gapp.LifecycleEvent) error {
		switch event {
		case gapp.LifecycleStart:
			return svc.Start()
		case gapp.LifecycleStop:
			return svc.Stop()
		default:
			return nil
		}
	}
}

func toHealthFunc(svc *Server) gapp.HealthFunc {
	return func() (bool, interface{}) {
		if svc.Health() {
			return true, "health"
		} else {
			return false, "unhealthy"
		}
	}
}

func toStateEndpoint(svc *Server) gapp.Endpoint {
	return gapp.Endpoint{
		Route: "/state",
		Index: "/state",
		Title: "gateway state",
		Handler: gapp.ToHandlerFunc(func(r *gapp.Request) (interface{}, error) {
			return svc.State(), nil
		}),
	}
}
