package future

import (
	"errors"

	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
)

var (
	ErrSymbolNotExists          = errors.New("Scmeta: symbol not found")
	ErrSymbolNotInTradingStatus = errors.New("Scmeta: symbol not in trading status")
)

func (s *Scmeta) InitPerpetualSymbol(coin future.Coin) error {
	// 永续币种直接初始化在内存
	symbols, err := s.GetPerpetualSymbols(coin)
	if err != nil {
		return err
	}
	s.contLock.Lock()
	defer s.contLock.Unlock()
	for _, symbol := range symbols {
		sc, err := s.GetSymbolConfig(symbol)
		if err != nil {
			return err
		}
		if sc != nil {
			contract := contractFromConfig(sc)
			if contract != nil {
				s.contracts[symbol] = contract
			}
		}
	}
	return nil
}

// InsertOrUpdateCachedSymbolContract 更新维护的contract的信息。
// Note: UpdateDefaultConfig 在tquery中是为了更新DefaultSymbolContractConfig下的配置信息。 但是在scmeta里，已经不维护全局变量
// DefaultSymbolContractConfig了，建议使用此方法代替。
func (s *Scmeta) InsertOrUpdateCachedSymbolContract(contract *modelsv1.Contract) {
	s.contLock.Lock()
	s.contracts[future.Symbol(contract.Symbol)] = contract
	s.contLock.Unlock()
}

// GetCachedSymbolContract 由于老代码缘故，存在nil返回值
func (s *Scmeta) GetCachedSymbolContract(symbol future.Symbol) *modelsv1.Contract {
	s.contLock.RLock()
	contract, exists := s.contracts[symbol]
	s.contLock.RUnlock()
	if exists {
		return contract
	}
	sc, err := s.GetSymbolConfig(symbol)
	if sc == nil || err != nil {
		return nil
	}
	contract = contractFromConfig(sc)
	if contract != nil {
		s.contLock.Lock()
		s.contracts[symbol] = contract
		s.contLock.Unlock()
	}
	return contract
}

func (s *Scmeta) CheckSymbolCanTrade(symbol future.Symbol) (bool, error) {
	contract := s.GetCachedSymbolContract(symbol)
	if contract == nil {
		// symbol没有交易相关配置，则直接报错。
		return false, ErrSymbolNotExists
	}
	if contract.ContractStatus != futenumsv1.ContractStatus_CONTRACT_STATUS_TRADING {
		// 合约已关闭或未开启，不允许交易。
		return false, ErrSymbolNotInTradingStatus
	}
	return true, nil
}

func (s *Scmeta) HasSymbolContract(symbol future.Symbol) bool {
	contract := s.GetCachedSymbolContract(symbol)
	return contract != nil
}

func (s *Scmeta) GetExpectSettlePriceX(symbol future.Symbol) int64 {
	contract := s.GetCachedSymbolContract(symbol)
	if contract == nil {
		return 0
	}
	return contract.ExpectSettlePriceX
}

func contractFromConfig(sc *modelsv1.SymbolConfig) *modelsv1.Contract {
	if sc == nil {
		return nil
	}
	contract := &modelsv1.Contract{
		Symbol:         sc.Symbol,
		SymbolName:     sc.SymbolName,
		QuoteSymbol:    sc.QuoteSymbol,
		ContractType:   sc.ContractType,
		ContractStatus: sc.ContractStatus,
		Year:           9999,
		Coin:           sc.Coin,
		MaxPriceX:      sc.MaxPriceX,
		MinPriceX:      sc.MinPriceX,
	}
	return contract
}
