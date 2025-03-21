package future

import (
	"errors"
	"fmt"
	"strconv"

	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/fbu/future/sdk.git/pkg/sccalc"
)

var (
	ErrSymbolConfigNotFound = errors.New("Scmeta, symbol Config not found.")
)

// SymbolName 获取合约名称
func (s *Scmeta) SymbolName(symbol future.Symbol) string {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "UNKNOWN"
	}
	return sc.SymbolName
}

// SymbolFromName 从 string 获取合约编号
func (s *Scmeta) SymbolFromName(symbolName string) future.Symbol {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()

	for symbol, config := range s.configs {
		if config.SymbolName == symbolName {
			return symbol
		}
	}
	return 0
}

// GetSymbolConfig 获取合约动态下发信息
func (s *Scmeta) GetSymbolConfig(symbol future.Symbol) (*modelsv1.SymbolConfig, error) {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()

	cfg := s.configs[symbol]
	if cfg == nil {
		return nil, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	} else {
		return cfg, nil
	}
}

// GetSupportedSymbols 获取当前币种所有合约
func (s *Scmeta) GetSupportedSymbols(coin future.Coin) []future.Symbol {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()

	var symbols []future.Symbol
	for symbol, config := range s.configs {
		if coin == future.Coin(config.Coin) {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

// GetPerpetualSymbols 获取当前币种所有永续合约
func (s *Scmeta) GetPerpetualSymbols(coin future.Coin) ([]future.Symbol, error) {
	var symbols []future.Symbol
	for _, symbol := range s.GetSupportedSymbols(coin) {
		sc, err := s.GetSymbolConfig(symbol)
		if err != nil {
			return nil, err
		}
		if sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL ||
			sc.ContractType == futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL {
			symbols = append(symbols, symbol)
		}
	}
	return symbols, nil
}

// IsInverse 是否为反向币种
func (s *Scmeta) IsInverse(coin future.Coin) (bool, error) {
	symbols := s.GetSupportedSymbols(coin)
	if len(symbols) == 0 {
		return false, errors.New("IsInverse: no symbol found")
	}

	sc, err := s.GetSymbolConfig(symbols[0])
	if err != nil {
		return false, err
	}

	return sccalc.IsInverse(sc)
}

func (s *Scmeta) GetQtyX(qty int64, symbol future.Symbol) (int64, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return 0, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return int64ToEN(qty, sc.QtyScale)
}

func (s *Scmeta) GetQtyXFromString(qty string, symbol future.Symbol) (int64, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return 0, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringToEN(qty, sc.QtyScale)
}

func (s *Scmeta) GetPriceX(price string, symbol future.Symbol) (int64, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return 0, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringToEN(price, sc.PriceScale)
}

func (s *Scmeta) ResetQtyXOrSizeXToString(qtyOrSizeX int64, symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringFromEN(qtyOrSizeX, sc.QtyScale, sc.LotFraction)
}

func (s *Scmeta) ResetPriceToString(price string, symbol future.Symbol) (string, error) {
	priceX, err := strconv.ParseInt(price, 10, 64)
	if err != nil {
		return "", err
	}

	return s.ResetPriceXToString(priceX, symbol)
}

func (s *Scmeta) ResetPriceXToString(priceX int64, symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringFromEN(priceX, sc.PriceScale, sc.PriceFraction)
}

func (s *Scmeta) ResetPriceXToStringWithPriceScale(priceX int64, symbol future.Symbol, priceScale int32) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}
	// 如果是0则取动态下发的
	if priceScale == 0 {
		priceScale = sc.PriceScale
	}
	return stringFromEN(priceX, priceScale, sc.PriceFraction)
}

// GetLowestRiskLimit cloned from git.bybit.com/engine/golib/mod/util/symbolconfig.GetLowestRiskInfo
func (s *Scmeta) GetLowestRiskLimit(symbol future.Symbol) (*modelsv1.SymbolConfig_RiskLimit, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil {
		return nil, errors.New(err.Error() + fmt.Sprintf("symbol %d", symbol))
	}
	for _, rl := range sc.RiskLimits {
		if rl.IsLowestRisk {
			return rl, nil
		}
	}
	return nil, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
}

func (s *Scmeta) GetRiskLimitMaxPositionValue(symbol future.Symbol, riskId int64) (int64, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil {
		return 0, errors.New(err.Error() + fmt.Sprintf("symbol %d", symbol))
	}
	for _, rl := range sc.RiskLimits {
		if rl.RiskId == riskId {
			return rl.MaxOrdPzValueX, nil
		}
	}
	return 0, fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
}

func (s *Scmeta) GetBaseCurrency(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}
	return sc.BaseCurrency, nil
}

func (s *Scmeta) GetTakerFee(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringFromE8ForSymbolInfo(sc.DefaultTakerFeeRateE8)
}

func (s *Scmeta) GetMakerFee(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return stringFromE8ForSymbolInfo(sc.DefaultMakerFeeRateE8)
}

func (s *Scmeta) GetMinLeverage(symbol future.Symbol) string {
	return "1"
}

func (s *Scmeta) GetMaxPrice(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetPriceXToString(sc.MaxPriceX, symbol)
}

func (s *Scmeta) GetMinPrice(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetPriceXToString(sc.MinPriceX, symbol)
}

func (s *Scmeta) GetMinTradingQty(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetQtyXOrSizeXToString(sc.MinQtyX, symbol)
}

func (s *Scmeta) GetMaxTradingQty(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetQtyXOrSizeXToString(sc.MaxNewOrderQtyX, symbol)
}

func (s *Scmeta) GetTickSize(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetPriceXToString(sc.TickSizeX, symbol)
}

func (s *Scmeta) GetQuoteCurrency(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return sc.QuoteCurrency, nil
}

func (s *Scmeta) GetLeverageStep(symbol future.Symbol) string {
	return "0.01"
}

func (s *Scmeta) GetQtyStep(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return s.ResetQtyXOrSizeXToString(sc.MinQtyX, symbol)
}

func (s *Scmeta) GetPriceScale(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return strconv.FormatInt(int64(sc.PriceScale), 10), nil
}

func (s *Scmeta) GetPriceFraction(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return strconv.FormatInt(int64(sc.PriceFraction), 10), nil
}

func (s *Scmeta) GetContractStatus(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return sc.GetContractStatus().String(), nil
}

func (s *Scmeta) GetSymbolAlias(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}

	return sc.SymbolAlias, nil
}

func (s *Scmeta) GetContractType(symbol future.Symbol) (string, error) {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return "", fmt.Errorf("%w, symbol: %v", ErrSymbolConfigNotFound, symbol.Label())
	}
	return sc.GetContractType().String(), nil
}

func (s *Scmeta) GetAllCoins() map[future.Coin]string {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	newCoinMap := make(map[future.Coin]string)
	for _, sc := range s.configs {
		newCoinMap[future.Coin(sc.Coin)] = sc.CoinName
	}
	return newCoinMap
}

// GetAllTradingCoins 获取除了TEST COIN外的所有coin
func (s *Scmeta) GetAllTradingCoins() map[future.Coin]string {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	newCoinMap := make(map[future.Coin]string)
	for _, sc := range s.configs {
		if sc.Coin != 9 {
			newCoinMap[future.Coin(sc.Coin)] = sc.CoinName
		}
	}
	return newCoinMap
}

// GetOnlineTradingCoins 获取线上Coin列表 过滤测试coin和已下架coin
func (s *Scmeta) GetOnlineTradingCoins() map[future.Coin]string {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	newCoinMap := make(map[future.Coin]string)
	for _, sc := range s.configs {
		if sc.Coin != 9 && sc.ContractStatus != futenumsv1.ContractStatus_CONTRACT_STATUS_CLOSED {
			newCoinMap[future.Coin(sc.Coin)] = sc.CoinName
		}
	}
	return newCoinMap
}

// GetAllCrossIdx 获取所有crossIdx
func (s *Scmeta) GetAllCrossIdx() map[string]future.CrossIdx {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	newCrossIdxMap := make(map[string]future.CrossIdx)
	for _, sc := range s.configs {
		newCrossIdxMap[sc.CrossName] = future.CrossIdx(sc.CrossIdx)
	}
	return newCrossIdxMap
}

// GetCoinByCrossIdx 获取CrossIdx对应的coin
func (s *Scmeta) GetCoinByCrossIdx(idx int32) future.Coin {
	s.cfgLock.RLock()
	defer s.cfgLock.RUnlock()
	for _, sc := range s.configs {
		if sc.CrossIdx == idx {
			return future.Coin(sc.Coin)
		}
	}
	return 0
}

func (s *Scmeta) GetSymbolToCoin(symbol future.Symbol) future.Coin {
	sc, err := s.GetSymbolConfig(symbol)
	if err != nil || sc == nil {
		return 0
	}
	return future.Coin(sc.Coin)
}

func (s *Scmeta) GetOfflineCoins() []future.Coin {
	return []future.Coin{future.Coin(s.CoinFromName("LUNA")), future.Coin(s.CoinFromName("TEST"))}
}

// 获取合约编号
func (s *Scmeta) SymbolFromCache(symbol string) future.Symbol {
	s.snLock.RLock()
	defer s.snLock.RUnlock()
	symbolName, exist := s.symbolName[symbol]
	if exist {
		return symbolName
	}
	return 0
}
