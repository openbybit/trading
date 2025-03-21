package gsymbol

import (
	"encoding/json"
	"strconv"
	"strings"
)

type RiskLimit struct {
	Symbol                  int32  `json:"symbol"`
	RiskId                  int64  `json:"riskId"`
	IsLowestRisk            bool   `json:"lowestRisk"` //
	MaxOrdPzValueX          int64  `json:"maxOrdPzValueX"`
	MaxLeverageE2           int64  `json:"maxLeverageE2"`
	MaintenanceMarginRateE4 int64  `json:"maintenanceMarginRateE4"`
	InitialMarginRateE4     int64  `json:"initialMarginRateE4"`
	SymbolStr               string `json:"symbolStr"`
	// Section                 []string `json:"section"`
}

// ExchangeDetail 对应IndexPriceWeight??
type ExchangeDetail struct {
	WeightE4       int64  `json:"weightE4"`
	Exchange       string `json:"exchange"`
	OriginalPairTo string `json:"originalPairTo"`
	ReqChannel     string `json:"reqChannel"`
	RspChannel     string `json:"rspChannel"`
}

// FutureConfig 和modelsv1.SymbolConfig有些许差异
// modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
// 其中含有omitempty为缺失字段
type FutureConfig struct {
	Symbol                       int32             `json:"symbol"`
	SymbolName                   string            `json:"symbolName"`
	BaseCurrency                 string            `json:"baseCurrency"`
	QuoteCurrency                string            `json:"quoteCurrency"`
	Coin                         int32             `json:"coin"`
	CoinName                     string            `json:"coinName"`
	SymbolAlias                  string            `json:"symbolAlias"`
	SymbolDesc                   string            `json:"symbolDesc"`
	QuoteSymbol                  int32             `json:"quoteSymbol"`
	ContractType                 int32             `json:"contractType"`
	ImportTimeE9                 int64             `json:"importTimeE9"`
	StartTradingTimeE9           int64             `json:"startTradingTimeE9"`
	SettleTimeE9                 int64             `json:"settleTimeE9"`
	StartCalcSettlePriceTimeE9   int64             `json:"startCalcSettlePriceTimeE9"`
	ContractStatus               int32             `json:"contractStatus"`
	CrossIdx                     int32             `json:"crossIdx"`
	CrossName                    string            `json:"crossName"`
	IsUnrealisedProfitBorrowable bool              `json:"unrealisedProfitBorrowable"` // UnrealisedProfitBorrowable
	Mode                         int               `json:"mode"`
	PriceScale                   int32             `json:"priceScale"`
	NewPriceScale                int32             `json:"newPriceScale,omitempty"` // 78
	ValueScale                   int32             `json:"valueScale"`
	OneE4                        int64             `json:"oneE4"`
	OneE8                        int64             `json:"oneE8"`
	OneX                         int64             `json:"oneX"`
	PriceFraction                int32             `json:"priceFraction"`
	TickSizeX                    int64             `json:"tickSizeX"`
	TickSizeFraction             int32             `json:"tickSizeFraction"`
	MinPriceX                    int64             `json:"minPriceX"`
	MaxPriceX                    int64             `json:"maxPriceX"`
	LotFraction                  int32             `json:"lotFraction"`
	LotSizeX                     int64             `json:"lotSizeX"`
	MinQtyX                      int64             `json:"minQtyX"`
	MaxNewOrderQtyX              int64             `json:"maxNewOrderQtyX"`
	MaxPositionSizeX             int64             `json:"maxPositionSizeX"`
	MaxOrderBookQtyX             int64             `json:"maxOrderBookQtyX"`
	WalletBalanceFraction        int32             `json:"walletBalanceFraction"`
	MinValueX                    uint64            `json:"minValueX"`
	MaxValueX                    uint64            `json:"maxValueX"`
	DefaultSettleFeeRateE8       int64             `json:"defaultSettleFeeRateE8"`
	DefaultTakerFeeRateE8        int64             `json:"defaultTakerFeeRateE8"`
	MaxTakerFeeRateE8            int64             `json:"maxTakerFeeRateE8"`
	MinTakerFeeRateE8            int64             `json:"minTakerFeeRateE8"`
	DefaultMakerFeeRateE8        int64             `json:"defaultMakerFeeRateE8"`
	MaxMakerFeeRateE8            int64             `json:"maxMakerFeeRateE8"`
	MinMakerFeeRateE8            int64             `json:"minMakerFeeRateE8"`
	OpenInterestLimitX           int64             `json:"openInterestLimitX"`
	MarkPricePassthroughCross    bool              `json:"markPricePassthroughCross"`
	HasFundingFee                bool              `json:"hasFundingFee"`
	RiskLimits                   []*RiskLimit      `json:"riskLimitList"`
	RiskLimitCount               int64             `json:"riskLimitCount"`
	StartRiskId                  int64             `json:"startRiskId"`
	BaseMaxOrdPzValueX           int64             `json:"baseMaxOrdPzValueX"`
	StepMaxOrdPzValueX           int64             `json:"stepMaxOrdPzValueX"`
	BaseMaintenanceMarginRateE4  int64             `json:"baseMaintenanceMarginRateE4"`
	StepMaintenanceMarginRateE4  int64             `json:"stepMaintenanceMarginRateE4"`
	BaseInitialMarginRateE4      int64             `json:"baseInitialMarginRateE4"`
	StepInitialMarginRateE4      int64             `json:"stepInitialMarginRateE4"`
	BuyAdlUserId                 int64             `json:"buyAdlUserId"`
	SellAdlUserId                int64             `json:"sellAdlUserId"`
	DailyAdlQtyX                 int64             `json:"dailyAdlQtyX"`
	ExchangeDetail               []*ExchangeDetail `json:"exchangeDetail"`
	OrderBookDepthValueX         int64             `json:"orderBookDepthValueX"`
	ImpactMarginNotionalX        int64             `json:"impactMarginNotionalX"`
	FundingRateClampE8           int64             `json:"fundingRateClampE8"`
	FundingRateIntervalMin       int64             `json:"fundingRateIntervalMin"`
	SettleFundingImmediately     bool              `json:"settleFundingImmediately"`
	IndexPriceOffset             int64             `json:"indexPriceOffset"`
	IndexPriceSum                int64             `json:"indexPriceSum"`
	IndexPriceCount              int64             `json:"indexPriceCount"`
	IndexSort                    float64           `json:"indexSort"`
	EnableConfig                 string            `json:"enableConfig"`
	Version                      int64             `json:"version"`
	ObDepthMergeTimes            string            `json:"obDepthMergeTimes"`
	UpnlFraction                 int32             `json:"upnlFraction"`
	QtyScale                     int32             `json:"qtyScale"`
	PriceLimitPntE6              int64             `json:"priceLimitPntE6"`
	PostOnlyFactor               int32             `json:"postOnlyFactor"`
	ExhibitSiteList              string            `json:"exhibitSiteList,omitempty"`        // 82
	OpenInterestLimitRangeE2     int64             `json:"openInterestLimitRange,omitempty"` // 83
	SettleCurrency               string            `json:"settleCurrency"`                   // pb中不存在
}

type futureMap struct {
	list       []*FutureConfig
	mapByID    map[int]*FutureConfig
	mapByName  map[string]*FutureConfig
	mapByAlias map[string]*FutureConfig
	mapByCoin  map[int][]*FutureConfig
}

type FutureManager struct {
	version   string
	list      []*FutureConfig
	brokerMap map[int]*futureMap
}

type futureAllConfig struct {
	Data    []*FutureConfig `json:"data"`
	Version string          `json:"version"`
}

func (m *FutureManager) build(data string) error {
	c := futureAllConfig{}
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return err
	}

	m.version = c.Version
	m.list = c.Data
	m.brokerMap = make(map[int]*futureMap)

	for _, f := range c.Data {
		exhibitSiteSet := m.parseExhibitSiteList(f.ExhibitSiteList)

		for brokerID := range exhibitSiteSet {
			fm := m.getOrCreateConfigMap(brokerID)
			fm.list = append(fm.list, f)
			fm.mapByID[int(f.Symbol)] = f
			fm.mapByName[f.SymbolName] = f
			fm.mapByAlias[f.SymbolAlias] = f
			fm.mapByCoin[int(f.Coin)] = append(fm.mapByCoin[int(f.Coin)], f)
		}
	}

	return nil
}

func (m *FutureManager) getOrCreateConfigMap(brokerID int) *futureMap {
	r, ok := m.brokerMap[brokerID]
	if !ok {
		r = &futureMap{
			mapByID:    make(map[int]*FutureConfig),
			mapByName:  make(map[string]*FutureConfig),
			mapByAlias: make(map[string]*FutureConfig),
			mapByCoin:  make(map[int][]*FutureConfig),
		}
		m.brokerMap[brokerID] = r
	}

	return r
}

func (m *FutureManager) parseExhibitSiteList(x string) map[int]struct{} {
	res := make(map[int]struct{})
	if x != "" {
		tokens := strings.Split(x, ",")
		for _, t := range tokens {
			b, err := strconv.Atoi(strings.TrimSpace(t))
			if err != nil {
				continue
			}
			res[b] = struct{}{}
		}
	} else {
		res[brokerID_BYBIT] = struct{}{}
	}

	return res
}

func (m *FutureManager) GetList(opts ...Option) []*FutureConfig {
	o := Options{}
	o.init(opts...)
	if x, ok := m.brokerMap[o.brokerID]; ok {
		return x.list
	}

	return nil
}

func (m *FutureManager) GetByID(id int, opts ...Option) *FutureConfig {
	o := Options{}
	o.init(opts...)
	if x, ok := m.brokerMap[o.brokerID]; ok {
		return x.mapByID[id]
	}

	return nil
}

func (m *FutureManager) GetByName(name string, opts ...Option) *FutureConfig {
	o := Options{}
	o.init(opts...)
	if x, ok := m.brokerMap[o.brokerID]; ok {
		return x.mapByName[name]
	}

	return nil
}

func (m *FutureManager) GetByAlias(name string, opts ...Option) *FutureConfig {
	o := Options{}
	o.init(opts...)
	if x, ok := m.brokerMap[o.brokerID]; ok {
		return x.mapByAlias[name]
	}

	return nil
}

func (m *FutureManager) GetByCoin(coin int, opts ...Option) []*FutureConfig {
	o := Options{}
	o.init(opts...)
	if x, ok := m.brokerMap[o.brokerID]; ok {
		return x.mapByCoin[coin]
	}

	return nil
}
