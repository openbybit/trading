package bizmetedata

import (
	"context"

	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"google.golang.org/grpc/metadata"
)

type blockTrade struct {
}

func init() {
	gmetadata.Register(blocktradeKey, &blockTrade{})
}

type BlockTrade struct {
	TakerMemberId         int64 `json:"taker_member_id,omitempty"`
	TakerFuturesAccountId int64 `json:"taker_futures_account_id,omitempty"`
	TakerUnifiedAccountId int64 `json:"taker_unified_account_id,omitempty"`
	TakerUnifiedTradingID int64 `json:"taker_unified_trading_id,omitempty"`
	TakerOptionAccountId  int64 `json:"taker_option_account_id,omitempty"`
	TakerSpotAccountId    int64 `json:"taker_spot_account_id,omitempty"`
	TakerLoginStatus      int32 `json:"taker_login_status,omitempty"`
	TakerTradeStatus      int32 `json:"taker_trade_status,omitempty"`
	TakerWithdrawStatus   int32 `json:"taker_withdraw_status,omitempty"`
	TakerAIOFlag          bool  `json:"taker_aio_flag,omitempty"`

	MakerMemberId         int64 `json:"maker_member_id,omitempty"`
	MakerFuturesAccountId int64 `json:"maker_futures_account_id,omitempty"`
	MakerUnifiedAccountId int64 `json:"maker_unified_account_id,omitempty"`
	MakerUnifiedTradingID int64 `json:"maker_unified_trading_id,omitempty"`
	MakerOptionAccountId  int64 `json:"maker_option_account_id,omitempty"`
	MakerSpotAccountId    int64 `json:"maker_spot_account_id,omitempty"`
	MakerLoginStatus      int32 `json:"maker_login_status,omitempty"`
	MakerTradeStatus      int32 `json:"maker_trade_status,omitempty"`
	MakerWithdrawStatus   int32 `json:"maker_withdraw_status,omitempty"`
	MakerAIOFlag          bool  `json:"maker_aio_flag,omitempty"`
}

var blocktradeKey = "blocktrade"

type blocktradeCtxKey struct{}

// WithBlockTradeMetadata sets the block trade metadata in the context.
func WithBlockTradeMetadata(ctx context.Context, data *BlockTrade) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(blocktradeKey, data)
	} else {
		return context.WithValue(ctx, blocktradeCtxKey{}, data)
	}
	return nil
}

// BlockTradeFromContext extracts the block trade metadata from the context.
func BlockTradeFromContext(ctx context.Context) *BlockTrade {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(blocktradeKey)
	} else {
		v = ctx.Value(blocktradeCtxKey{})
	}
	data, ok := v.(*BlockTrade)
	if !ok {
		return nil
	}
	return data
}

// Extract extracts the block trade metadata from the context.
func (b *blockTrade) Extract(ctx context.Context) metadata.MD {
	data := BlockTradeFromContext(ctx)
	if data == nil {
		return nil
	}

	bytes, err := util.JsonMarshal(data)
	if err != nil {
		return nil
	}

	md := make(metadata.MD, 1)
	md.Set("block_trade", string(bytes))
	return md
}
