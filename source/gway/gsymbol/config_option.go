package gsymbol

import "encoding/json"

const (
	OptionTypeCall = "C"
	OptionTypePut  = "P"
)

/*
example:

	{
		"assetType": "OPTION",
		"baseCoin": "SOL",
		"coinpairId": 3,
		"contractSize": 10,
		"contractType": "LinearOption",
		"crossId": 10000,
		"deliveryTime": 1680249600,
		"expiry": "CURRENT_QUARTER",
		"id": 32065998,
		"multiplier": 10,
		"onlineTime": 1665043200,
		"quoteCoin": "USD",
		"settleCoin": "USDC",
		"standard": "U",
		"status": "DELIVERING",
		"strikePrice": 31000,
		"symbolName": "SOL-31MAR23-31000-C",
		"symbolType": "C"
	}
*/
type OptionConfig struct {
	AssetType    string  `json:"assetType"`    // 资产类型
	BaseCoin     string  `json:"baseCoin"`     // 标的币
	CoinPairID   int32   `json:"coinpairId"`   // 币对ID
	ContractSize int32   `json:"contractSize"` // 合约大小
	ContractType string  `json:"contractType"` // 合约类型
	CrossID      int32   `json:"crossId"`      // 撮合ID
	DeliveryTime int64   `json:"deliveryTime"` // 交割日期 单位：秒
	Expiry       string  `json:"expiry"`       // 期权周期
	ID           int32   `json:"id"`           // symbol id
	Multiplier   int32   `json:"multiplier"`   // 合约倍率
	OnlineTime   int64   `json:"onlineTime"`   // 上线时间 单位：秒
	QuoteCoin    string  `json:"quoteCoin"`    // 报价币
	SettleCoin   string  `json:"settleCoin"`   // 结算币
	Standard     string  `json:"standard"`     // 本位
	Status       string  `json:"status"`       // 期权状态
	StrikePrice  float64 `json:"strikePrice"`  // 行权价，Money类型
	SymbolName   string  `json:"symbolName"`   // symbol 名称
	SymbolType   string  `json:"symbolType"`   // C/P
}

type OptionManager struct {
	version   string
	list      []*OptionConfig
	mapByID   map[int]*OptionConfig
	mapByName map[string]*OptionConfig
}

type optionAllConfig struct {
	Data    []*OptionConfig `json:"data"`
	Version string          `json:"version"`
}

func (m *OptionManager) build(data string) error {
	c := optionAllConfig{}
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return err
	}

	m.version = c.Version
	m.list = c.Data
	m.mapByID = make(map[int]*OptionConfig)
	m.mapByName = make(map[string]*OptionConfig)

	for _, d := range c.Data {
		m.mapByID[int(d.ID)] = d
		m.mapByName[d.SymbolName] = d
	}

	return nil
}

func (m *OptionManager) GetList() []*OptionConfig {
	return m.list
}

func (m *OptionManager) GetByID(id int) *OptionConfig {
	return m.mapByID[id]
}

func (m *OptionManager) GetByName(name string) *OptionConfig {
	return m.mapByName[name]
}
