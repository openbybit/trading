package server

import (
	"bgw/pkg/diagnosis"
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/core/logx"

	"bgw/pkg/server/cluster"
	_ "bgw/pkg/server/cluster/selector/consistent_hash"
	_ "bgw/pkg/server/cluster/selector/metas_roundrobin"
	_ "bgw/pkg/server/cluster/selector/multi_registry"
	_ "bgw/pkg/server/cluster/selector/raft"
	_ "bgw/pkg/server/cluster/selector/random"
	_ "bgw/pkg/server/cluster/selector/roundrobin"
	_ "bgw/pkg/server/cluster/selector/symbols_roundrobin"
	_ "bgw/pkg/server/cluster/selector/zone_raft"
	_ "bgw/pkg/server/cluster/selector/zone_roundrobin"
	_ "bgw/pkg/server/cluster/selector/zone_vip_raft"
	"bgw/pkg/server/filter/accesslog"
	_ "bgw/pkg/server/filter/accesslog"
	"bgw/pkg/server/filter/antireplay"
	_ "bgw/pkg/server/filter/antireplay"
	"bgw/pkg/server/filter/apilimiter"
	_ "bgw/pkg/server/filter/apilimiter"
	"bgw/pkg/server/filter/auth"
	_ "bgw/pkg/server/filter/auth"
	"bgw/pkg/server/filter/ban"
	_ "bgw/pkg/server/filter/ban"
	"bgw/pkg/server/filter/biz_limiter"
	_ "bgw/pkg/server/filter/biz_limiter"
	"bgw/pkg/server/filter/bsp"
	"bgw/pkg/server/filter/compliance"
	_ "bgw/pkg/server/filter/compliance"
	fcontext "bgw/pkg/server/filter/context"
	"bgw/pkg/server/filter/cors"
	_ "bgw/pkg/server/filter/cors"
	_ "bgw/pkg/server/filter/cryption"
	"bgw/pkg/server/filter/geoip"
	_ "bgw/pkg/server/filter/geoip"
	"bgw/pkg/server/filter/gray"
	_ "bgw/pkg/server/filter/gray"
	"bgw/pkg/server/filter/limiter"
	_ "bgw/pkg/server/filter/limiter"
	"bgw/pkg/server/filter/metrics"
	_ "bgw/pkg/server/filter/metrics"
	"bgw/pkg/server/filter/open_interest"
	_ "bgw/pkg/server/filter/open_interest"
	"bgw/pkg/server/filter/openapi"
	_ "bgw/pkg/server/filter/openapi"
	"bgw/pkg/server/filter/request"
	_ "bgw/pkg/server/filter/request"
	"bgw/pkg/server/filter/response"
	_ "bgw/pkg/server/filter/response"
	"bgw/pkg/server/filter/signature"
	_ "bgw/pkg/server/filter/signature"
	"bgw/pkg/server/filter/trace"
	_ "bgw/pkg/server/filter/trace"

	"bgw/pkg/config"
	"bgw/pkg/server/http"
	"bgw/pkg/server/ws"
	"bgw/pkg/service"
)

func Run(name string, shardIndex int) {
	config.Flags.ShardIndex = shardIndex
	if err := run(name); err != nil {
		glog.Errorf(context.TODO(), "gapp run fail, index=%v err=%v", shardIndex, err)
		_ = glog.Sync()
	}
}

func run(name string) error {
	var svc server
	switch name {
	case "http":
		initAlert()
		initSechub()
		service.InitLogger()
		initTracing()
		gmetric.Init("bgw")
		registerFilters()

		service.InitGrpc()
		logx.DisableStat()
		cluster.InitCloudCfg()
		diagnosis.InitAdmin()
		diagnosis.ConfigPreflight()
		svc = http.New()
		glog.Infof(context.TODO(),
			"run bgw: is_prod=%v,project_env_name=%v, project_name=%v, env_name=%v, app_name=%v, az=%v, azid=%v, token_key=%v",
			env.IsProduction(),
			env.ProjectEnvName(),
			env.ProjectName(),
			env.EnvName(),
			env.AppName(),
			env.AvailableZoneID(),
			config.GetSecureTokenKey(),
		)

		return gapp.Run(
			gapp.WithAddr(config.GetWingAddr()),
			gapp.WithDefaultEndpoints(),
			gapp.WithHealth(toHealthFunc(svc)),
			gapp.WithEndpoints(toStateEndpoint(svc)),
			gapp.WithEndpoints(svc.Endpoints()...),
			gapp.WithLifecycles(toLifecycle(svc)),
		)
	case "ws":
		return ws.Run()
	default:
		return fmt.Errorf("invalid service type,%v", name)
	}
}

func registerFilters() {
	cors.Init()
	fcontext.Init()
	accesslog.Init()
	request.Init()
	signature.Init()
	trace.Init()
	metrics.Init()
	auth.Init()
	openapi.Init()
	response.Init()
	gray.Init()
	geoip.Init()
	open_interest.Init()
	antireplay.Init()
	apilimiter.Init()
	biz_limiter.Init()
	compliance.Init()
	limiter.Init()
	ban.Init()
	bsp.Init()
}

type server interface {
	Start() error
	Stop() error
	Health() bool
	State() interface{}
	Endpoints() []gapp.Endpoint
}

func toLifecycle(svc server) gapp.LifecycleFunc {
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

func toHealthFunc(svc server) gapp.HealthFunc {
	return func() (bool, interface{}) {
		if svc.Health() {
			return true, "health"
		} else {
			return false, "unhealthy"
		}
	}
}

func toStateEndpoint(svc server) gapp.Endpoint {
	return gapp.Endpoint{
		Route: "/state",
		Index: "/state",
		Title: "gateway state",
		Handler: gapp.ToHandlerFunc(func(r *gapp.Request) (interface{}, error) {
			return svc.State(), nil
		}),
	}
}
