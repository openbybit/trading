package ban

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service"
	"bgw/pkg/service/symbolconfig"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"errors"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
)

func TradeCheckBatchSymbol(ctx *types.Ctx, symbolFieldName string, app string, uid int64, siteApi bool, banStatus *UserStatusWrap) (string, error) {
	banSvc, err := GetBanService()
	if err != nil {
		return "", nil
	}
	banInfo := make(map[string]int64)
	// 批量下单的symbol前置校验
	syms, err := retrieveSymbols(ctx, symbolFieldName)
	if err != nil && (app == constant.AppTypeFUTURES || app == constant.AppTypeOPTION) {
		return "", berror.ErrInvalidRequest
	}
	glog.Debug(ctx, "retrieveSymbols", glog.Any("syms", syms))
	for s := range syms {
		opts := []Option{WithSymbol(s), WithSiteAPI(siteApi)}
		bannedReduceOnly, err := banSvc.VerifyTrade(service.GetContext(ctx), uid, app, banStatus, opts...)
		if errors.Is(err, berror.ErrInvalidRequest) {
			// 批量下单前面已经判断了symbol合法性
			continue
		}
		if err != nil {
			bizErr, ok := err.(berror.BizErr)
			if ok {
				banInfo[s] = bizErr.GetCode()
			}
			continue
		}
		if bannedReduceOnly {
			banInfo[s] = 1
		} else {
			banInfo[s] = 0
		}
	}
	return string(util.ToJSON(banInfo)), nil
}

func TradeCheckSingleSymbol(ctx *types.Ctx, app string, symbolFieldName string, uid int64, siteApi bool, banStatus *UserStatusWrap) error {
	banSvc, err := GetBanService()
	if err != nil {
		return nil
	}
	opts := retrieveSymbolOptions(ctx, siteApi, symbolFieldName)
	bannedReduceOnly, err := banSvc.VerifyTrade(service.GetContext(ctx), uid, app, banStatus, opts...)
	if err != nil {
		return err
	}
	bizmetedata.WithTradeCheckMetadata(ctx, &bizmetedata.TradeCheck{
		BannedReduceOnly: bannedReduceOnly,
	})

	return nil
}

func getErrFromBanItemMap(banType BanType, defaultError error, banItemMap map[BanType]*ban.UserStatus_BanItem) error {
	if banType == BantypeUnspecified || banItemMap == nil {
		return defaultError
	}

	if banItem, exist := banItemMap[banType]; exist {
		// 根据error_code 判断是否采用封禁的错误码 >0 采用封禁的错误码 else 采用过去的默认错误码
		if banItem.GetErrorCode() > 0 {
			return berror.NewBizErr(int64(banItem.GetErrorCode()), banItem.GetReasonText())
		}
		return defaultError
	}

	return defaultError
}

func getErrFromBanItem(banItem *ban.UserStatus_BanItem, defaultErr error) error {
	if banItem.GetErrorCode() > 0 {
		return berror.NewBizErr(int64(banItem.GetErrorCode()), banItem.GetReasonText())
	}
	return defaultErr
}

func retrieveSymbols(ctx *types.Ctx, sfn string) (map[string]struct{}, error) {
	var ss []string
	var err error
	if sfn == "" {
		ss, err = symbolconfig.GetBatchSymbol(ctx)
	} else {
		ss, err = symbolconfig.GetBatchSymbolByFieldName(ctx, sfn)
	}
	if err != nil {
		return nil, err
	}
	syms := make(map[string]struct{})
	for _, s := range ss {
		syms[s] = struct{}{}
	}
	return syms, nil
}

func retrieveSymbolOptions(ctx *types.Ctx, siteApi bool, sfn string) []Option {
	opts := []Option{WithSiteAPI(siteApi)}

	// 期货、期权统一都是symbol字段拦截。现货有不同版本，因此需要动态解析
	if sfn == "" {
		sym := symbolconfig.GetSymbol(ctx)
		// 不指定symbolFiledName，则默认为symbol字段（V5）
		opts = append(opts, WithSymbol(sym))
	} else {
		sym := symbolconfig.GetSymbolWithSymbolFiledName(ctx, sfn)
		// 一般情况下其他乱七八糟版本会指定symbol字段
		opts = append(opts, WithSymbol(sym))
	}
	return opts
}

func IsUsdcBanType(banType BanType) bool {
	switch banType {
	case BantypeUsdcPerpetualAllKo:
		return true
	case BantypeUsdcAll:
		return true
	case BantypeUsdcLu:
		return true
	case BantypeUsdcFutureAll:
		return true
	case BantypeUsdcFutureLu:
		return true
	case BantypeUsdcFutureAllKo:
		return true
	default:
		return false
	}
}
