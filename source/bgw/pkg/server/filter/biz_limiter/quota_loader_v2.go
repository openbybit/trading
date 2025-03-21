package biz_limiter

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/coocood/freecache"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/etcd"
)

var (
	limitV2Loaders sync.Map

	quotaCacheV2     *freecache.Cache
	quotaCacheV2Once sync.Once
)

type quotaLoaderV2 struct {
	ctx       context.Context
	app       string
	configure config_center.Configure
	once      sync.Once
}

func newQuotaLoaderV2(app string) *quotaLoaderV2 {
	ql := &quotaLoaderV2{
		app: app,
	}

	return ql
}

func (ql *quotaLoaderV2) init(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = berror.NewInterErr(err.Error())
		}
	}()
	glog.Info(ctx, "redis_limiter_v2 get appName", glog.String("app", ql.app))

	quotaCacheV2Once.Do(func() {
		// quota cache
		size := config.Global.Data.CacheSize.BizLimitQuotaCacheSize
		if size < cacheSize {
			size = cacheSize
		}

		quotaCacheV2 = freecache.NewCache(size * 1024 * 1024)
	})

	ql.once.Do(func() {
		ec, e := etcd.NewEtcdConfigure(ctx)
		if e != nil {
			err = e
			return
		}

		if e = ec.Put(ctx, ql.root(), time.Now().String()); e != nil {
			err = e
			return
		}

		ql.configure = ec
		ql.ctx = ctx
		if e = ql.watch(ctx); e != nil {
			err = e
			return
		}

		glog.Info(ctx, "redis_limiter_v2 etcd listener init success", glog.String("app", ql.app), glog.String("root", ql.root()))
	})

	return
}

func (ql *quotaLoaderV2) root() string {
	return filepath.Join(constant.RootDataPath, constant.LimiterQuotaV2, ql.app)
}

func (ql *quotaLoaderV2) watch(ctx context.Context) error {
	return ql.configure.Listen(ctx, ql.root(), ql)
}

// OnEvent fired on version changed
// nolint
func (ql *quotaLoaderV2) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}

	qv := decode(re.Key, re.Value)
	if qv == nil {
		return nil
	}

	random := rand.Intn(14400)
	switch re.Action {
	case observer.EventTypeAdd:
		_ = quotaCacheV2.Set([]byte(re.Key), []byte(cast.Itoa(qv.Quota)), cacheExpireSeconds+random)
	case observer.EventTypeUpdate:
		_ = quotaCacheV2.Set([]byte(re.Key), []byte(cast.Itoa(qv.Quota)), cacheExpireSeconds+random)
	case observer.EventTypeDel:
		_ = quotaCacheV2.Del([]byte(re.Key))
	}

	return nil
}

// GetEventType remoting etcd watch event
// nolint
func (ql *quotaLoaderV2) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority
// nolint
func (ql *quotaLoaderV2) GetPriority() int {
	return -1
}

func (ql *quotaLoaderV2) getUidKey(uid int64) string {
	// /BGW_data/quota_v2/spot:12345
	return ql.root() + ":" + cast.Int64toa(uid)
}

func (ql *quotaLoaderV2) getKey(key string) string {
	return filepath.Join(constant.RootDataPath, constant.LimiterQuotaV2, key)
}

var (
	noExistValue       = []byte("-1")
	emptyValue         = []byte("0")
	slowQueryThreshold = time.Millisecond * 10
)

func (ql *quotaLoaderV2) getQuotaWithKey(ctx context.Context, key string) (int, bool) {
	v, err := quotaCacheV2.Get([]byte(key))
	if err == nil {
		if bytes.Equal(v, noExistValue) {
			glog.Debug(ctx, "getQuotaWithKey cache hit, but is -1", glog.String("key", key))
			return 0, false
		}
		glog.Debug(ctx, "getQuotaWithKey cache hit", glog.String("key", key))
		return cast.Atoi(string(v)), true
	}
	glog.Debug(ctx, "getQuotaWithKey cache not hit", glog.String("key", key))

	random := rand.Intn(14400)

	now := time.Now()
	var decodeStart time.Time
	var decodeEnd time.Time
	var cacheSetEnd time.Time

	defer func() {
		gmetric.ObserveDefaultLatencySince(now, "quota", "etcd")
		if time.Since(now) > slowQueryThreshold {
			glog.Info(ctx, "getQuotaWithKey time cost",
				glog.Duration("etcd cost", decodeStart.Sub(now)),
				glog.Duration("decode cost", decodeEnd.Sub(decodeStart)),
				glog.Duration("cache set", cacheSetEnd.Sub(decodeEnd)))
		}
	}()

	// maybe remove by freeCache lru or not user quota in etcd
	value, err := ql.configure.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, etcd.ErrKVPairNotFound) {
			glog.Error(ctx, "biz limiter getQuotaWithKey error", glog.String("key", key), glog.String("error", err.Error()))
			return 0, false
		}
		// kv not found
		if err = quotaCacheV2.Set([]byte(key), noExistValue, cacheExpireSeconds+random); err != nil {
			glog.Error(ctx, "biz limiter set cache error", glog.String("key", key), glog.String("error", err.Error()))
		}
		return 0, false
	}

	decodeStart = time.Now()
	qv := decode(key, value)
	decodeEnd = time.Now()
	if qv == nil {
		if err = quotaCacheV2.Set([]byte(key), emptyValue, cacheExpireSeconds+random); err != nil {
			glog.Error(ctx, "biz limiter set cache error", glog.String("key", key), glog.String("error", err.Error()))
		}
		return 0, false
	}

	if err = quotaCacheV2.Set([]byte(key), []byte(cast.Itoa(qv.Quota)), cacheExpireSeconds+random); err != nil {
		glog.Error(ctx, "biz limiter getQuotaWithKey set cache error", glog.String("key", key), glog.String("error", err.Error()))
	}
	cacheSetEnd = time.Now()
	return qv.Quota, true
}

func (ql *quotaLoaderV2) getQuota(ctx context.Context, params *rateParams) int {
	// /BGW_data/quota_v2/spot:spot/v1/order:GET:12345 100
	etcdKey := ql.getKey(params.key)
	quota, exist := ql.getQuotaWithKey(ctx, etcdKey)
	if !exist && ql.app == "spot" {
		// /BGW_data/quota_v2/spot:12345 100
		q, ok := ql.getQuotaWithKey(ctx, ql.getUidKey(params.uid))
		if ok {
			return q
		}
	}
	return quota
}

func (ql *quotaLoaderV2) getQuotaByUid(ctx context.Context, key string, uid int64) int {
	// /BGW_data/quota_v2/spot:spot/v1/order:GET:12345 100
	etcdKey := ql.getKey(key)
	quota, exist := ql.getQuotaWithKey(ctx, etcdKey)
	if !exist && ql.app == "spot" {
		// /BGW_data/quota_v2/spot:12345 100
		q, ok := ql.getQuotaWithKey(ctx, ql.getUidKey(uid))
		if ok {
			return q
		}
	}
	return quota
}
