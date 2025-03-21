package biz_limiter

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"code.bydev.io/frameworks/byone/kafka"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/openapiv3/settings"
	"github.com/coocood/freecache"

	"bgw/pkg/common/kafkaconsume"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

const (
	cacheLongExpireSeconds = 240 * 3600 // 10 day
	mixerCacheSize         = 100
	mixerOpenapi           = "mixer-openapi"
)

const (
	BTCUSD   = "BTCUSD"
	v3Symbol = "v3"
)

var (
	rateLimitMgr RateLimit
	rateOnce     sync.Once
)

var (
	rateLimitOnce sync.Once
	quotaMgr      *quotaManager
)

type RateLimit interface {
	GetQuota(ctx context.Context, uid int64, app, group, symbol string) (int64, error)
}

type quotaManager struct {
	ctx        context.Context
	quotaCache *freecache.Cache
	client     ggrpc.ClientConnInterface
	kafkaOnce  sync.Once
}

func NewQuotaManager(ctx context.Context) (RateLimit, error) {
	var err error
	rateLimitOnce.Do(func() {
		doNewQuotaManager(ctx)
	})
	if quotaMgr == nil {
		gmetric.IncDefaultError("mixer_quota", "empty_mixer_service")
		glog.Error(ctx, "empty mixer quota manager")
		return nil, fmt.Errorf("empty mixer quota manager: %w", err)
	}

	return quotaMgr, err
}

func doNewQuotaManager(ctx context.Context) {
	rpcClient, err := zrpc.NewClient(config.Global.Mixer, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), "dial mixer-openapi failed,error=%v", err)
		galert.Error(context.Background(), "dial mixer-openapi failed,error=%v", galert.WithField("error", err))
		return
	}
	quotaMgr = &quotaManager{
		ctx:        ctx,
		client:     rpcClient.Conn(),
		quotaCache: freecache.NewCache(mixerCacheSize * 1024 * 1024),
	}
	_ = diagnosis.Register(&quotaDiagnose{
		svc:  quotaMgr,
		cfg:  config.Global.Mixer,
		kCfg: config.Global.KafkaCli,
	})
}

func (q *quotaManager) GetQuota(ctx context.Context, uid int64, app, group, symbol string) (int64, error) {
	q.kafkaOnce.Do(func() {
		kafkaconsume.AsyncHandleKafkaMessage(ctx, constant.EventRateLimitChange, config.Global.KafkaCli,
			q.HandleQuotaMessage, lqOnErr)
	})

	if uid <= 0 || group == "" || symbol == "" {
		return 0, berror.NewBizErr(10001, "params error")
	}

	key := q.getKey(uid, app, group, symbol)
	v, err := q.quotaCache.Get([]byte(key))
	if err == nil {
		quota := cast.AtoInt64(string(v))
		glog.Debug(ctx, "member rate limit cache hit", glog.String("key", key))
		return quota, nil
	}

	glog.Debug(ctx, "member rate limit cache not hit", glog.String("key", key))

	rate, needCache := q.queryQuota(ctx, uid, group, symbol)
	if !needCache {
		return rate, nil
	}

	q.saveQuota(ctx, key, rate)

	return rate, nil
}

func (q *quotaManager) saveQuota(ctx context.Context, key string, rate int64) {
	expire := cacheLongExpireSeconds
	if err := q.quotaCache.Set([]byte(key), []byte(cast.Int64toa(rate)), expire); err != nil {
		glog.Info(ctx, "member rate limit cache set error", glog.String("key", key), glog.String("error", err.Error()))
	}
}

func (q *quotaManager) queryQuota(ctx context.Context, uid int64, group, symbol string) (int64, bool) {
	now := time.Now()
	defer func() {
		gmetric.ObserveDefaultLatencySince(now, "quota", mixerOpenapi)
	}()

	req := &settings.QueryRateLimitReq{
		Uid:    uid,
		Group:  group,
		Symbol: symbol,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := settings.NewOpenapiSettingsClient(q.client).QueryUserRateLimit(ctx, req)
	if err != nil {
		galert.Error(ctx, "getMixer QueryUserRateLimit error, "+err.Error())
		return 0, false
	}
	glog.Debug(ctx, "QueryUserRateLimit info", glog.Any("limit-req", req), glog.Any("limit-resp", resp))
	switch resp.RetCode {
	case 0: // pass
	default:
		glog.Info(ctx, "mixer openapi error", glog.Any("req", req), glog.Any("resp", resp))
		return 0, false
	}

	return resp.Limit, true
}

func (q *quotaManager) getKey(uid int64, app, group, symbol string) string {
	return app + ":" + cast.Int64toa(uid) + ":" + group + ":" + symbol // futures:12345:position:BTCUSDT
}

// RateLimitSignal member rate limit message
type RateLimitSignal struct {
	APP     string                 `json:"app"` // futures
	UserID  int64                  `json:"user_id"`
	Changes map[string][]rateLimit `json:"changes"`
}

type rateLimit struct {
	Symbol       string `json:"symbol"`
	NewRateLimit int64  `json:"new_rate_limit"`
}

func (q *quotaManager) HandleQuotaMessage(ctx context.Context, msg *gkafka.Message) {
	var rate RateLimitSignal
	if err := util.JsonUnmarshal(msg.Value, &rate); err != nil {
		glog.Error(ctx, "HandleQuotaMessage Unmarshal error", glog.String("error", err.Error()))
		return
	}
	glog.Info(ctx, "member rate limit msg", glog.Int64("uid", rate.UserID), glog.Any("rateLimit-msg", rate), glog.Int64("offset", msg.Offset))
	if rate.UserID <= 0 {
		return
	}

	for group, limits := range rate.Changes {
		for _, limit := range limits {
			if rate.APP == "" {
				continue
			}
			key := q.getKey(rate.UserID, rate.APP, group, limit.Symbol)
			glog.Debug(ctx, "member rate limit symbol will update", glog.String("key", key))

			expire := cacheLongExpireSeconds + rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
			if err := q.quotaCache.Set([]byte(key), []byte(cast.Int64toa(limit.NewRateLimit)), expire); err != nil {
				glog.Info(ctx, "member rate limit cache update error", glog.String("key", key), glog.String("error", err.Error()))
			}
		}
	}
}

func lqOnErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "copytrade consumer err "+err.Error())
	}
}

type quotaDiagnose struct {
	svc  *quotaManager
	cfg  zrpc.RpcClientConf
	kCfg kafka.UniversalClientConfig
}

func (o *quotaDiagnose) Key() string {
	return mixerOpenapi
}

func (o *quotaDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["kafka"] = diagnosis.DiagnoseKafka(ctx, constant.EventRateLimitChange, o.kCfg)
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	resp["redis"] = diagnosis.DiagnoseRedis(ctx)
	resp["etcd"] = diagnosis.DiagnoseEtcd(ctx)
	return resp, nil
}
