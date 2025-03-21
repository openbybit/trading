package gopeninterest

import (
	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
)

func (l *limiter) Limit(uid int64, symbol int32, side int32) bool {
	return l.limit(future.UserID(uid), future.Symbol(symbol), futenumsv1.Side(side))
}

func (l *limiter) limit(userID future.UserID, sym future.Symbol, side futenumsv1.Side) bool {
	l.configsLock.RLock()
	defer l.configsLock.RUnlock()

	dto := l.configs[sym]
	if dto == nil {
		return false
	}
	if side == futenumsv1.Side_SIDE_BUY {
		_, ok := dto.BuyExceededResultMap[userID]
		return ok
	} else {
		_, ok := dto.SellExceededResultMap[userID]
		return ok
	}
}

func (l *limiter) CheckUserOpenInterestExceeded(uid int64, symbol int32) (buyOI, sellOI bool) {
	return l.checkUserOpenInterestExceeded(future.UserID(uid), future.Symbol(symbol))
}

// CheckUserOpenInterestExceeded 用户限仓 可以 单向 也可以是双向
func (l *limiter) checkUserOpenInterestExceeded(userID future.UserID, sym future.Symbol) (buyOI, sellOI bool) {
	l.configsLock.RLock()
	defer l.configsLock.RUnlock()

	dto := l.configs[sym]
	if dto == nil {
		return false, false
	}
	_, buyOI = dto.BuyExceededResultMap[userID]
	_, sellOI = dto.SellExceededResultMap[userID]
	return
}

func (l *limiter) GetAllSymbol() []future.Symbol {
	l.configsLock.RLock()
	defer l.configsLock.RUnlock()

	var res []future.Symbol
	for sym := range l.configs {
		res = append(res, sym)
	}
	return res
}

// 针对GetAllSymbol优化
func (l *limiter) GetAllSymbolMap() map[future.Symbol]struct{} {
	l.configsLock.RLock()
	defer l.configsLock.RUnlock()
	res := make(map[future.Symbol]struct{})
	for sym := range l.configs {
		res[sym] = struct{}{}
	}
	return res
}
