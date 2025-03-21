package future

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
)

type Scmeta struct {
	coins coinNameMap

	cfgLock sync.RWMutex
	configs map[future.Symbol]*modelsv1.SymbolConfig

	bcLock        sync.RWMutex
	brokerConfigs map[BrokerID]map[future.Symbol]*modelsv1.SymbolConfig

	contLock  sync.RWMutex
	contracts map[future.Symbol]*modelsv1.Contract

	broker                 BrokerID
	enableAllBrokerSymbols bool

	snLock     sync.RWMutex
	symbolName map[string]future.Symbol

	once sync.Once
	init chan struct{}
}

func New(ctx context.Context, cfg *Config) (*Scmeta, error) {
	m := &Scmeta{
		coins:                  _coins,
		contracts:              make(map[future.Symbol]*modelsv1.Contract),
		enableAllBrokerSymbols: cfg.AllBrokerSymbols,
		init:                   make(chan struct{}),
	}

	sc, err := newBaseSymbol(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sc.Register(m)

	select {
	case <-m.init:
	case <-time.After(5 * time.Second):
		return nil, errors.New("symbol Config init timeout")
	}

	return m, nil
}

func (s *Scmeta) OnEvent(event map[future.Symbol]*modelsv1.SymbolConfig) error {
	s.genInitMaps(event)
	s.genAllBrokerConfig(event)

	s.cfgLock.Lock()
	if s.enableAllBrokerSymbols {
		s.configs = event
	} else {
		s.configs = s.brokerConfigs[s.broker]
	}
	s.cfgLock.Unlock()

	// symbol config更新时，也需要更新contracts.
	s.contLock.Lock()
	s.contracts = make(map[future.Symbol]*modelsv1.Contract)
	s.contLock.Unlock()

	s.getSymbolNameMap(event)

	s.once.Do(func() {
		close(s.init)
	})

	return nil
}

var brokerSymbolAliasToName = make(map[BrokerID]map[string]string) // 本地站独有symbol列表 brokerSymbolAliasToName[1][BTCUSDT] = BTC2USDT，是一个本地站 special symbol列表

// genInitMaps 处理动态下发数据，生成内存各种map
func (s *Scmeta) genInitMaps(event map[future.Symbol]*modelsv1.SymbolConfig) {
	tempBrokerSymbolAliasToName := make(map[BrokerID]map[string]string)
	for _, brokerID := range BrokerID_value {
		tempBrokerSymbolAliasToName[brokerID] = make(map[string]string)
		for _, cfg := range event {
			exhibitSiteList := cfg.GetExhibitSiteList()
			if exhibitSiteList == "" || exhibitSiteList == "0" {
				continue
			}

			if cfg.GetSymbolName() != cfg.GetSymbolAlias() && containBrokerID(exhibitSiteList, brokerID) {
				tempBrokerSymbolAliasToName[brokerID][cfg.GetSymbolAlias()] = cfg.GetSymbolName()
			}
		}
	}

	// brokerID -> symbolAlias -> symbolName
	brokerSymbolAliasToName = tempBrokerSymbolAliasToName
}

func containBrokerID(exhibitSiteList string, brokerID BrokerID) bool {
	brokerIDStr := "," + strconv.Itoa(int(brokerID)) + ","
	exhibitSiteList = "," + strings.ReplaceAll(exhibitSiteList, " ", "") + ","
	return strings.Contains(exhibitSiteList, brokerIDStr)
}

// genAllBrokerConfig 处理动态下发数据，生成brokerConfigs
func (s *Scmeta) genAllBrokerConfig(event map[future.Symbol]*modelsv1.SymbolConfig) {
	tempBrokerConfigs := make(map[BrokerID]map[future.Symbol]*modelsv1.SymbolConfig)
	for _, brokerID := range BrokerID_value {
		tempBrokerConfigs[brokerID] = make(map[future.Symbol]*modelsv1.SymbolConfig)
		for symbol, cfg := range event {
			if isSymbolSupported(cfg.GetSymbolName(), cfg.GetExhibitSiteList(), brokerID) {
				tempBrokerConfigs[brokerID][symbol] = cfg
			}
		}
	}

	s.bcLock.Lock()
	s.brokerConfigs = tempBrokerConfigs
	s.bcLock.Unlock()
}

// isSymbolSupported 返回此 symbol 是否被 broker 支持
// 是一个黑名单逻辑，如果请求的 symbolName 被命中，则说明此 broker 的 相同 symbolAlias 有对应的新的 symbolName，故返回 false，不被支持
func isSymbolSupported(symbolName, exhibitSiteList string, brokerID BrokerID) bool {
	// 如果展示列表为空，则标识此 symbol 为全站 symbol
	if exhibitSiteList == "" {
		// 如果请求的 brokerID 为主站，则默认返回 true
		if brokerID == BrokerID_BYBIT {
			return true
		}

		// 如果请求的 brokerID 不为主站，则在本地站 symbol map进行命中 brokerID
		aliasToNameMap, ok := brokerSymbolAliasToName[brokerID]
		if !ok {
			// 如果 brokerID 没有命中，则默认返回 true
			return true
		}

		// aliasToNameMap 保存的为 symbol 别名到 symbol 名称的映射，不同broker的 symbolName 不同，但是 symbolAlias 相同
		// 使用 symbolName 命中，如果命中，则说明此 symbol(e.g. BTCUSDT) 有对应的 broker symbol(e.g. BTC2USDT)
		// 此 symbol 不被请求的 brokerID 支持，返回 false
		_, ok = aliasToNameMap[symbolName]

		return !ok
	}

	// 如果展示列表不为空，则验证此symbol
	return containBrokerID(exhibitSiteList, brokerID)
}

func (s *Scmeta) getSymbolNameMap(event map[future.Symbol]*modelsv1.SymbolConfig) {
	tempSymbolNameMap := make(map[string]future.Symbol)
	for symbol, cfg := range event {
		tempSymbolNameMap[cfg.SymbolName] = symbol
	}

	s.snLock.Lock()
	s.symbolName = tempSymbolNameMap
	s.snLock.Unlock()
}
