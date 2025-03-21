package gquote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/ghdts"
	"code.bydev.io/fbu/gateway/gway.git/gquote/option/quote/index"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

const (
	obuTickPeriod = time.Hour * 24
)

const (
	obuQuoteIndexServiceName = "option-quote-index"

	// obuHdtsTopicEdp             = "obu_quote_edp"
	// obuHdtsTopicIndexPrice      = "obu_quote_index_price"
	// obuHdtsTopicMarkPrice       = "obu_quote_mark_price"
	// obuHdtsTopicUnderlyingPrice = "obu_quote_underlying_price"
	obuHdtsTopicQuotePrice = "obu_quote_quote_price"
)

var (
	defaultObuBaseCoins  = []string{"BTC", "ETH", "SOL"}
	defaultObuQuoteCoins = []string{"USD"}
)

var globalOptionService = newOptionService()

func GetOptionService() OptionService {
	return globalOptionService
}

func SetMockQuotePrice(obj interface{}) {
	var msg *index.QuotePrice
	switch x := obj.(type) {
	case *index.QuotePrice:
		msg = x
	case string:
		msg = &index.QuotePrice{}
		if err := json.Unmarshal([]byte(x), &msg); err != nil {
			panic(fmt.Errorf("err: %v, data: %v", err, x))
		}
	default:
		panic(fmt.Errorf("invalid QuotePrice type"))
	}
	globalOptionService.(*optionService).parseQuotePrice(msg)
}

type Greeks struct {
	Delta decimal.Decimal
	Gamma decimal.Decimal
	Theta decimal.Decimal
	Vega  decimal.Decimal
}

type OptionOrderBookPrice struct {
	BestAsk     decimal.Decimal
	BestAskSize decimal.Decimal
	BestBid     decimal.Decimal
	BestBidSize decimal.Decimal
}

type OptionQuotePrice struct {
	SymbolName             string
	BaseCoin               string
	QuoteCoin              string
	IndexPrice             decimal.Decimal
	MarkPrice              decimal.Decimal
	UnderlyingPrice        decimal.Decimal
	EstimatedDeliveryPrice decimal.Decimal
	UnderlyingOriginPrice  decimal.Decimal
	MarkIv                 decimal.Decimal
	UnSmoothMarkIv         decimal.Decimal
	Greeks                 Greeks
	OrderBookPrice         OptionOrderBookPrice //
	DeliveryTime           time.Time            // 行权时间
}

// isExpired 判断是否过期
func (qp *OptionQuotePrice) isExpired(now time.Time) bool {
	return !qp.DeliveryTime.IsZero() && now.Sub(qp.DeliveryTime) > time.Minute*10
}

// OptionConfig 启动配置信息,通常填空使用默认值即可
type OptionConfig struct {
	Address    string   // grpc nacos address
	BaseCoins  []string // defaultObuBaseCoins
	QuoteCoins []string // defaultObuQuoteCoins
	Logger     Logger   // default stdOut
}

type Logger interface {
	Printf(format string, args ...interface{})
}

type OptionService interface {
	// Start 启动服务,通常使用默认值即可
	Start(conf *OptionConfig) error
	// GetQuotePrice 通过Symbol名获取QuotePrice
	GetQuotePrice(symbolName string) *OptionQuotePrice
	// GetQuotePriceList 获取全部QuotePrice
	GetQuotePriceList() []*OptionQuotePrice
}

func newOptionService() OptionService {
	return &optionService{}
}

type optionService struct {
	consumers []ghdts.Consumer //
	symbolMap sync.Map         // symbol_name -> OptionPrice
	ticker    *time.Ticker     //
	logger    Logger
}

func (s *optionService) GetQuotePrice(symbol string) *OptionQuotePrice {
	value, ok := s.symbolMap.Load(symbol)
	if ok {
		res, _ := value.(*OptionQuotePrice)
		return res
	}

	return nil
}

func (s *optionService) GetQuotePriceList() []*OptionQuotePrice {
	res := make([]*OptionQuotePrice, 0, 200)
	s.symbolMap.Range(func(key, value any) bool {
		if qp, ok := value.(*OptionQuotePrice); ok {
			res = append(res, qp)
		}
		return true
	})

	return res
}

func (s *optionService) Start(conf *OptionConfig) error {
	if conf == nil {
		conf = &OptionConfig{}
	}
	s.logger = conf.Logger

	if s.logger == nil {
		s.logger = log.New(os.Stdout, "", 0)
	}
	if err := s.startGrpc(conf); err != nil {
		return err
	}

	var errors []string
	if consumer, err := ghdts.Consume(context.Background(), obuHdtsTopicQuotePrice, s.onConsumeQuotePrice); err != nil {
		errors = append(errors, err.Error())
	} else {
		s.consumers = append(s.consumers, consumer)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ","))
	}

	t := time.NewTicker(obuTickPeriod)
	s.ticker = t
	go func() {
		for range t.C {
			s.refresh()
		}
	}()

	return nil
}

func (s *optionService) Stop() error {
	for _, c := range s.consumers {
		_ = c.Close()
	}
	s.consumers = nil

	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	return nil
}

func (s *optionService) startGrpc(conf *OptionConfig) error {
	addr := strings.TrimSpace(conf.Address)
	if addr == "" {
		if env.IsProduction() {
			addr = fmt.Sprintf("nacos:///%s", obuQuoteIndexServiceName)
		} else {
			namespace := env.ProjectEnvName()
			if namespace == "" {
				// local
				namespace = "unify-test-1"
			}
			addr = fmt.Sprintf("nacos:///%s?namespace=%s", obuQuoteIndexServiceName, namespace)
		}
	}

	baseCoins := conf.BaseCoins

	if len(baseCoins) == 0 {
		baseCoins = defaultObuBaseCoins
	}

	quoteCoins := conf.QuoteCoins
	if len(quoteCoins) == 0 {
		quoteCoins = defaultObuQuoteCoins
	}

	s.logger.Printf("optionService grpc dial, addr: %v", addr)

	conn, err := ggrpc.Dial(context.Background(), addr)
	if err != nil {
		s.logger.Printf("optionService dial fail, addr: %v, err: %v", addr, err)
		return err
	}

	client := index.NewIndexPriceServiceClient(conn)

	timestamp := time.Now().Unix()
	for _, baseCoin := range baseCoins {
		for _, quoteCoin := range quoteCoins {
			req := index.QuotePriceRequest{
				ReqId:     randString(),
				BaseCoin:  baseCoin,
				QuoteCoin: quoteCoin,
				Timestamp: timestamp,
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			rsp, err := client.Query(ctx, &req)
			cancel()
			if err != nil {
				return err
			}
			if err := s.parseQuotePrice(rsp); err != nil {
				return err
			}
		}
	}

	if cc, ok := conn.(io.Closer); ok {
		_ = cc.Close()
	}

	s.logger.Printf("optionService process grpc success, addr: %v, size: %v", addr, len(s.GetQuotePriceList()))

	return nil
}

func (s *optionService) onConsumeQuotePrice(ctx context.Context, msg *ghdts.ConsumerMessage) error {
	quotePrice := &index.QuotePrice{}
	if err := proto.Unmarshal(msg.Value, quotePrice); err != nil {
		log.Printf("unmarshal fail, topic: %v, offset: %v, err: %v", msg.Topic, msg.Offset, err)
		return err
	}

	return s.parseQuotePrice(quotePrice)
}

func (s *optionService) parseQuotePrice(rsp *index.QuotePrice) error {
	symbolMap := make(map[string]*OptionQuotePrice)

	for symbol, v := range rsp.MarkPriceMap {
		qp := s.getOrCreateBySymbol(symbolMap, symbol, rsp)
		qp.MarkPrice = toDecimal(v.UnscaledValue, v.Scale)
	}

	for symbol, v := range rsp.MarkIVMap {
		qp := s.getOrCreateBySymbol(symbolMap, symbol, rsp)
		qp.MarkIv = toDecimal(v.UnscaledValue, v.Scale)
	}

	for symbol, g := range rsp.GreeksMap {
		qp := s.getOrCreateBySymbol(symbolMap, symbol, rsp)
		if g.Delta != nil {
			qp.Greeks.Delta = toDecimal(g.Delta.UnscaledValue, g.Delta.Scale)
		}
		if g.Gamma != nil {
			qp.Greeks.Gamma = toDecimal(g.Gamma.UnscaledValue, g.Gamma.Scale)
		}
		if g.Theta != nil {
			qp.Greeks.Theta = toDecimal(g.Theta.UnscaledValue, g.Theta.Scale)
		}
		if g.Vega != nil {
			qp.Greeks.Vega = toDecimal(g.Vega.UnscaledValue, g.Vega.Scale)
		}
	}

	for symbol, v := range rsp.UnSmoothMarkIVMap {
		qp := s.getOrCreateBySymbol(symbolMap, symbol, rsp)
		qp.UnSmoothMarkIv = toDecimal(v.UnscaledValue, v.Scale)
	}

	for symbol, v := range rsp.OrderBookPriceMap {
		qp := s.getOrCreateBySymbol(symbolMap, symbol, rsp)
		if v.BestAsk != nil {
			qp.OrderBookPrice.BestAsk = toDecimal(v.BestAsk.UnscaledValue, v.BestAsk.Scale)
		}
		if v.BestAskSize != nil {
			qp.OrderBookPrice.BestAskSize = toDecimal(v.BestAskSize.UnscaledValue, v.BestAskSize.Scale)
		}
		if v.BestBid != nil {
			qp.OrderBookPrice.BestBid = toDecimal(v.BestBid.UnscaledValue, v.BestBid.Scale)
		}
		if v.BestBidSize != nil {
			qp.OrderBookPrice.BestBidSize = toDecimal(v.BestBidSize.UnscaledValue, v.BestBidSize.Scale)
		}
	}

	for key, qp := range symbolMap {
		if !qp.DeliveryTime.IsZero() {
			expireTime := qp.DeliveryTime.Unix()
			if m, ok := rsp.UnderlyingPriceMap[expireTime]; ok {
				qp.UnderlyingPrice = toDecimal(m.UnscaledValue, m.Scale)
			}
			if m, ok := rsp.UnderlyingOriginPriceMap[expireTime]; ok {
				qp.UnderlyingOriginPrice = toDecimal(m.UnscaledValue, m.Scale)
			}
			if m, ok := rsp.EstimatedDeliveryPriceMap[expireTime]; ok {
				qp.EstimatedDeliveryPrice = toDecimal(m.UnscaledValue, m.Scale)
			}
		}
		s.symbolMap.Store(key, qp)
	}

	return nil
}

func (s *optionService) getOrCreateBySymbol(dict map[string]*OptionQuotePrice, symbol string, msg *index.QuotePrice) *OptionQuotePrice {
	if v, ok := dict[symbol]; ok {
		return v
	}

	v := &OptionQuotePrice{
		SymbolName: symbol,
		BaseCoin:   msg.BaseCoin,
		QuoteCoin:  msg.QuoteCoin,
	}

	if msg.IndexPrice != nil {
		v.IndexPrice = toDecimal(msg.IndexPrice.UnscaledValue, msg.IndexPrice.Scale)
	}

	dt, err := parseSymbolDeliveryTime(symbol)
	if err == nil {
		v.DeliveryTime = dt
	} else {
		s.logger.Printf("optionService parse DeliveryTime fail, symbol_name: %v, err: %v", symbol, err)
	}

	dict[symbol] = v
	return v
}

// refresh 清理过期的symbol
func (s *optionService) refresh() {
	keys := make([]string, 0)
	s.symbolMap.Range(func(key, value any) bool {
		return true
	})

	// use utc time
	now := time.Now().UTC()

	retain := make([]string, 0, len(keys))
	for _, key := range keys {
		qp := s.GetQuotePrice(key)
		if qp.isExpired(now) {
			s.symbolMap.Delete(key)
		} else {
			retain = append(retain, key)
		}
	}

	s.logger.Printf("optionService refresh, time: %v, cost: %v, before_size: %v, after_size: %v", time.Now(), time.Since(now), len(keys), len(retain))
}
