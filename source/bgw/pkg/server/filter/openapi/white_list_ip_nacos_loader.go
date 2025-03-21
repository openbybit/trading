package openapi

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/config_center/nacos"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"context"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"sync"
)

type ipListNaocsLoader struct {
	ctx       context.Context
	impCache  *sync.Map
	userCache *sync.Map
}

type naCfg struct {
	Data []brokerIp `json:"data"`
}
type brokerIp struct {
	BrokerId string `json:"brokerId"`
	Ips      string `json:"ipLists"`
}

const impIpListNacosKey = "BROKER_IP_DATA"
const usIpListNacosKey = "app-white-ips"

type WhiteListIPLoaderIface interface {
	GetIpWhiteList(context.Context, *user.MemberLoginExt) (string, bool)
}

func newIpListNacosLoader(ctx context.Context) (WhiteListIPLoaderIface, error) {
	glog.Info(ctx, "openapi_white_list_ip nacos receive start signal")

	il := &ipListNaocsLoader{
		ctx:       ctx,
		impCache:  &sync.Map{},
		userCache: &sync.Map{},
	}
	var na string
	var group string
	if env.IsProduction() {
		na = "public"
		group = "MP_DATA"
	} else {
		na = "optionmp"
		group = "INST_DATA"
	}
	ec, err := nacos.NewNacosConfigure(ctx, nacos.WithNameSpace(na), nacos.WithGroup(group))
	if err != nil {
		return nil, err
	}
	if e := ec.Listen(ctx, impIpListNacosKey, il); e != nil {
		return nil, e
	}

	group = "USER-SERVICE-GROUP"
	if env.IsProduction() {
		na = constant.DEFAULT_NAMESPACE
	} else {
		na = env.ProjectEnvName()
	}
	ec, err = nacos.NewNacosConfigure(ctx, nacos.WithNameSpace(na), nacos.WithGroup(group))
	if err != nil {
		return nil, err
	}
	if e := ec.Listen(ctx, usIpListNacosKey, il); e != nil {
		return nil, e
	}
	glog.Info(ctx, "openapi ip list nacos listener init success")
	return il, nil
}

func (il *ipListNaocsLoader) updateCache(mp *sync.Map, v string) error {
	nc := &naCfg{}
	err := jsoniter.Unmarshal([]byte(v), nc)
	if err != nil {
		return err
	}
	tmpMap := make(map[string]string)
	for _, d := range nc.Data {
		tmpMap[d.BrokerId] = d.Ips
	}

	delMap := make(map[string]struct{})
	mp.Range(func(key, value any) bool {
		_, ok := tmpMap[key.(string)]
		if !ok {
			delMap[key.(string)] = struct{}{}
		}
		return true
	})
	for k, vv := range tmpMap {
		mp.Store(k, vv)
	}
	for k := range delMap {
		mp.Delete(k)
	}
	return nil
}

// OnEvent fired on version changed
func (il *ipListNaocsLoader) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if re.Key == impIpListNacosKey {
		glog.Info(il.ctx, "ip whitelist OnEvent", glog.String("key", re.Key))
		err := il.updateCache(il.impCache, re.Value)
		if err != nil {
			glog.Error(context.Background(), "ip whitelist update cache failed")
		}
		return err
	}
	if re.Key == usIpListNacosKey {
		glog.Info(il.ctx, "us ip whitelist OnEvent", glog.String("key", re.Key))
		err := il.updateCache(il.userCache, re.Value)
		if err != nil {
			glog.Error(context.Background(), "us ip whitelist update cache failed")
		}
		return err
	}
	return nil
}

// GetEventType remoting etcd watch event
func (il *ipListNaocsLoader) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

func (il *ipListNaocsLoader) GetPriority() int {
	return -1
}

func (il *ipListNaocsLoader) GetIpWhiteList(_ context.Context, mle *user.MemberLoginExt) (string, bool) {
	ips, ok := il.userCache.Load(mle.AppId)
	if ok {
		return ips.(string), true
	}
	ips, ok = il.impCache.Load(mle.AppId)
	if !ok {
		return "", false
	}
	return ips.(string), true
}
