package ban

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/service/symbolconfig"
	"context"
	"fmt"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
	"strings"
)

const (
	optionsCoinBanPrefix      = "usdc_options_coins"
	optionsCoinBanPrefixFmt   = "usdc_options_coins_%s"
	optionsCoinBanRoPrefixFmt = "usdc_options_coins_%s_lu"

	OptionsAllKO = "options_all_ko"
)

var staticOptionsBanMap = map[key]bool{
	newBanKey(DERIVATIVESType, TradeTag, "usdc_options_all"): false,
	newBanKey(DERIVATIVESType, TradeTag, "usdc_options_lu"):  true,
	newBanKey(UTAType, TradeTag, LightenUp):                  true, // todo 兼容老逻辑，从封禁项上来看没这个， 删掉
	newBanKey(UTAType, TradeTag, AllTrade):                   false,
	newBanKey(TradeType, TradeTag, LightenUp):                true,
	newBanKey(TradeType, TradeTag, AllTrade):                 false,
	newBanKey(DBUType, TradeTag, OptionsAllKO):               false,
}

func tradeCheckOptions(_ context.Context, banItem *ban.UserStatus_BanItem, opts *Options) (bannedReduceOnly bool, err error) {
	k := key{
		biz:   banItem.GetBizType(),
		tag:   banItem.GetTagName(),
		value: banItem.GetTagValue(),
	}
	// 静态交易封禁
	ro, ok := staticOptionsBanMap[k]
	if ok && ro {
		return true, nil
	} else if ok && !ro {
		return false, getErrFromBanItem(banItem, berror.ErrOpenAPIUserUsdtAllBanned)
	}
	// 期权币对级别的封禁
	if k.biz == DERIVATIVESType && k.tag == TradeTag && strings.Contains(k.value, optionsCoinBanPrefix) {
		if opts.symbol == "" {
			return false, berror.ErrInvalidRequest
		}
		symcfg := symbolconfig.GetOptionManager()
		if symcfg == nil {
			return false, nil
		}
		g := symcfg.GetByName(opts.symbol)
		if g == nil {
			return false, nil
		}
		coin := g.BaseCoin
		allBanValue := fmt.Sprintf(optionsCoinBanPrefixFmt, coin)
		allBanRoValue := fmt.Sprintf(optionsCoinBanRoPrefixFmt, coin)

		switch k.value {
		case allBanValue:
			return false, berror.ErrOpenAPIUserUsdtAllBanned
		case allBanRoValue:
			return true, nil
		}
	}
	// 其他放过
	return false, nil
}
