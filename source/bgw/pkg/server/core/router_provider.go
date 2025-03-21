package core

import (
	"bytes"
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"
)

// GetUserIDFunc func(ctx *types.Ctx) (*UserInfo,bool, error)
type GetUserIDFunc func(ctx *types.Ctx) (int64, bool, error)
type GetAccountTypeFunc func(ctx context.Context, uid int64) (constant.AccountType, error)

type CtxRouteDataProvider struct {
	ctx       *types.Ctx
	userFn    GetUserIDFunc
	accountFn GetAccountTypeFunc
}

func NewCtxRouteDataProvider(ctx *types.Ctx, userFn GetUserIDFunc, accountFn GetAccountTypeFunc) RouteDataProvider {
	return &CtxRouteDataProvider{
		ctx:       ctx,
		userFn:    userFn,
		accountFn: accountFn,
	}
}

func (p *CtxRouteDataProvider) GetMethod() string {
	return cast.UnsafeBytesToString(p.ctx.Method())
}

func (p *CtxRouteDataProvider) GetPath() string {
	return cast.UnsafeBytesToString(p.ctx.Path())
}

func (p *CtxRouteDataProvider) GetValue(key string) (value string) {
	defer func() {
		if key == keyCategory {
			md := metadata.MDFromContext(p.ctx)
			md.Category = &value
		}
	}()
	if p.ctx.IsPost() {
		if bytes.HasPrefix(p.ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
			return cast.UnsafeBytesToString(p.ctx.PostArgs().Peek(key))
		}
		return util.JsonGetString(p.ctx.PostBody(), key)
	}

	return cast.UnsafeBytesToString(p.ctx.QueryArgs().Peek(key))
}

// GetValues peek all values of key and is only used for multi_category check now.
func (p *CtxRouteDataProvider) GetValues(key string) (values [][]byte) {
	defer func() {
		if key == keyCategory && len(values) == 1 {
			md := metadata.MDFromContext(p.ctx)
			ct := string(values[0])
			md.Category = &ct
		}
	}()
	if p.ctx.IsPost() {
		if bytes.HasPrefix(p.ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
			return p.ctx.PostArgs().PeekMulti(key)
		}
		return util.JsonGetAllVal(p.ctx.PostBody(), key)
	}

	return p.ctx.QueryArgs().PeekMulti(key)
}

func (p *CtxRouteDataProvider) GetUserID() (int64, bool, error) {
	if p.userFn != nil {
		return p.userFn(p.ctx)
	}

	return 0, false, fmt.Errorf("not support")
}

func (p CtxRouteDataProvider) GetAccountType(uid int64) (constant.AccountType, error) {
	if p.accountFn != nil {
		return p.accountFn(p.ctx, uid)
	}

	return constant.AccountTypeUnknown, fmt.Errorf("not support")
}
