package ws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	"code.bydev.io/fbu/gateway/gway.git/gconfig/nacos"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/conf"

	"bgw/pkg/common/constant"
)

var enableDebug = false

const (
	confDataIdServer = "bgws_config"     // 服务端配置
	confDataIdSdk    = "bgws_config_sdk" // sdk动态配置
)

type topicType uint8

const (
	topicTypePrivate topicType = 0 // 私有推送topic
	topicTypePublic  topicType = 1 // 公有推送topic
)

var gConfigMgr = configMgr{}

func getConfigMgr() *configMgr {
	return &gConfigMgr
}

func getAppConf() *AppConf {
	return &gConfigMgr.staticConf.App
}

func getStaticConf() *Config {
	return gConfigMgr.GetStaticConf()
}

func getDynamicConf() *dynamicConf {
	return gConfigMgr.GetDynamicConf()
}

type configMgr struct {
	staticConf  Config       // 静态配置
	dynamicConf atomic.Value // 动态配置
	sdkConf     atomic.Value // sdk配置
	topicMap    atomic.Value // 合法的topic映射关系
	topicMux    sync.Mutex   // 写并发控制
}

func (c *configMgr) GetStaticConf() *Config {
	return &c.staticConf
}

func (c *configMgr) GetSdkConf() *sdkConf {
	cfg, _ := c.sdkConf.Load().(*sdkConf)
	return cfg
}

func (c *configMgr) GetDynamicConf() *dynamicConf {
	cfg, ok := c.dynamicConf.Load().(*dynamicConf)
	if ok {
		return cfg
	}

	cfg = newDynamicConf()
	c.dynamicConf.Store(cfg)

	return cfg
}

// AddTopic 动态添加合法topic, 只增不减
func (c *configMgr) AddTopic(typ topicType, topics []string) {
	if len(topics) == 0 {
		return
	}

	c.topicMux.Lock()
	defer c.topicMux.Unlock()

	old, _ := c.topicMap.Load().(map[string]topicType)
	newTopics := make([]string, 0)
	for _, t := range topics {
		oldTyp, ok := old[t]
		if !ok {
			newTopics = append(newTopics, t)
		} else if typ != oldTyp {
			WSErrorInc("topic_type_conflict", t)
			glog.Errorf(context.Background(), "topic type conflict, %v", t)
		}
	}

	if len(newTopics) == 0 {
		return
	}

	glog.Debugf(context.Background(), "register topic: %v, %v", typ, topics)

	// copy new map
	n := make(map[string]topicType)
	for k, v := range old {
		n[k] = v
	}

	for _, t := range newTopics {
		n[t] = typ
	}

	c.topicMap.Store(n)
}

// CheckTopics 校验是否是合法topic
func (c *configMgr) CheckTopics(topics []string) (successes []string, fails []string) {
	dict, _ := c.topicMap.Load().(map[string]topicType)
	for _, t := range topics {
		if _, ok := dict[t]; ok {
			successes = append(successes, t)
		} else {
			fails = append(fails, t)
		}
	}

	return
}

// HasPrivateTopics 是否包含私有推送topic
func (c *configMgr) HasPrivateTopics(topics []string) bool {
	dict, _ := c.topicMap.Load().(map[string]topicType)
	for _, t := range topics {
		if x, ok := dict[t]; ok && x == topicTypePrivate {
			return true
		}
	}

	return false
}

// IgnorePublicTopics 忽略公有推送topic,需要严格校验
func (c *configMgr) IgnorePublicTopics(topics []string) []string {
	result := make([]string, 0, len(topics))
	dict, _ := c.topicMap.Load().(map[string]topicType)
	for _, t := range topics {
		if x, ok := dict[t]; ok && x == topicTypePublic {
			continue
		}
		result = append(result, t)
	}

	return result
}

func (c *configMgr) GetAllTopics() map[string]topicType {
	return c.topicMap.Load().(map[string]topicType)
}

// LoadStaticConfig 加载静态配置
func (c *configMgr) LoadStaticConfig() {
	config := Config{}

	dir, _ := gconfig.FindConfDir()
	path := filepath.Join(dir, "bgws.toml")
	conf.MustLoad(path, &config)

	config.WS.MaxRequestBodySize *= 1024 * 1024
	setDefaultNacosConfig(&config.Nacos)
	setDefaultNacosConfig(&config.MasqRpc.Nacos.NacosConf)
	setDefaultNacosConfig(&config.BanRpc.Nacos.NacosConf)
	setDefaultNacosConfig(&config.UserRpc.Nacos.NacosConf)

	if env.IsProduction() {
		if config.App.Cluster == "" {
			panic(fmt.Errorf("cluster is required in production"))
		}
	}

	if config.RPC.ListenUnixAddr == "" {
		if isLocalDev() {
			dir, err := os.UserHomeDir()
			if err != nil {
				panic(fmt.Errorf("get user home dir fail: %v", err))
			}
			config.RPC.ListenUnixAddr = filepath.Join(dir, "tmp/bgws.connector.sock")
		} else {
			config.RPC.ListenUnixAddr = "/tmp/bgws.connector.sock"
		}
	}

	if config.WS.ServiceName == "" {
		config.WS.ServiceName = config.App.Cluster
	}

	enableDebug = config.App.Mode == "debug"

	c.AddTopic(topicTypePrivate, config.App.PrivateTopicList)

	c.staticConf = config
}

func (c *configMgr) LoadDynamicConfig() {
	c.sdkConf.Store(newSdkConf())
	c.dynamicConf.Store(newDynamicConf())

	const group = "bgws"
	namespace := c.getNamespace()
	address := fmt.Sprintf("nacos://?namespace=%s&group=%s", namespace, group)

	dataIdServer := c.getServerConfigNacosKey(c.GetStaticConf().App.Cluster)

	client, err := nacos.New(address)
	if err != nil {
		glog.Errorf(context.Background(), "create nacos client fail", namespace, err)
		return
	}

	if err := client.Listen(context.Background(), dataIdServer, gconfig.ListenFunc(c.onLoadDynamicConfig), gconfig.WithForceGet(true)); err != nil {
		glog.Errorf(context.Background(), "listen server config fail, err: %v", err)
	}

	if err := client.Listen(context.Background(), confDataIdSdk, gconfig.ListenFunc(c.onLoadSdkConfig), gconfig.WithForceGet(true)); err != nil {
		glog.Errorf(context.Background(), "listen sdk config fail, err: %v", err)
	}

	glog.Info(context.Background(), "config init finish", glog.String("addr", address), glog.String("server_config_key", dataIdServer))
}

func (c *configMgr) getServerConfigNacosKey(cluster string) string {
	dataIdServer := confDataIdServer
	if cluster != "" {
		dataIdServer = fmt.Sprintf("%s_%s", confDataIdServer, cluster)
	}

	return dataIdServer
}

func (c *configMgr) getNamespace() string {
	namespace := constant.BGWConfigNamespace
	// namespace := "public"
	if !env.IsProduction() {
		if ns := env.ProjectEnvName(); ns != "" {
			namespace = ns
		}
	}

	return namespace
}

func (c *configMgr) onLoadDynamicConfig(ev *gconfig.Event) {
	if ev == nil || ev.Value == "" {
		return
	}

	cfg := newDynamicConf()
	if err := cfg.Parse([]byte(ev.Value)); err == nil {
		c.dynamicConf.Store(cfg)
		glog.Info(context.Background(), "load dynamic config success", glog.Any("config", cfg))
	} else {
		glog.Error(context.Background(), "load dynamic config fail", glog.NamedError("error", err))
	}
}

func (c *configMgr) onLoadSdkConfig(ev *gconfig.Event) {
	if ev == nil || ev.Value == "" {
		return
	}

	cfg := newSdkConf()
	if err := cfg.Parse(ev.Value); err == nil {
		if !cfg.Disable {
			_ = DispatchEvent(&SyncConfigEvent{})
		}
		c.sdkConf.Store(cfg)
		glog.Info(context.Background(), "load sdk config success", glog.Any("config", cfg))
	} else {
		glog.Error(context.Background(), "load sdk config fail", glog.NamedError("error", err))
	}
}
