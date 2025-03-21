package apilimiter

import (
	"context"
	"errors"
	"strconv"
	"time"

	ratelimitv1 "code.bydev.io/fbu/future/api/openapigen.git/pkg/bybit/future/open/ratelimit/v1"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/conhash"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"git.bybit.com/svc/mod/pkg/bplatform"

	"bgw/pkg/common"
	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/discovery"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

const (
	apiLimitServiceName = "open-contract-core"
)

func Init() {
	filter.Register(filter.APILimiterKey, New())
}

type apiLimiter struct {
	discovery  discovery.ServiceRegistryModule
	openAPIURL *common.URL
	apiNodes   *conhash.Consistent
}

// New auth filter.
func New() filter.Filter {
	namespace := config.GetNamespace()
	group := constant.DEFAULT_GROUP
	if env.IsProduction() {
		namespace = constant.DEFAULT_NAMESPACE
	}
	url, _ := common.NewURL(apiLimitServiceName,
		common.WithProtocol(constant.NacosProtocol),
		common.WithNamespace(namespace),
		common.WithGroup(group),
	)
	return &apiLimiter{
		openAPIURL: url,
		apiNodes:   conhash.New(),
	}
}

func (a *apiLimiter) GetName() string {
	return filter.APILimiterKey
}

func (a *apiLimiter) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		md := metadata.MDFromContext(c)
		route := md.GetRoute()
		if !route.Valid() {
			glog.Error(c, "user route error", glog.Any("route", route))
			return berror.NewInterErr("apiLimiter user route error")
		}

		glog.Debug(c, "api_limit check", glog.Int64("uid", md.UID), glog.String("path", md.Path))
		resp, err := a.limitCheck(c, md.UID, md.Path)
		if resp != nil && bplatform.Client(md.GetPlatForm()) == bplatform.OpenAPI {
			a.setLimitHeader(c, resp)
			glog.Debug(c, "api_limit check result", glog.Int64("uid", md.UID),
				glog.String("path", md.Path),
				glog.Int64("quota", resp.GetQuota()),
				glog.Int64("remaining", resp.GetRemaining()),
				glog.Int64("reset", resp.GetResetTimeStamp()),
				glog.Bool("allowed", !resp.GetExceeded()))
		}
		if err != nil {
			return err
		}
		return next(c)
	}
}

// Init implement filter.Initializer
// no need args
func (a *apiLimiter) Init(ctx context.Context, args ...string) error {
	a.discovery = discovery.NewServiceRegistry(ctx)
	return a.discovery.Watch(ctx, a.openAPIURL)
}

func (a *apiLimiter) limitCheck(ctx *types.Ctx, uid int64, path string) (*ratelimitv1.QueryRateLimitResponse, error) {
	conn, err := a.getConn(ctx)
	if err != nil {
		glog.Error(ctx, "api_limit error", glog.String("err", err.Error()))
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	req := &ratelimitv1.QueryRateLimitRequest{
		UserId: uid,
		Path:   path,
	}
	resp, err := ratelimitv1.NewContractRateLimitAPIServiceClient(conn.Client()).QueryRateLimit(ctx, req)
	if err != nil {
		glog.Error(ctx, "api_limit call error", glog.String("err", err.Error()))
		gmetric.IncDefaultError("api_limit_filter", "call")
		return nil, nil // if openapi internal error, not block biz request
	}

	if resp.GetExceeded() {
		glog.Debug(ctx, "api_limiter blocked", glog.Int64("uid", uid), glog.String("path", path))
		gmetric.IncDefaultError("api_limit_filter", "exceed")
		return resp, berror.ErrVisitsLimit
	}
	return resp, nil
}

func (a *apiLimiter) getConn(ctx context.Context) (pool.Conn, error) {
	instances := a.discovery.GetInstances(a.openAPIURL)
	if len(instances) == 0 {
		return nil, errors.New("no instances")
	}

	if len(instances) == 1 {
		addr := instances[0].GetHost() + ":" + strconv.Itoa(instances[0].GetPort())
		return pool.GetConn(ctx, addr)
	}

	nodes := make([]string, 0, len(instances))
	for _, i := range instances {
		nodes = append(nodes, i.GetHost()+":"+strconv.Itoa(i.GetPort()))
		glog.Debug(ctx, "api_limit instance parse", glog.String("host", i.GetHost()), glog.Int64("port", int64(i.GetPort())), glog.String("convertAddr", i.GetHost()+":"+strconv.Itoa(i.GetPort())))
	}

	if a.apiNodes.Diff(nodes) {
		a.apiNodes.Set(nodes)
	}
	md := metadata.MDFromContext(ctx)
	node, err := a.apiNodes.Get(strconv.FormatInt(md.UID, 10))
	if err != nil {
		return nil, err
	}
	glog.Debug(ctx, "api_limit instances", glog.Int64("counts", int64(len(instances))), glog.String("addr", node), glog.Int64("uid", md.UID))
	return pool.GetConn(ctx, node)
}

func (a *apiLimiter) setLimitHeader(ctx *types.Ctx, resp *ratelimitv1.QueryRateLimitResponse) {
	quota := int(resp.GetQuota())
	remaining := int(resp.GetRemaining())
	rateLimit := metadata.RateLimitInfo{
		RateLimitStatus: remaining,
		RateLimit:       quota,
	}
	ctx.Response.Header.Set(constant.HeaderAPILimit, cast.Itoa(quota))
	ctx.Response.Header.Set(constant.HeaderAPILimitStatus, cast.Itoa(remaining))
	var restTime int
	if resp.GetExceeded() { // block
		restTime = int(resp.GetResetTimeStamp())
	} else {
		restTime = int(time.Now().UnixNano() / 1e6)
	}
	rateLimit.RateLimitResetMs = restTime
	ctx.Response.Header.Set(constant.HeaderAPILimitResetTimestamp, cast.Itoa(restTime))
	metadata.ContextWithRateLimitInfo(ctx, rateLimit)
}
