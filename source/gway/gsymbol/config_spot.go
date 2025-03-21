package gsymbol

import "encoding/json"

type SpotConfig struct {
	BaseCoin          string  `json:"baseCoin"`          // 标的币种
	BaseCoinID        int32   `json:"baseCoinId"`        // 标的币种Id
	BaseCoinName      string  `json:"baseCoinName"`      // 标的币种名称
	BaseCoinType      int32   `json:"baseCoinType"`      // 标的币类型（法币、数币、代币） 币种类型：1数币、2代币 3 法币
	BasePrecision     int32   `json:"basePrecision"`     // 标的币精度
	BrokerId          int32   `json:"brokerId"`          // 站点id
	Direction         string  `json:"direction"`         // (杠杆代币) 交易对方向: Long多，Short空
	InLeverage        float64 `json:"inleverage"`        // 杠杆倍数
	IsTest            int32   `json:"isTest"`            // 是否测试币对：0 否，1 是
	MakerBuyFee       string  `json:"makerBuyFee"`       // 买makerFee
	MakerSellFee      string  `json:"makerSellFee"`      // 卖makerFee
	MarginLoanOpen    int32   `json:"marginLoanOpen"`    // （是否全仓）可借状态 0停用，1启动
	MinPricePrecision string  `json:"minPricePrecision"` // 每次价格变动，最小的变动单位
	OpenPrice         float64 `json:"openPrice"`         // 开盘价
	SettleCoin        string  `json:"settleCoin"`        // 结算币种
	SettleCoinId      int32   `json:"settleCoinId"`      // 结算币种Id
	SettleCoinName    string  `json:"settleCoinName"`    // 结算币种名称
	SettleCoinType    int32   `json:"settleCoinType"`    // * 结算币类型（法币、数币、代币） 币种类型：1数币、2代币 3 法币
	SettlePrecision   int32   `json:"settlePrecision"`   // 结算币精度
	Status            int32   `json:"status"`            // 状态：0 停用，1 启用
	SymbolFullName    string  `json:"symbolFullName"`    // 交易对自定义名称
	SymbolId          int32   `json:"symbolId"`          // 交易对Id，全局唯一
	SymbolName        string  `json:"symbolName"`        // 交易对名称
	SymbolType        int32   `json:"symbolType"`        // 业务类型：1.币币  2.杠杆
	TakerBuyFee       string  `json:"takerBuyFee"`       // 买takerFee
	TakerSellFee      string  `json:"takerSellFee"`      // 卖takerFee
	minPriceScale     int32   // 通过MinPricePrecision计算出精度
}

func (s *SpotConfig) MinPriceScale() int32 {
	return s.minPriceScale
}

type SpotManager struct {
	version   string
	list      []*SpotConfig
	mapByID   map[int]*SpotConfig
	mapByName map[string]*SpotConfig
}

type spotAllConfig struct {
	Data    []*SpotConfig `json:"data"`
	Version string        `json:"version"`
}

func (m *SpotManager) build(data string) error {
	c := spotAllConfig{}
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return err
	}

	m.version = c.Version
	m.list = c.Data
	m.mapByID = make(map[int]*SpotConfig)
	m.mapByName = make(map[string]*SpotConfig)

	for _, d := range c.Data {
		d.minPriceScale = int32(calcScale(d.MinPricePrecision))
		m.mapByID[int(d.SymbolId)] = d
		m.mapByName[d.SymbolName] = d
	}

	return nil
}

func (m *SpotManager) GetList() []*SpotConfig {
	return m.list
}

func (m *SpotManager) GetByID(id int) *SpotConfig {
	return m.mapByID[id]
}

func (m *SpotManager) GetByName(name string) *SpotConfig {
	return m.mapByName[name]
}
