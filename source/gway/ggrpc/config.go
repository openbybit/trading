package ggrpc

import (
	"net/url"
	"os"
	"strings"

	"code.bydev.io/frameworks/byone/core/discov/nacos"
	"code.bydev.io/frameworks/byone/zrpc"
)

const (
	testAddrEfficiency = "nacos-test.efficiency.ww5sawfyut0k.bitsvc.io:8848"
	testAddrInfra      = "nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848"

	defaultUsername  = "bybit-nacos"
	defaultPassword  = "bybit-nacos"
	defaultNamespace = "public"

	keyUseOldAddr = "use_dev_addr"
	keyNamespace  = "namespace"

	defaultGroup = "DEFAULT_GROUP"
)

func newConfig(target string, breaker bool) zrpc.RpcClientConf {
	cfg := zrpc.RpcClientConf{}
	cfg.Target = target
	cfg.NonBlock = true // nonblock is necessary
	midwares := zrpc.ClientMiddlewaresConf{
		Trace:      true,
		Prometheus: true,
		Breaker:    breaker,
	}
	cfg.Middlewares = midwares

	u, _ := url.Parse(target)
	if u == nil {
		return cfg
	}
	if u.Scheme == "nacos" {
		cfg.Target = ""
		cfg.Nacos = buildNacosConf(u)
		cfg.AvailabilityZone = zrpc.AvailabilityZoneClientConfig{Enable: true}
	}
	return cfg
}

func buildNacosConf(u *url.URL) nacos.NacosConf {
	q := u.Query()
	addr := u.Host
	if addr == "" {
		if q.Has(keyUseOldAddr) {
			addr = testAddrEfficiency
		} else {
			addr = os.Getenv("NACOS_REGISTRY_ADDRESS")
			if addr == "" {
				addr = testAddrInfra
			}
		}
	}

	namespace := defaultNamespace
	if q.Has(keyNamespace) {
		namespace = q.Get(keyNamespace)
	}

	group := defaultGroup
	if q.Has("group") {
		group = q.Get("group")
	}

	username := u.User.Username()
	password, _ := u.User.Password()
	if username == "" {
		username = defaultUsername
	}

	if password == "" {
		password = defaultPassword
	}
	addresses := strings.Split(addr, ",")
	serCfgs := make([]nacos.ServerConfig, 0)
	for _, addr := range addresses {
		serCfgs = append(serCfgs, nacos.ServerConfig{Address: addr})
	}

	res := nacos.NacosConf{}
	res.NamespaceId = namespace
	res.NacosConf.ServerConfigs = serCfgs
	res.Username = username
	res.Password = password
	res.AppName = os.Getenv("MY_PROJECT_NAME")
	res.TimeoutMs = 5000
	res.NotLoadCacheAtStart = true
	res.UpdateCacheWhenEmpty = true
	res.LogDir = "/tmp/nacos/log"
	res.CacheDir = "/tmp/nacos/cache"
	res.LogLevel = "info"
	key := u.Path
	key = strings.TrimPrefix(key, "/")
	res.Key = key
	res.Group = group
	return res
}
