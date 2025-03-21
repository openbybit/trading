package future

import (
	"encoding/json"
	"sort"
	"time"

	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
)

// SymbolIsClosed 判断symbol是否已经停止交易
func (s *Scmeta) SymbolIsClosed(symbol future.Symbol) bool {
	if symbol == future.Symbol(int32(0)) {
		return false
	}
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}

	if sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_CLOSED ||
		sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_SETTLING {
		return true
	}
	return false
}

// SymbolIsClosedWithCloseTime 判断symbol是否已经停止交易
func (s *Scmeta) SymbolIsClosedWithCloseTime(symbol future.Symbol, closeUnixTime int64) bool {
	if symbol == 0 {
		return false
	}
	// 有下架时间配置要和当前时间比较，小于当前时间的都认为已下架
	if closeUnixTime > 0 && time.Now().Unix() >= closeUnixTime {
		return true
	}
	config, err := s.GetSymbolConfig(symbol)
	if err != nil || config == nil {
		return false
	}
	if config.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_CLOSED ||
		config.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_SETTLING {
		return true
	}
	return false
}

// PendingClosedPerpetualSymbol 是否交易停止前30分钟内(非交割)
func (s *Scmeta) PendingClosedPerpetualSymbol(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}

	if sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_CLOSED ||
		sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES ||
		sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES {
		return false
	}

	if sc.SettleTimeE9 > 0 {
		timeDiff := sc.SettleTimeE9/1e9 - time.Now().Unix()
		// 未交割前&开始进行交割时间范围内
		if timeDiff >= 0 && sc.StartCalcSettlePriceTimeE9 <= time.Now().UnixNano() {
			return true
		}
	}
	return false
}

// PendingClosedPerpetualSymbolWithReduceOnlyUnixTime 是否交易停止前30分钟内(非交割)
func (s *Scmeta) PendingClosedPerpetualSymbolWithReduceOnlyUnixTime(symbol future.Symbol, reduceOnlyUnixTime int64) bool {
	// 有待下架时间配置的和当前时间比较，小于当前时间的都认为待下架
	if reduceOnlyUnixTime > 0 && time.Now().Unix() >= reduceOnlyUnixTime {
		return true
	}
	config, err := s.GetSymbolConfig(symbol)
	if err != nil || config == nil {
		return false
	}
	if config.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_CLOSED ||
		config.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES ||
		config.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES {
		return false
	}
	if config.SettleTimeE9 > 0 {
		timeDiff := config.SettleTimeE9/1e9 - time.Now().Unix()
		// 未交割前&开始进行交割时间范围内
		if timeDiff >= 0 && config.StartCalcSettlePriceTimeE9 <= time.Now().UnixNano() {
			return true
		}
	}
	return false
}

// IsSymbolValid 是否为已配置的的合约
func (s *Scmeta) IsSymbolValid(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}
	return s.IsValidByType(sc.ContractType)
}

// IsLinearSymbol 是否为正向合约
func (s *Scmeta) IsLinearSymbol(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}
	return s.IsLinearByType(sc.ContractType)
}

// IsLinearUsdcSymbol 是否为usdc合约
func (s *Scmeta) IsLinearUsdcSymbol(symbol future.Symbol) bool {
	cfg, err := s.GetSymbolConfig(symbol)
	if err != nil {
		return false
	}
	// usdc
	return s.IsLinearUsdc(future.Coin(cfg.Coin))
}

// IsLinearUsdc 是否usdc
func (s *Scmeta) IsLinearUsdc(coin future.Coin) bool {
	return coin == 16
}

// IsInversePerpetualSymbol 是否为反向永续合约
func (s *Scmeta) IsInversePerpetualSymbol(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}
	return s.IsInversePerpetualByType(sc.ContractType)
}

// IsInverseFutureSymbol 是否为反向交割合约
func (s *Scmeta) IsInverseFutureSymbol(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}
	return s.IsInverseFutureByType(sc.ContractType)
}

// IsActiveFuturesSymbol 是否为处于可交易状态的反向交割合约
func (s *Scmeta) IsActiveFuturesSymbol(symbol future.Symbol) bool {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return false
	}
	currentTimeStamp := time.Now().UnixNano()
	if sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES &&
		sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_TRADING &&
		currentTimeStamp <= sc.SettleTimeE9 {
		return true
	}
	return false
}

// GetInverseSymbolList 所有反向永续和反向交割
func (s *Scmeta) GetInverseSymbolList() []future.Symbol {
	var symbols []future.Symbol
	for k, v := range s.coins {
		if v == "TEST" || v == "UNKNOWN" {
			continue
		}
		isInverse, _ := s.IsInverse(k)
		if !isInverse {
			continue
		}
		symbols = append(symbols, s.GetSupportedSymbols(k)...)
	}

	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i] < symbols[j]
	})

	return symbols
}

// GetInversePerpetualSymbolList 所有反向永续列表
func (s *Scmeta) GetInversePerpetualSymbolList() []future.Symbol {
	var symbols []future.Symbol
	for k, v := range s.coins {
		if v == "TEST" || v == "UNKNOWN" {
			continue
		}
		for _, symbol := range s.GetSupportedSymbols(k) {
			if s.IsLinearSymbol(symbol) {
				break
			}
			if !s.IsInversePerpetualSymbol(symbol) {
				continue
			}
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

// GetInverseFuturesSymbolList 所有反向交割列表
func (s *Scmeta) GetInverseFuturesSymbolList() []future.Symbol {
	var symbols []future.Symbol
	for k, v := range s.coins {
		if v == "TEST" || v == "UNKNOWN" {
			continue
		}
		for _, symbol := range s.GetSupportedSymbols(k) {
			if s.IsLinearSymbol(symbol) {
				break
			}
			if !s.IsInverseFutureSymbol(symbol) {
				continue
			}
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

// GetLinearPerpetualSymbolList 所有正向永续列表(包含usdc永续)
func (s *Scmeta) GetLinearPerpetualSymbolList() []future.Symbol {
	// usdt & usdc
	coins := []future.Coin{future.Coin(5), future.Coin(16)}
	return s.getSymbolListByContractType(coins, futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL)
}

// GetLinearUSDTPerpetualSymbolList usdt正向永续列表
func (s *Scmeta) GetLinearUSDTPerpetualSymbolList() []future.Symbol {
	// usdt
	coins := []future.Coin{future.Coin(5)}
	return s.getSymbolListByContractType(coins, futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL)
}

// GetLinearUsdcPerpetualSymbols 所有正向usdc永续列表
func (s *Scmeta) GetLinearUsdcPerpetualSymbols() []future.Symbol {
	// usdc
	coins := []future.Coin{future.Coin(16)}
	return s.getSymbolListByContractType(coins, futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL)
}

func (s *Scmeta) getSymbolListByContractType(coins []future.Coin, contractType futenumsv1.ContractType) []future.Symbol {
	var symbols []future.Symbol
	for _, k := range coins {
		for _, symbol := range s.GetSupportedSymbols(k) {
			sc, _ := s.GetSymbolConfig(symbol)
			if sc.ContractType == contractType {
				symbols = append(symbols, symbol)
			}
		}
	}
	return symbols
}

// GetAllSymbolList 获取所有正反交割symbol list
func (s *Scmeta) GetAllSymbolList() []future.Symbol {
	inverse := s.GetInverseSymbolList()
	linear := s.GetLinearPerpetualSymbolList()
	return append(inverse, linear...)
}

// GetAllInverseFuturesActiveSymbolList 获取可交易状态的交割合约
func (s *Scmeta) GetAllInverseFuturesActiveSymbolList() []future.Symbol {
	var activeFutureSymbols []future.Symbol
	for _, symbol := range s.GetInverseFuturesSymbolList() {
		sc, _ := s.GetSymbolConfig(symbol)
		currentTimeStamp := time.Now().UnixNano()
		if sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES &&
			sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_TRADING &&
			currentTimeStamp <= sc.SettleTimeE9 {
			activeFutureSymbols = append(activeFutureSymbols, symbol)
		}
	}
	return activeFutureSymbols
}

// GetAllSymbolsRiskLimit 获取所有合约的风险限额列表
func (s *Scmeta) GetAllSymbolsRiskLimit() map[string][]*modelsv1.SymbolConfig_RiskLimit {
	allSymbolsRiskLimitMap := make(map[string][]*modelsv1.SymbolConfig_RiskLimit)
	for k, v := range s.coins {
		if v != "TEST" && v != "UNKNOWN" {
			for _, symbol := range s.GetSupportedSymbols(k) {
				sc, _ := s.GetSymbolConfig(symbol)
				allSymbolsRiskLimitMap[sc.SymbolName] = sc.RiskLimits
			}
		}
	}
	return allSymbolsRiskLimitMap
}

// GetInversePerpetualSymbolListByCoin 根据coin获相应反向永续列表
func (s *Scmeta) GetInversePerpetualSymbolListByCoin(coin future.Coin) []future.Symbol {
	var symbols []future.Symbol
	coinName := s.CoinName(coin)
	if coinName != "TEST" && coinName != "UNKNOWN" {
		for _, symbol := range s.GetSupportedSymbols(coin) {
			sc, _ := s.GetSymbolConfig(symbol)
			if sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL {
				symbols = append(symbols, symbol)
			}
		}
	}
	return symbols
}

// GetAllInverseFuturesActiveSymbolListByCoin 根据coin获取相应的可交易状态的交割合约
func (s *Scmeta) GetAllInverseFuturesActiveSymbolListByCoin(coin future.Coin) []future.Symbol {
	var activeFutureSymbols []future.Symbol
	coinName := s.CoinName(coin)
	if coinName != "TEST" && coinName != "UNKNOWN" {
		for _, symbol := range s.GetSupportedSymbols(coin) {
			sc, _ := s.GetSymbolConfig(symbol)
			currentTimeStamp := time.Now().UnixNano()
			if sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES &&
				sc.ContractStatus == futenumsv1.ContractStatus_CONTRACT_STATUS_TRADING &&
				currentTimeStamp <= sc.SettleTimeE9 {
				activeFutureSymbols = append(activeFutureSymbols, symbol)
			}
		}
	}
	return activeFutureSymbols
}

type EnableConfig struct {
	EnableUidList   []int64  `json:"enable_uid_list"`
	EnableShardList []string `json:"enable_shard_list"`
}

func (s *Scmeta) GetEnableUidListAndShardList(symbol future.Symbol) (uidList []int64, shardList []string, err error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return nil, nil, err
	}

	enableConfig := &EnableConfig{}
	if err := json.Unmarshal([]byte(sc.EnableConfig), enableConfig); err != nil {
		return nil, nil, err
	}

	return enableConfig.EnableUidList, enableConfig.EnableShardList, nil
}
