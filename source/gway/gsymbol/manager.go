package gsymbol

import (
	"context"
	"os"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	"code.bydev.io/fbu/gateway/gway.git/gconfig/nacos"
)

const (
	nacosKeyFuture = "FUTURE_SYMBOL_CONFIG_FULL"
	nacosKeyOption = "OPTION-MP-SYMBOL-ALL"
	nacosKeySpot   = "MP-SPOT-SYMBOL"

	nacosGroupMpData = "MP_DATA"
)

// Config 启动配置信息,通常使用默认值即可
type Config struct {
	NacosAddress string
}

type Manager interface {
	Start(conf *Config) error
	Stop()

	GetFutureManager() *FutureManager
	GetOptionManager() *OptionManager
	GetSpotManager() *SpotManager
}

func isProd() bool {
	envName := os.Getenv("MY_ENV_NAME")
	return envName == "testnet" || envName == "mainnet"
}

var _ Manager = &manager{}

func newManager() *manager {
	m := &manager{}
	m.futureMgr.Store(&FutureManager{})
	m.optionMgr.Store(&OptionManager{})
	m.spotMgr.Store(&SpotManager{})
	return m
}

type manager struct {
	client gconfig.Configure

	futureMgr atomic.Value
	optionMgr atomic.Value
	spotMgr   atomic.Value
}

func (m *manager) GetFutureManager() *FutureManager {
	if x, ok := m.futureMgr.Load().(*FutureManager); ok {
		return x
	}

	return nil
}

func (m *manager) GetOptionManager() *OptionManager {
	if x, ok := m.optionMgr.Load().(*OptionManager); ok {
		return x
	}

	return nil
}

func (m *manager) GetSpotManager() *SpotManager {
	if x, ok := m.spotMgr.Load().(*SpotManager); ok {
		return x
	}

	return nil
}

func (m *manager) Start(conf *Config) error {
	address := ""
	if !isProd() {
		// 测试环境,指定namespace
		address = os.Getenv("MY_PROJECT_ENV_NAME")
	}

	if conf != nil && conf.NacosAddress != "" {
		address = conf.NacosAddress
	}

	client, err := nacos.New(address)
	if err != nil {
		return err
	}

	listenKeys := []string{nacosKeyFuture, nacosKeyOption, nacosKeySpot}
	for _, key := range listenKeys {
		if err := client.Listen(
			context.Background(),
			key,
			gconfig.ListenFunc(m.onEvent),
			gconfig.WithForceGet(true),
			gconfig.WithGroup(nacosGroupMpData),
		); err != nil {
			return err
		}
	}

	m.client = client

	return nil
}

func (m *manager) onEvent(ev *gconfig.Event) {
	switch ev.Key {
	case nacosKeyFuture:
		fm := &FutureManager{}
		if err := fm.build(ev.Value); err == nil {
			m.futureMgr.Store(fm)
		}
	case nacosKeyOption:
		om := &OptionManager{}
		if err := om.build(ev.Value); err == nil {
			m.optionMgr.Store(om)
		}
	case nacosKeySpot:
		sm := &SpotManager{}
		if err := sm.build(ev.Value); err == nil {
			m.spotMgr.Store(sm)
		}
	}
}

func (m *manager) Stop() {
}
