package bizmetedata

import (
	"context"

	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"google.golang.org/grpc/metadata"
)

type tradeCheck struct{}

func init() {
	gmetadata.Register(tradeCheckKey, &tradeCheck{})
}

type TradeCheck struct {
	BannedReduceOnly bool `json:"banned_reduce_only,omitempty"`
}

var tradeCheckKey = "tradeCheck"

type tradeCheckCtxKey struct{}

// WithTradeCheckMetadata set trade check metadata in context
func WithTradeCheckMetadata(ctx context.Context, data *TradeCheck) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(tradeCheckKey, data)
	} else {
		return context.WithValue(ctx, tradeCheckCtxKey{}, data)
	}
	return nil
}

// TradeCheckFromContext extracts the block trade metadata from the context.
func TradeCheckFromContext(ctx context.Context) *TradeCheck {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(tradeCheckKey)
	} else {
		v = ctx.Value(tradeCheckCtxKey{})
	}
	data, ok := v.(*TradeCheck)
	if !ok {
		return nil
	}
	return data
}

// Extract TradeCheck metadata from context
func (t *tradeCheck) Extract(ctx context.Context) metadata.MD {
	data := TradeCheckFromContext(ctx)
	if data == nil {
		return nil
	}

	bytes, err := util.JsonMarshal(data)
	if err != nil {
		return nil
	}
	md := make(metadata.MD, 1)
	md.Set("tradecheck", string(bytes))
	return md
}
