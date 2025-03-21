package tradingroute

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"bgw/pkg/diagnosis"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"bgw/pkg/remoting/nacos"
	"bgw/pkg/service"

	routerv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/uta/route/v1"
	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	nacosCenter "code.bydev.io/fbu/gateway/gway.git/gconfig/nacos"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	"code.bydev.io/frameworks/byone/core/syncx"
	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/coocood/freecache"
)

func IsRoutingService(registry string) bool {
	return strings.HasPrefix(registry, "routing://")
}

const (
	defaultCacheSize          = 100
	defaultCacheExpireSeconds = 3600 // 普通用户cache过期时间,异步刷新

	freshMaxSize        = 1000 // 异步刷新buffer大小
	freshMinIntervalSec = 10   // 两次刷新最小时间间隔
)

const (
	sourceCache = "cache"
	sourceRpc   = "rpc"
)

const (
	scopeFutures = "futures"
	scopeOption  = "option"
	scopeSpot    = "spot"
)

type routerType string

const (
	routerTypeCommon routerType = "common"
	routerTypeDemo   routerType = "demo"
)

var once sync.Once
var onceDemo sync.Once
var global Routing
var globalDemo Routing

func GetRouting() Routing {
	once.Do(func() {
		global = newRouting(routerTypeCommon)
	})
	return global
}

func GetDemoRouting() Routing {
	onceDemo.Do(func() {
		globalDemo = newRouting(routerTypeDemo)
	})
	return globalDemo
}

type (
	GetRouteRequest = routerv1.GetRouteRequest
)

type Endpoint struct {
	DataId     string
	Address    string
	Source     string // cache/grpc
	IsAioUser  bool   //
	ExpireTime int64  // 过期时间
}

type Routing interface {
	Namespace() string
	// IsAioUser 判断用户是否是特殊做市商,路由到all-in-one节点
	IsAioUser(ctx context.Context, userId int64) (bool, error)
	// GetEndpoint 获取路由信息
	GetEndpoint(ctx context.Context, in *GetRouteRequest) (Endpoint, error)
	// ClearInstances 清空缓存数据并返回历史数据
	ClearInstances() map[string]string
	// ClearRoutings 清空所有路由
	ClearRoutings()
	// ClearRoutingByUser 通过uid清空路由缓存
	ClearRoutingByUser(userId int64, scope string)
}

func newRouting(routerType routerType) Routing {
	r := &routing{
		routingCache: freecache.NewCache(defaultCacheSize * 1024 * 1024),
		freshCh:      make(chan *freshMessage, freshMaxSize),
		singleFlight: syncx.NewSingleFlight(),
		routerType:   routerType,
	}
	r.namespace = config.GetNamespace()
	if env.IsProduction() {
		r.namespace = constant.DEFAULT_NAMESPACE
	}
	switch routerType {
	case routerTypeCommon:
		r.routerGroup = "uta_router"
		r.engineGroup = "uta"
	case routerTypeDemo:
		r.routerGroup = "uta_router_da"
		r.engineGroup = "uta_da"
	}
	r.Init()
	return r
}

type freshMessage struct {
	UserId int64
	Scope  string
}

// https://uponly.larksuite.com/docx/M8SydGvsjosCv6xqYyluBV4UstY
type routing struct {
	namespace    string
	routerGroup  string
	engineGroup  string
	routerType   routerType
	routeClient  routerv1.RoutingAPIClient
	configClient gconfig.Configure
	instances    sync.Map
	routingCache *freecache.Cache

	freshCh  chan *freshMessage
	freshMap sync.Map

	singleFlight syncx.SingleFlight
}

type routeCache struct {
	DataId     string
	IsAioUser  bool
	ExpireTime int64 // 过期时间,unix seconds
}

type routeInfo struct {
	Addr string `json:"addr"`
}

// Namespace get namespace
func (r *routing) Namespace() string {
	return r.namespace
}

func (r *routing) Init() {
	r.initRouteClient()
	r.initConfigClient()
	r.initDiagnosis()
	go r.asyncFreshLoop()
}

func (r *routing) initDiagnosis() {
	_ = diagnosis.Register(&trDiagnose{
		key: "uta_router_" + r.routerGroup,
		cfg: config.Global.UtaRouter,
		svc: r,
	})
}

func (r *routing) initRouteClient() {
	r.routeClient = nil

	var rpcClient zrpc.Client
	var err error
	switch r.routerType {
	case routerTypeCommon:
		rpcClient, err = zrpc.NewClient(config.Global.UtaRouter, zrpc.WithDialOptions(service.DefaultDialOptions...))
	case routerTypeDemo:
		rpcClient, err = zrpc.NewClient(config.Global.UtaRouterDa, zrpc.WithDialOptions(service.DefaultDialOptions...))
	default:
		err = errors.New("unknown routerType")
	}

	if err != nil {
		glog.Errorf(context.Background(), "utaRouter rpc dial fail, error=%v", err)
		galert.Error(context.Background(), "utaRouter rpc dial fail", galert.WithField("error", err))
		return
	}

	r.routeClient = routerv1.NewRoutingAPIClient(rpcClient.Conn())

	glog.Infof(context.Background(), "dial uta_router success, namespace=%v", r.namespace)
}

func (r *routing) initConfigClient() {
	r.configClient = nil
	conf, err := nacos.GetNacosConfig(r.namespace)
	if err != nil {
		galert.Error(context.Background(), "get nacos config fail", galert.WithField("namespace", r.namespace), galert.WithField("error", err))
		return
	}
	cli, err := gnacos.NewConfigClient(conf)
	if err != nil {
		galert.Error(context.Background(), "create nacos config client fail", galert.WithField("error", err))
		return
	}

	r.configClient = nacosCenter.NewWithClient(cli, r.engineGroup)
	glog.Infof(context.Background(), "init nacos config client success, namespace=%v", r.namespace)
}

func (r *routing) ClearInstances() map[string]string {
	// 删除所有cache instance
	keys := make([]string, 0)
	result := make(map[string]string)
	r.instances.Range(func(key, value interface{}) bool {
		sk, ok := key.(string)
		if ok {
			keys = append(keys, sk)
		}
		if r, ok := value.(*routeInfo); ok {
			result[sk] = r.Addr
		}
		return true
	})

	for _, k := range keys {
		r.instances.Delete(k)
	}

	return result
}

func (r *routing) ClearRoutings() {
	r.routingCache.Clear()
}

func (r *routing) ClearRoutingByUser(userId int64, scope string) {
	if userId <= 0 {
		return
	}

	var scopes []string
	if scope == "" {
		scopes = []string{"", scopeFutures, scopeOption, scopeSpot}
	} else {
		scopes = []string{scope}
	}

	for _, sc := range scopes {
		key := toKey(userId, sc)
		r.routingCache.Del(cast.UnsafeStringToBytes(key))
	}
}

// IsAioUser 判断是否是特殊做市商,路由到特定节点
func (r *routing) IsAioUser(ctx context.Context, userId int64) (bool, error) {
	if userId == 0 {
		return false, nil
	}

	if r.routeClient == nil || r.configClient == nil {
		gmetric.IncDefaultError("routing", "invalid_client")
		glog.Error(ctx, "empty routing client")
		return false, berror.ErrDefault
	}

	route, err := r.getDataIdFromRouting(ctx, &GetRouteRequest{UserId: userId})
	if err != nil {
		return false, fmt.Errorf("GetRoute fail, err: %w", err)
	}

	return route.IsAioUser, nil
}

func (r *routing) GetEndpoint(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
	if r.routeClient == nil || r.configClient == nil {
		return Endpoint{}, berror.ErrDefault
	}

	if in.UserId == 0 {
		return Endpoint{}, berror.ErrParams
	}

	route, err := r.getDataIdFromRouting(ctx, in)
	if err != nil {
		return Endpoint{}, fmt.Errorf("GetRoute fail, err: %w", err)
	}

	addr, err := r.getInstance(ctx, route.DataId)
	if err != nil {
		return route, fmt.Errorf("getInstance fail: dataId=%v, err=%v", route.DataId, err)
	}

	route.Address = addr

	return route, nil
}

func (r *routing) getDataIdFromRouting(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
	key := toKey(in.UserId, in.Scope)
	cacheKey := cast.UnsafeStringToBytes(key)
	if value, err := r.routingCache.Get(cacheKey); err == nil && value != nil {
		rc := routeCache{}
		dec := gob.NewDecoder(bytes.NewReader(value))
		if err := dec.Decode(&rc); err == nil {
			r.freshCache(in, rc.ExpireTime, &rc)
			return Endpoint{Source: sourceCache, DataId: rc.DataId, IsAioUser: rc.IsAioUser, ExpireTime: rc.ExpireTime}, nil
		}
	}

	rsp, expireTime, err := r.invokeRouting(ctx, in, cacheKey)
	if err != nil {
		return Endpoint{}, err
	}

	return Endpoint{DataId: rsp.DataId, Source: sourceRpc, IsAioUser: !rsp.IsOldUta, ExpireTime: expireTime}, nil
}

func (r *routing) invokeRouting(ctx context.Context, in *GetRouteRequest, cacheKey []byte) (*routerv1.GetRouteResponse, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	rsp, err := r.routeClient.GetRoute(ctx, in)
	if err != nil {
		gmetric.IncDefaultError("routing", "invoke")
		return nil, 0, fmt.Errorf("GetRoute fail, err: %w", err)
	}

	expireSeconds := rsp.CacheTtl
	expireTime := int64(-1)
	if expireSeconds > 0 {
		expireTime = time.Now().Add(time.Duration(rsp.CacheTtl) * time.Second).Unix()
		if expireSeconds < defaultCacheExpireSeconds {
			expireSeconds = defaultCacheExpireSeconds
		}
	}

	if expireSeconds != 0 {
		rc := routeCache{DataId: rsp.DataId, IsAioUser: !rsp.IsOldUta, ExpireTime: expireTime}
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(&rc); err == nil {
			if len(cacheKey) == 0 {
				cacheKey = []byte(toKey(in.UserId, in.Scope))
			}
			if err := r.routingCache.Set(cacheKey, buf.Bytes(), int(expireSeconds)); err != nil {
				gmetric.IncDefaultError("routing", "cache")
			}
		}
	}

	return rsp, expireTime, nil
}

func (r *routing) getInstance(ctx context.Context, dataId string) (string, error) {
	// fast get from cache
	addr, ok := r.getInstanceFromCache(dataId)
	if ok {
		return addr, nil
	}

	do, err := r.singleFlight.Do(dataId, func() (any, error) {
		addr, ok := r.getInstanceFromCache(dataId)
		if ok {
			return addr, nil
		}

		// lazy get from nacos
		value, err := r.configClient.Get(ctx, dataId)
		if err != nil {
			return "", berror.WithMessage(err, dataId)
		}

		info, err := r.parse(dataId, value)
		if err != nil {
			return "", err
		}

		r.instances.Store(dataId, info)
		if err := r.configClient.Listen(ctx, dataId, gconfig.ListenFunc(r.onConfigChanged)); err != nil {
			glog.Errorf(ctx, "nacos listen fail,dataId=%v, err=%v", dataId, err)
			if !errors.Is(err, gconfig.ErrDuplicateListen) {
				gmetric.IncDefaultError("routing", "nacos_listen_fail")
			}
		}

		return info.Addr, nil
	})

	if err != nil {
		return "", err
	}

	return do.(string), nil
}

func (r *routing) getInstanceFromCache(dataId string) (string, bool) {
	value, ok := r.instances.Load(dataId)
	if !ok {
		return "", false
	}

	info, ok := value.(*routeInfo)
	if !ok {
		return "", false
	}

	return info.Addr, true
}

func (r *routing) onConfigChanged(ev *gconfig.Event) {
	if ev.Type == gconfig.EventTypeDelete {
		r.instances.Delete(ev.Key)
		glog.Info(context.Background(), "aio routes cache delete", glog.String("key", ev.Key))
	} else {
		info, err := r.parse(ev.Key, ev.Value)
		if err != nil {
			glog.Error(context.Background(), "aio routes config changed failed", glog.String("err", err.Error()))
		} else {
			r.instances.Store(ev.Key, info)
			glog.Info(context.Background(), "aio routes cache update", glog.String("key", ev.Key), glog.String("value", info.Addr))
		}
	}
}

func (r *routing) parse(key string, value string) (*routeInfo, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("invalid route address, key: %v", key)
	}

	info := &routeInfo{}
	if strings.HasPrefix(value, "{") {
		if err := json.Unmarshal([]byte(value), info); err != nil {
			gmetric.IncDefaultError("routing", "unmarshal_fail")
			glog.Errorf(context.Background(), "unmarshal route fail, error: %v, key: %v, value: %v", err, key, value)
			return nil, fmt.Errorf("unmarshal route fail, error: %v, key: %v, value: %v", err, key, value)
		}
	} else {
		info.Addr = value
	}
	return info, nil
}

// freshCache 刷新cache,超过60s,同步刷新，否则异步刷新
func (r *routing) freshCache(in *routerv1.GetRouteRequest, expireTime int64, out *routeCache) {
	if expireTime <= 0 {
		return
	}

	now := time.Now().Unix()
	if now < expireTime {
		return
	}

	if now > expireTime+60 {
		rsp, _, err := r.invokeRouting(context.Background(), in, nil)
		if err == nil && rsp != nil {
			out.DataId = rsp.DataId
			out.IsAioUser = !rsp.IsOldUta
		}
		return
	}

	uid := in.UserId

	if lastTime, ok := r.freshMap.Load(uid); ok && now-lastTime.(int64) < freshMinIntervalSec {
		return
	}

	select {
	case r.freshCh <- &freshMessage{UserId: in.UserId, Scope: in.Scope}:
		r.freshMap.Store(uid, now)
	default:
		gmetric.IncDefaultCounter("routing", "fresh_discard")
	}
}

func (r *routing) asyncFreshLoop() {
	for msg := range r.freshCh {
		_, _, _ = r.invokeRouting(context.Background(), &routerv1.GetRouteRequest{UserId: msg.UserId, Scope: msg.Scope}, nil)
		r.freshMap.Delete(msg.UserId)
	}
}

func toKey(uid int64, scope string) string {
	if scope == "" {
		return strconv.FormatInt(uid, 10)
	}

	return fmt.Sprintf("%v_%v", uid, scope)
}

type trDiagnose struct {
	key string
	cfg zrpc.RpcClientConf
	svc *routing
}

func (trd *trDiagnose) Key() string {
	return trd.key
}

func (trd *trDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, trd.cfg)

	var addrs []string
	trd.svc.instances.Range(func(key, value any) bool {
		info, ok := value.(*routeInfo)
		if ok {
			addrs = append(addrs, info.Addr)
		}
		return true
	})
	var errs []error
	for _, addr := range addrs {
		err := diagnosis.Dial(ctx, addr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	resp["grpc_uta_engine"] = errs
	return resp, nil
}
