package limiter

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"golang.org/x/time/rate"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center/nacos"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/filter/limiter/manual_intervent"
	"bgw/pkg/server/metadata"
)

var (
	_ filter.Filter      = &globalLimiter{}
	_ filter.Initializer = &globalLimiter{}

	defaultGlobalRate = 20000
	fileName          = "bgw_limit_rate.yaml"
)

type globalLimiter struct {
	cluster          string
	limiter          atomic.Value
	rateValue        atomic.Value
	interveneLimiter *manual_intervent.InterveneLimiter
}

func newGlobalLimiter() filter.Filter {
	if value := config.AppCfg().QpsRate; value > 0 {
		defaultGlobalRate = value
	}

	gl := &globalLimiter{
		cluster:          config.GetHTTPServerConfig().ServiceRegistry.ServiceName,
		interveneLimiter: &manual_intervent.InterveneLimiter{},
	}
	// 初始化时读取静态配置
	l := rate.NewLimiter(rate.Limit(defaultGlobalRate), defaultGlobalRate)
	gl.limiter.Store(l)
	v := newRateLimit()
	gl.rateValue.Store(v)
	return gl
}

// GetName returns the name of the filter
func (l *globalLimiter) GetName() string {
	return filter.QPSRateLimitFilterKeyGlobal
}

// Init implements the filter.Initializer interface
func (l *globalLimiter) Init(ctx context.Context, _ ...string) error {
	nc, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	)

	if err != nil {
		gmetric.IncDefaultError("global_limiter", "nacos_client")
		glog.Error(ctx, "[global limiter]new nacos configure failed", glog.String("err", err.Error()))
		return err
	}

	if err := nc.Listen(ctx, fileName, l); err != nil {
		gmetric.IncDefaultError("global_limiter", "nacos_listen")
		glog.Error(ctx, "[global limiter]listen nacos configure failed", glog.String("err", err.Error()))
		return err
	}

	l.interveneLimiter.Init(ctx)
	return nil
}

// Do limit handle
func (l *globalLimiter) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {

		// 手工干预优先级最高,不计算在limiter中(如果接口被攻击,手工干预后,不影响limiter限流计数).
		if l.interveneLimiter.Intervene(ctx) {
			glog.Debug(ctx, "intervene hit")
			gmetric.IncDefaultError("intervene", "")
			ctx.SetStatusCode(http.StatusTooManyRequests)
			return berror.ErrVisitsLimit
		}

		if !l.getLimiter().Allow() {
			glog.Debug(ctx, "global_limit hit")
			gmetric.IncDefaultError("global_limit", "")
			ctx.SetStatusCode(http.StatusTooManyRequests)
			return berror.ErrVisitsLimit
		}

		if rule := l.getRule(ctx); rule != nil {
			if !rule.Allow() {
				glog.Debug(ctx, "call_origin_limit hit")
				gmetric.IncDefaultError("call_origin_limit", "")
				ctx.SetStatusCode(http.StatusTooManyRequests)
				return berror.ErrVisitsLimit
			}
		}

		return next(ctx)
	}
}

func (l *globalLimiter) getRule(ctx *types.Ctx) *rate.Limiter {
	cfg := l.loadLimitConfig()
	if cfg == nil {
		return nil
	}

	md := metadata.MDFromContext(ctx)
	if rule, ok := cfg.headerLimit[md.Intermediate.CallOrigin]; ok {
		return rule
	}

	return nil
}

func (l *globalLimiter) getLimiter() *rate.Limiter {
	// 一定会有有效的limiter
	rl, _ := l.limiter.Load().(*rate.Limiter)
	return rl
}

func (l *globalLimiter) loadLimitConfig() *rateLimitConfig {
	v, ok := l.rateValue.Load().(*rateLimitConfig)
	if ok && v != nil {
		return v
	}

	return nil
}

type rateLimitConfig struct {
	headerLimit map[string]*rate.Limiter
}

func newRateLimit() *rateLimitConfig {
	return &rateLimitConfig{
		headerLimit: make(map[string]*rate.Limiter),
	}
}

type limitData struct {
	QpsLimits map[string]string `yaml:"qps_limits"`
	RateLimit headersData       `yaml:"rate_limit"`
}

type headersData struct {
	Headers map[string]int `yaml:"headers"`
}

// OnEvent common rate limit config event handle
func (l *globalLimiter) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok || e.Value == "" {
		return nil
	}

	var data limitData
	if err := util.YamlUnmarshalString(e.Value, &data); err != nil {
		gmetric.IncDefaultError("global_limiter", "unmarshal_err")
		glog.Error(context.TODO(), "rete limit config update error", glog.String("key", e.Key), glog.String("error", err.Error()))
		return nil
	}

	limiter := newRateLimit()
	for header, limit := range data.RateLimit.Headers {
		if limit <= 0 || header == "" {
			continue
		}
		limiter.headerLimit[header] = rate.NewLimiter(rate.Limit(limit), limit)
	}

	l.rateValue.Store(limiter)
	glog.Info(context.TODO(), "rete limit config update success", glog.String("key", e.Key))
	str, ok := data.QpsLimits[l.cluster]
	qpsLimit := cast.Atoi(str)
	if ok && qpsLimit > 0 {
		rl := rate.NewLimiter(rate.Limit(qpsLimit), qpsLimit)
		l.limiter.Store(rl)
		label := fmt.Sprintf("%s:%d", l.cluster, qpsLimit)
		gmetric.IncDefaultError("global_limiter_update", label)
		glog.Info(context.Background(), fmt.Sprintf("%s update global limit: %d", l.cluster, qpsLimit))
	}

	if ok && qpsLimit <= 0 {
		gmetric.IncDefaultError("global_limiter", "limit_invalid")
		glog.Info(context.Background(), fmt.Sprintf("%s global limit invalid:%d", l.cluster, qpsLimit))
	}

	return nil
}

// GetEventType get event type
func (l *globalLimiter) GetEventType() reflect.Type {
	return nil
}

// GetPriority get priority
func (l *globalLimiter) GetPriority() int {
	return 0
}
