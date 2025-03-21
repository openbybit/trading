package compliance

import (
	"bytes"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/symbolconfig"
)

const metricLabelUaeCheck = "uae check"
const logGetSiteConfigErr = "uae check compliance wall get site config err"

func uaeLeverageCheck(ctx *types.Ctx, brokerID int32, uid int64, siteID string, products []string, md *metadata.Metadata) error {
	if siteID == "" || siteID == gcompliance.BybitSiteID {
		siteID = md.UserSiteID
	}
	if siteID != siteUAE {
		return nil
	}

	for _, product := range products {
		if product != md.Route.GetAppName(ctx) {
			continue
		}
		product = productMap(product)
		if product == "" {
			continue
		}

		jsonCfg, cfg, err := gw.GetSiteConfig(ctx, brokerID, uid, siteID, product, md.UserSiteID)
		if err != nil {
			gmetric.IncDefaultError("compliance_wall_sitecfg_err", metricLabelUaeCheck)
			glog.Error(ctx, logGetSiteConfigErr, glog.String("err", err.Error()))
			return nil
		}
		glog.Debug(ctx, "site config", glog.String("id", siteID), glog.String("Product", product),
			glog.String("cfg", jsonCfg))

		if cfg == nil {
			continue
		}

		// 杠杆拦截
		l := cfg.MaxLeverage
		buyL := cast.ToInt64(util.JsonGetString(ctx.Request.Body(), "buyLeverage"))
		sellL := cast.ToInt64(util.JsonGetString(ctx.Request.Body(), "sellLeverage"))
		glog.Debug(ctx, "args", glog.Int64("buyLeverage", buyL), glog.Int64("sellLeverage", sellL))
		if sellL > l || buyL > l {
			msg := fmt.Sprintf("leverage can not be greater than %d", l)
			return berror.NewBizErr(10001, msg)
		}
	}

	return nil
}

func uaeSymbolCheck(ctx *types.Ctx, brokerID int32, uid int64, siteID string, usc *UaeSymbolCheck, md *metadata.Metadata) error {
	if siteID == "" || siteID == gcompliance.BybitSiteID {
		siteID = md.UserSiteID
	}
	if siteID != siteUAE {
		return nil
	}

	var match bool
	for _, c := range usc.Category {
		if c == md.Route.GetAppName(ctx) {
			match = true
		}
	}

	if len(usc.Category) > 0 && !match {
		return nil
	}

	product := productMap(md.Route.GetAppName(ctx))
	if len(usc.Category) == 0 {
		product = "SPOT"
	}
	if product == "" {
		return nil
	}

	jsonCfg, cfg, err := gw.GetSiteConfig(ctx, brokerID, uid, siteID, product, md.UserSiteID)
	if err != nil {
		gmetric.IncDefaultError("compliance_wall_sitecfg_err", metricLabelUaeCheck)
		glog.Error(ctx, logGetSiteConfigErr, glog.String("err", err.Error()))
		return nil
	}
	glog.Debug(ctx, "site config", glog.String("id", siteID), glog.String("Product", product),
		glog.String("cfg", jsonCfg))

	if cfg == nil {
		return nil
	}

	// symbol拦截
	coinList := cfg.CoinWhiteList
	if len(coinList) == 0 {
		return nil
	}
	symbol := getSymbol(ctx, usc.SymbolField)
	baseCoin, settleCoin := getCoin(ctx, md.Route.GetAppName(ctx), symbol)
	glog.Debug(ctx, "coins", glog.String("baseCoin", baseCoin), glog.String("settleCoin", settleCoin),
		glog.Any("coin list", coinList))

	var (
		baseOK   bool
		settleOk bool
	)

	for _, c := range coinList {
		if c == baseCoin {
			baseOK = true
		}
		if c == settleCoin {
			settleOk = true
		}
	}

	if baseOK && settleOk {
		return nil
	}

	msg := fmt.Sprintf("symbol is not supported %s", symbol)
	return berror.NewBizErr(10001, msg)
}

func batchUaeSymbolCheck(ctx *types.Ctx, brokerID int32, uid int64, siteID string, usc *UaeSymbolCheck, md *metadata.Metadata) error {
	if siteID == "" || siteID == gcompliance.BybitSiteID {
		siteID = md.UserSiteID
	}
	if siteID != siteUAE {
		return nil
	}

	var match bool
	for _, c := range usc.Category {
		if c == md.Route.GetAppName(ctx) {
			match = true
		}
	}

	if !match {
		return nil
	}

	product := productMap(md.Route.GetAppName(ctx))
	if product == "" {
		return nil
	}

	jsonCfg, cfg, err := gw.GetSiteConfig(ctx, brokerID, uid, siteID, product, md.UserSiteID)
	if err != nil {
		gmetric.IncDefaultError("compliance_wall_sitecfg_err", metricLabelUaeCheck)
		glog.Error(ctx, logGetSiteConfigErr, glog.String("err", err.Error()))
		return nil
	}
	glog.Debug(ctx, "site config", glog.String("id", siteID), glog.String("Product", product),
		glog.String("cfg", jsonCfg))

	if cfg == nil {
		return nil
	}

	// symbol拦截
	coinList := cfg.CoinWhiteList
	if len(coinList) == 0 {
		return nil
	}

	symbols, err := symbolconfig.GetBatchSymbol(ctx)
	if err != nil {
		return berror.ErrInvalidRequest
	}
	syms := make(map[string]struct{})
	for _, s := range symbols {
		syms[s] = struct{}{}
	}
	batchSymbolCheck := make(map[string]int)
	for symbol := range syms {
		baseCoin, settleCoin := getCoin(ctx, md.Route.GetAppName(ctx), symbol)
		glog.Debug(ctx, "coins from symbols", glog.String("symbol", symbol), glog.String("baseCoin", baseCoin),
			glog.String("settleCoin", settleCoin))

		var (
			baseOK   bool
			settleOk bool
		)

		for _, c := range coinList {
			if c == baseCoin {
				baseOK = true
			}
			if c == settleCoin {
				settleOk = true
			}
		}

		if baseOK && settleOk {
			continue
		}
		batchSymbolCheck[symbol] = 1
	}

	if len(batchSymbolCheck) > 0 {
		md.BatchUaeSymbol = util.ToJSONString(batchSymbolCheck)
	}

	return nil
}

const (
	contractTypeLinearFutures   = 4
	contractTypeLinearPerpetual = 2
)

func getSymbol(ctx *types.Ctx, symbolFiled string) string {
	var symbol string
	if !ctx.IsGet() && !bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
		symbol = util.JsonGetString(ctx.PostBody(), symbolFiled)
	} else if ctx.IsGet() {
		symbol = string(ctx.QueryArgs().Peek(symbolFiled))
	} else {
		symbol = string(ctx.PostArgs().Peek(symbolFiled))
	}
	return symbol
}

func getCoin(ctx *types.Ctx, app, symbol string) (string, string) {
	switch app {
	case "futures":
		fm := symbolconfig.GetFutureManager()
		if fm == nil {
			glog.Debug(ctx, "get future symbol manager nil")
			return "", ""
		}
		cfg := fm.GetByName(symbol)
		if cfg == nil {
			glog.Debug(ctx, "get future symbol config nil")
			return "", ""
		}

		var (
			baseCoin   string
			settleCoin string
		)
		baseCoin = cfg.BaseCurrency
		if cfg.ContractType == contractTypeLinearFutures || cfg.ContractType == contractTypeLinearPerpetual {
			settleCoin = cfg.CoinName
		} else {
			settleCoin = cfg.BaseCurrency
		}
		return baseCoin, settleCoin
	case "spot":
		sm := symbolconfig.GetSpotManager()
		if sm == nil {
			glog.Debug(ctx, "get spot symbol manager nil")
			return "", ""
		}
		cfg := sm.GetByName(symbol)
		if cfg == nil {
			glog.Debug(ctx, "get spot symbol config nil")
			return "", ""
		}
		return cfg.BaseCoinName, cfg.SettleCoin
	default:
		return "", ""
	}
}

func productMap(category string) (product string) {
	switch category {
	case "spot":
		return "SPOT"
	case "futures":
		return "FUTURES"
	case "option":
		return "OPTION"
	default:
		return ""
	}
}
