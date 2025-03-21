package open_interest

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gopeninterest"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func Init() {
	filter.Register(filter.OpenInterestFilterKey, newOI)
}

var (
	once    sync.Once
	limiter gopeninterest.Limiter
)

func newOI() filter.Filter {
	return &oi{}
}

type oi struct {
	batch bool
}

func (o *oi) GetName() string {
	return filter.OpenInterestFilterKey
}

func (o *oi) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		md := metadata.MDFromContext(c)
		// only futures
		if md.Route.GetAppName(c) != constant.AppTypeFUTURES {
			return next(c)
		}

		uid := md.UID
		if uid <= 0 {
			return berror.NewInterErr("no uid in oi check")
		}

		sc, err := symbolconfig.GetSymbolConfig()
		if err != nil {
			gmetric.IncDefaultCounter("oi", "symbolconfig")
			glog.Error(c, "[oi]get symbol config failed", glog.String("err", err.Error()))
			return next(c)
		}

		if o.batch {
			symbols, err := symbolconfig.GetBatchSymbol(c)
			if err != nil {
				return berror.ErrInvalidRequest
			}
			syms := make(map[string]struct{})
			for _, s := range symbols {
				syms[s] = struct{}{}
			}

			batchOi := make(map[string]string)
			for s := range syms {
				symbol := int32(sc.SymbolFromName(s))
				if symbol == 0 {
					continue
				}

				buyOI, sellOI := limiter.CheckUserOpenInterestExceeded(uid, symbol)
				glog.Debug(c, "oi params", glog.String("symbol", s), glog.Int32("symbol int", symbol),
					glog.Bool("buyOI", buyOI), glog.Bool("sellOI", sellOI))

				batchOi[s] = convertVal(buyOI, sellOI)
			}
			md.BatchOI = string(util.ToJSON(batchOi))
			return next(c)
		}

		// get symbol from body
		d := util.JsonGetString(c.Request.Body(), "symbol")
		if d == "" {
			return berror.NewBizErr(10001, "params error: symbol invalid")
		}

		symbol := int32(sc.SymbolFromName(d))
		if symbol == 0 {
			return berror.NewBizErr(10001, "params error: symbol invalid "+d)
		}

		buyOI, sellOI := limiter.CheckUserOpenInterestExceeded(uid, symbol)
		glog.Debug(c, "oi params", glog.String("symbol", d), glog.Int32("symbol int", symbol),
			glog.Bool("buyOI", buyOI), glog.Bool("sellOI", sellOI))

		md.BuyOI = &buyOI
		md.SellOI = &sellOI
		return next(c)
	}
}

type oiInfo struct {
	Buy  bool
	Sell bool
}

// Init do filter init
func (o *oi) Init(ctx context.Context, args ...string) error {
	parse := flag.NewFlagSet("open interest", flag.ContinueOnError)
	parse.BoolVar(&o.batch, "batch", false, "batch orders")
	err := parse.Parse(args[1:])
	if err != nil {
		glog.Error(ctx, "open interest parse flags failed", glog.String("err", err.Error()))
		return err
	}

	err = initOI()
	if err != nil {
		galert.Error(ctx, fmt.Sprintf("init oi failed, %s", err.Error()))
		return err
	}
	return nil
}

func initOI() error {
	var err error
	once.Do(func() {
		oiCfg := &config.Global.OpenInterest
		if oiCfg == nil {
			err = fmt.Errorf("oi get empty config")
			return
		}

		temp := oiCfg.GetOptions("k123", "")
		k123 := strings.Split(temp, ",")
		temp = oiCfg.GetOptions("kabc", "")
		kabc := strings.Split(temp, ",")
		temp = oiCfg.GetOptions("kusdc", "")
		kusdc := strings.Split(temp, ",")
		if len(k123) == 0 || len(kabc) == 0 || len(kusdc) == 0 {
			err = fmt.Errorf("oi kafka addrs list empty")
			return
		}

		logResult := cast.StringToBool(oiCfg.GetOptions("enable_log_result", ""))
		inverse := cast.StringToBool(oiCfg.GetOptions("enable_inverse_coin", ""))
		linearUSDT := cast.StringToBool(oiCfg.GetOptions("enable_linear_usdt_coin", ""))
		linearUSDC := cast.StringToBool(oiCfg.GetOptions("enable_linear_usdc_coin", ""))

		cfg := &gopeninterest.Config{
			K123Brokers:  k123,
			KabcBrokers:  kabc,
			KusdcBrokers: kusdc,

			TopicNameTpl:         oiCfg.GetOptions("topic", ""),
			EnableLogResult:      logResult,
			EnableInverseCoin:    inverse,
			EnableLinearUSDTCoin: linearUSDT,
			EnableLinearUSDCCoin: linearUSDC,
		}

		sc, err1 := symbolconfig.GetSymbolConfig()
		if err1 != nil {
			err = fmt.Errorf("oi get symbol config failed, %s", err1.Error())
			return
		}

		limiter, err = gopeninterest.New(context.Background(), sc, cfg)
	})

	return err
}

func convertVal(buyOI, sellOI bool) string {
	var buy, sell int
	if buyOI {
		buy = 1
	}
	if sellOI {
		sell = 1
	}
	return fmt.Sprintf("%d#%d", buy, sell)
}
