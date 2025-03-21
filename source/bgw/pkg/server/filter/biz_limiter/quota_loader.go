package biz_limiter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/coocood/freecache"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/etcd"
)

const (
	cacheExpireSeconds = 240 * 3600
	cacheSize          = 120
)

var (
	errInvalidAppName     = errors.New("invalid app name")
	errInvalidQuotaLoader = errors.New("invalid quotaLoader in redis_limiter")
)

// used to store the quota value
// key: app name
// value: data provider
var limiterRules sync.Map
var (
	quotaCache     *freecache.Cache
	quotaCacheOnce sync.Once
)

type quotaValue struct {
	UID    int64  `json:"uid"`
	AID    uint64 `json:"aid,omitempty"`
	Symbol string `json:"symbol,omitempty"`
	Group  string `json:"group,omitempty"`
	Quota  int    `json:"quota"`
}

func newQuotaValue(uid int64, group string, quote int) *quotaValue {
	return &quotaValue{
		UID:   uid,
		Group: group,
		Quota: quote,
	}
}

type quotaLoader struct {
	ctx       context.Context
	app       string
	configure config_center.Configure
	once      sync.Once
}

func newQuotaLoader(app string) *quotaLoader {
	ql := &quotaLoader{
		app: app,
	}

	return ql
}

func (ql *quotaLoader) init(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = berror.NewInterErr(err.Error())
		}
	}()
	glog.Info(ctx, "redis_limiter get appName success", glog.String("app", ql.app))

	// init quota cache
	quotaCacheOnce.Do(func() {
		// quota cache
		size := config.Global.Data.CacheSize.BizLimitQuotaCacheSize
		if size < cacheSize {
			size = cacheSize
		}

		quotaCache = freecache.NewCache(size * 1024 * 1024)
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

		glog.Info(ctx, "redis_limiter etcd listener init success", glog.String("app", ql.app), glog.String("root", ql.root()))
	})

	return
}

func (ql *quotaLoader) root() string {
	return filepath.Join(constant.RootDataPath, ql.app, constant.LimiterQuota)
}

func (ql *quotaLoader) watch(ctx context.Context) error {
	return ql.configure.Listen(ctx, ql.root(), ql)
}

// decode parse metadata
func decode(path, content string) *quotaValue {
	qv := &quotaValue{}
	if !strings.HasPrefix(content, "{") {
		qv.Quota = cast.Atoi(content)
		return qv
	}
	err := util.JsonUnmarshalString(content, qv)
	if err != nil {
		glog.Error(context.TODO(), "quotaLoader JsonUnmarshalString error", glog.String("key", path), glog.String("content", content), glog.String("error", err.Error()))
		return nil
	}

	return qv
}

// OnEvent fired on version changed
// nolint
func (ql *quotaLoader) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}

	qv := decode(re.Key, re.Value)
	if qv == nil {
		return nil
	}

	var err error
	switch re.Action {
	case observer.EventTypeAdd:
		err = ql.setCache(qv)
	case observer.EventTypeUpdate:
		err = ql.setCache(qv)
	case observer.EventTypeDel:
		ql.deleteCache(qv)
	}
	if err != nil {
		glog.Error(context.TODO(), "quotaLoader OnEvent error", glog.String("error", err.Error()))
	}

	return nil
}

// GetEventType remoting etcd watch event
// nolint
func (ql *quotaLoader) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority
// nolint
func (ql *quotaLoader) GetPriority() int {
	return -1
}

func (ql *quotaLoader) getCacheKey(uid int64, group string) []byte {
	return []byte(fmt.Sprintf("%s-%d-%s", ql.app, uid, group))
}

func (ql *quotaLoader) setCache(value *quotaValue) error {
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
	key := ql.getCacheKey(value.UID, value.Group)
	nValue := []byte(cast.Itoa(value.Quota))
	old, ok, err := quotaCache.SetAndGet(key, nValue, cacheExpireSeconds+random)
	if err != nil {
		return err
	}
	if ok && !bytes.Equal(old, nValue) {
		glog.Info(context.TODO(), "old and new value not equal", glog.String("key", string(key)), glog.String("old", string(old)),
			glog.String("new", string(nValue)))
	}
	return nil
}

func (ql *quotaLoader) deleteCache(value *quotaValue) {
	key := ql.getCacheKey(value.UID, value.Group)
	quotaCache.Del(key)
}

func (ql *quotaLoader) getCache(ctx context.Context, uid int64, group string) (int, error) {
	key := ql.getCacheKey(uid, group)
	v, err := quotaCache.Get(key)
	if err == nil {
		glog.Debug(ctx, "getQuota cache hit", glog.Int64("uid", uid), glog.String("group", group))
		return cast.Atoi(string(v)), nil
	}
	glog.Debug(ctx, "getQuota cache not hit", glog.Int64("uid", uid), glog.String("group", group))
	return 0, fmt.Errorf("not found %s", string(key))
}

var errDecode = errors.New("QuotaValue decode error")

func (ql *quotaLoader) getQuota(ctx context.Context, uid int64, group string) int {
	if quote, err := ql.getCache(ctx, uid, group); err == nil {
		return quote
	}

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
				glog.Duration("set cache cost", cacheSetEnd.Sub(decodeEnd)))
		}
	}()

	// maybe remove by freeCache lru or not user quota in etcd
	// /BGW_data/option/quota/12345/groups/query
	etcdKey := fmt.Sprintf("%s/%d/groups/%s", ql.root(), uid, group)
	glog.Debug(ctx, "etcd key", glog.String("key", etcdKey))
	errHandle := func(err error) int {
		if errors.Is(err, etcd.ErrKVPairNotFound) || errors.Is(err, errDecode) {
			if err := ql.setCache(newQuotaValue(uid, group, 0)); err != nil {
				glog.Error(ctx, "biz limiter set cache error", glog.String("key", etcdKey), glog.String("error", err.Error()))
			}
			return 0
		}
		glog.Error(ctx, "biz limiter getQuota error", glog.String("key", etcdKey), glog.String("error", err.Error()))
		return 0
	}
	value, err := ql.configure.Get(ctx, etcdKey)
	if err != nil {
		return errHandle(err)
	}

	decodeStart = time.Now()
	qv := decode(etcdKey, value)
	decodeEnd = time.Now()
	if qv == nil {
		return errHandle(errDecode)
	}

	if err = ql.setCache(qv); err != nil {
		glog.Error(ctx, "biz limiter set cache error", glog.String("key", etcdKey), glog.String("error", err.Error()))
	}
	cacheSetEnd = time.Now()

	return qv.Quota
}

func getUnifiedQuota(ctx context.Context, uid int64, group string) int {
	value, ok := limiterRules.Load(optionService)
	if !ok {
		glog.Error(ctx, "load loader failed", glog.String("app", optionService), glog.Int64("uid", uid), glog.String("group", group))
		return 0
	}

	loader, ok := value.(*quotaLoader)
	if !ok {
		glog.Error(ctx, "get loader failed", glog.String("app", optionService), glog.Int64("uid", uid), glog.String("group", group))
		return 0
	}

	return loader.getQuota(ctx, uid, group)
}
