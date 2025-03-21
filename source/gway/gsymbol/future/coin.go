package future

import "code.bydev.io/fbu/future/sdk.git/pkg/future"

func (s *Scmeta) CoinName(coin future.Coin) string {
	return s.coins.CoinName(coin)
}

func (s *Scmeta) CoinFromName(coinName string) future.Coin {
	return s.coins.CoinFromName(coinName)
}

type coinNameMap map[future.Coin]string

// 新pb里不维护（刻意）映射，作为过渡这里hardcode一份
var _coins = coinNameMap{
	0:  "UNKNOWN",
	1:  "BTC",
	2:  "ETH",
	3:  "EOS",
	4:  "XRP",
	5:  "USDT",
	6:  "DOT",
	7:  "DOGE",
	8:  "LTC",
	9:  "TEST",
	10: "XLM",
	11: "USD",
	12: "BIT",
	16: "USDC",
	18: "SOL",
	20: "ADA",
	24: "LUNA",
	43: "MANA",
}

func (cm coinNameMap) CoinName(coin future.Coin) string {
	if coinName, ok := cm[coin]; ok {
		return coinName
	}
	return "UNKNOWN"
}

func (cm coinNameMap) CoinFromName(coinName string) future.Coin {
	for coin, name := range cm {
		if name == coinName {
			return coin
		}
	}
	return 0
}

func NewStaticCoinMeta() CoinMeta {
	return _coins
}

type CoinMeta interface {
	CoinName(coin future.Coin) string
	CoinFromName(coinName string) future.Coin
}
