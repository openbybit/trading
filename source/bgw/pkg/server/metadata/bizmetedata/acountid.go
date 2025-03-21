package bizmetedata

import (
	"context"

	"bgw/pkg/common/types"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"google.golang.org/grpc/metadata"
)

const (
	AccountIDSuffix = "_account_id"
)

type accountID struct{}

var accountIDKey = "account_id"

type accountIDCtxKey struct{}

func init() {
	gmetadata.Register(accountIDKey, &accountID{})
}

type AccountID struct {
	accountIDs map[string]int64
}

// NewAccountID create account id
func NewAccountID() *AccountID {
	return &AccountID{}
}

// SetAccountID set account id
func (a *AccountID) SetAccountID(app string, aid int64) {
	if a.accountIDs == nil {
		a.accountIDs = make(map[string]int64, 1)
	}
	a.accountIDs[app] = aid
}

// WithAccountIDMetadata sets the block trade metadata in the context.
func WithAccountIDMetadata(ctx context.Context, data *AccountID) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(accountIDKey, data)
	} else {
		return context.WithValue(ctx, accountIDCtxKey{}, data)
	}
	return nil
}

// AccountIDFromContext extracts the account id metadata from the context.
func AccountIDFromContext(ctx context.Context) *AccountID {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(accountIDKey)
	} else {
		v = ctx.Value(accountIDCtxKey{})
	}
	data, ok := v.(*AccountID)
	if !ok {
		return nil
	}
	return data
}

// Extract extracts the account id metadata from the context.
func (b *accountID) Extract(ctx context.Context) metadata.MD {
	data := AccountIDFromContext(ctx)
	if data == nil {
		return nil
	}

	md := make(metadata.MD, 2)
	for app, aid := range data.accountIDs {
		md.Set(app+AccountIDSuffix, cast.Int64toa(aid))
	}
	return md
}
