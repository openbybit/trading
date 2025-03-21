package ban

import (
	"bgw/pkg/common/berror"
	"context"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
)

type spotBanInfo struct {
	bc         bizCode
	sym        string
	settleCoin string
}
type bizCode int

const (
	SPOTAllKO = "spot_all_ko"
)

var staticSpotBanMap = map[key]bool{
	newBanKey(SPOTType, TradeTag, "spot_all"):  false,
	newBanKey(TradeType, TradeType, LightenUp): true,
	newBanKey(TradeType, TradeTag, AllTrade):   false,
	newBanKey(DBUType, TradeTag, SPOTAllKO):    false,
}

func tradeCheckSpot(_ context.Context, banItem *ban.UserStatus_BanItem) (err error) {
	k := key{
		biz:   banItem.GetBizType(),
		tag:   banItem.GetTagName(),
		value: banItem.GetTagValue(),
	}

	// 静态交易封禁
	_, ok := staticSpotBanMap[k]
	if ok {
		return getErrFromBanItem(banItem, berror.ErrOpenAPIUserLoginBanned)
	}
	// 其他放过
	return nil
}
