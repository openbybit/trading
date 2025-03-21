package openapi

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

const (
	ipcheckNacosFile = "ipcheck_whitelist"

	bgwIPcheckAlertTitle = "IP检查配置更新"
)

var globalIpCheckMgr = &ipCheckMgr{}

func getIpCheckMgr() *ipCheckMgr {
	return globalIpCheckMgr
}

type IpCheckWhitelistConfig struct {
	Enable  bool               `yaml:"enable" json:"enable"` // 是否开启次功能,如果不开启则默认返回true
	UidList []int64            `yaml:"uids" json:"uids"`     // uid白名单列表
	uidMap  map[int64]struct{} // 构建后数据
}

func (c *IpCheckWhitelistConfig) build() {
	c.uidMap = make(map[int64]struct{})
	for _, uid := range c.UidList {
		c.uidMap[uid] = struct{}{}
	}
}

// ipCheckMgr 白名单中的用户跳过白名单校验
type ipCheckMgr struct {
	config atomic.Value
	client config_center.Configure // nacos client to listen remote config
	once   sync.Once
}

func (m *ipCheckMgr) Init() {
	m.once.Do(func() {
		m.doInit()
	})
}

func (m *ipCheckMgr) doInit() {
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),
		nacos.WithNameSpace(constant.BGWConfigNamespace),
	)
	if err != nil {
		glog.Error(context.Background(), "ipCheckMgr mgr get nacos client failed", glog.String("err", err.Error()))
		return
	}
	m.client = nacosCfg
	if err = nacosCfg.Listen(context.Background(), ipcheckNacosFile, m); err != nil {
		msg := fmt.Sprintf("ipCheckMgr listen error, err = %s, file = %s", err.Error(), ipcheckNacosFile)
		galert.Error(context.Background(), msg, galert.WithTitle(bgwIPcheckAlertTitle))
		return
	}
}

// CanSkipCheck 判断是否能跳过ip校验
func (m *ipCheckMgr) CanSkipIpCheck(uid int64) bool {
	// 配置加载失败，则跳过ip校验
	conf := m.GetConfig()
	if conf == nil || !conf.Enable {
		return true
	}

	_, ok := conf.uidMap[uid]
	return ok
}

func (m *ipCheckMgr) GetConfig() *IpCheckWhitelistConfig {
	res, ok := m.config.Load().(*IpCheckWhitelistConfig)
	if ok {
		return res
	}

	return nil
}

func (m *ipCheckMgr) build(data []byte) error {
	conf := &IpCheckWhitelistConfig{}

	if err := util.YamlUnmarshal(data, conf); err != nil {
		return err
	}

	conf.build()
	m.config.Store(conf)
	return nil
}

func (m *ipCheckMgr) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}

	glog.Info(context.TODO(), "ipCheckMgr OnEvent", glog.String("content", e.Value), glog.String("key", e.Key))
	if e.Value == "" {
		return nil
	}

	if err := m.build([]byte(e.Value)); err != nil {
		glog.Error(context.TODO(), "ipCheckMgr OnEvent", glog.String("content", e.Value), glog.String("key", e.Key), glog.String("err", err.Error()))
		return nil
	}

	glog.Info(context.Background(), "ipCheckMgr load ok", glog.Any("config", m.GetConfig()))

	return nil
}

func (m *ipCheckMgr) GetEventType() reflect.Type {
	return nil
}

func (m *ipCheckMgr) GetPriority() int {
	return 0
}
