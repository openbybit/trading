package future

import (
	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
)

type BrokerID int32

const (
	BrokerID_BYBIT           BrokerID = 0
	BrokerID_SMARTBIT        BrokerID = 1 // 韩国站
	BrokerID_TURKEY          BrokerID = 2 // 土耳其站
	BrokerID_WHITE_LABEL_XM  BrokerID = 3 // 白标XM站
	BrokerID_WHITE_LABEL_ZFX BrokerID = 4 // 白标ZFX站
	BrokerID_CBU_SITE        BrokerID = 5 // 白标站标示
)

var (
	BrokerID_name = map[BrokerID]string{
		0: "BYBIT",
		1: "SMARTBIT",
		2: "TURKEY",
		3: "WHITE_LABEL_XM",
		4: "WHITE_LABEL_ZFX",
		5: "CBU_SITE",
	}
	BrokerID_value = map[string]BrokerID{
		"BYBIT":           0,
		"SMARTBIT":        1,
		"TURKEY":          2,
		"WHITE_LABEL_XM":  3,
		"WHITE_LABEL_ZFX": 4,
		"CBU_SITE":        5,
	}
)

// GetAllBrokerSymbolConfig 获取所有站点合约动态下发信息
func (s *Scmeta) GetAllBrokerSymbolConfig() map[BrokerID]map[future.Symbol]*modelsv1.SymbolConfig {
	return s.brokerConfigs
}

func (s *Scmeta) GetBrokerSymbolAliasToName() map[BrokerID]map[string]string {
	return brokerSymbolAliasToName
}

// GetSupportedSymbolsByBrokerID 获取当前币种所有合约
func (s *Scmeta) GetSupportedSymbolsByBrokerID(coin future.Coin, brokerID int32) []future.Symbol {
	brokerSymbolAliasMap := s.GetLocalSiteSymbolAliasMap(brokerID)
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	var symbols []future.Symbol
	for symbol, config := range s.configs {
		if coin == future.Coin(config.Coin) {
			if !isCanSupportedSymbolWithSymbolAliasMap(config.SymbolName, config.ExhibitSiteList, brokerID, brokerSymbolAliasMap) {
				continue
			}
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

// 是否可以支持symbol，本地站需求，带上broker下的symbol alias map
// case1 exhibitSiteList = ""，brokerID=0代表是bybit主站 会支持，brokerID>0 本地站的，如果不和自己symbol别名重复可以支持
// case2 exhibitSiteList != "" ,是否包含brokerID
func isCanSupportedSymbolWithSymbolAliasMap(symbolName, exhibitSiteList string, brokerID int32, brokerSymbolAliasMap map[string]struct{}) bool {
	if exhibitSiteList == "" {
		// 主站
		if brokerID == 0 {
			return true
		}
		// 本地站
		_, ok := brokerSymbolAliasMap[symbolName]
		return !ok
	}

	return containBrokerID(exhibitSiteList, BrokerID(brokerID))
}

// GetLocalSiteSymbolAliasMap 获取本地站的symbolalias映射
func (s *Scmeta) GetLocalSiteSymbolAliasMap(brokerID int32) map[string]struct{} {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	brokerSymbolAliasMap := make(map[string]struct{}, len(s.configs))
	for _, v := range s.configs {
		if v.ExhibitSiteList != "" && containBrokerID(v.ExhibitSiteList, BrokerID(brokerID)) {
			brokerSymbolAliasMap[v.SymbolAlias] = struct{}{}
		}
	}
	return brokerSymbolAliasMap
}

// GetChangedBrokerSymbolName 主要用于本地站入参symbolName转换，兼容传symname和symalias
func (s *Scmeta) GetChangedBrokerSymbolName(symbolName string, brokerID int32) string {
	if brokerID == 0 || symbolName == "UNKNOWN" {
		return symbolName
	}
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()

	for _, sc := range s.configs {
		if (sc.SymbolAlias == symbolName || sc.SymbolName == symbolName) &&
			containBrokerID(sc.ExhibitSiteList, BrokerID(brokerID)) {
			return sc.SymbolName
		}
	}

	return symbolName
}

// GetBrokerSymbolAlias 主要用于出参symbolName转换，兼容传主站symbol和非主站symbol
func (s *Scmeta) GetBrokerSymbolAlias(symbol future.Symbol, brokerID int32) string {
	symCfg, err := s.GetSymbolConfig(symbol)
	if err != nil || symCfg == nil {
		return "UNKNOWN"
	}
	if brokerID == 0 {
		return symCfg.SymbolName
	}
	if containBrokerID(symCfg.ExhibitSiteList, BrokerID(brokerID)) {
		return symCfg.SymbolAlias
	}

	return symCfg.SymbolName
}

// IsSupportSymbolByBrokerID 存在性能问题，未来换成缓存版的
func (s *Scmeta) IsSupportSymbolByBrokerID(symbol future.Symbol, brokerID int32) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}

	return isSymbolSupported(sc.SymbolName, sc.ExhibitSiteList, BrokerID(brokerID))
}

// IsSupportSymbolNameByBrokerID 存在性能问题，未来换成缓存版的
func (s *Scmeta) IsSupportSymbolNameByBrokerID(symbolName string, brokerID int32) bool {
	symbol := s.SymbolFromName(symbolName)
	if symbol == 0 {
		return false
	}

	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}

	return isSymbolSupported(sc.SymbolName, sc.ExhibitSiteList, BrokerID(brokerID))

}
