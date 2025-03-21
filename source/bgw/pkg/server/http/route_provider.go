package http

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter/auth"
	"bgw/pkg/server/filter/openapi"
	"bgw/pkg/service/user"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"context"
	"fmt"
)

func getUserID(ctx *types.Ctx) (int64, bool, error) {
	var uid int64
	var err error
	apikey := string(ctx.Request.Header.Peek(constant.HeaderAPIKey))
	if apikey != "" {
		uid, err = openapi.GetMemberID(ctx, apikey)
		return uid, false, err
	}
	token := auth.GetToken(ctx)
	if token != "" {
		return auth.GetMemberID(ctx, token)
	}
	apikey = getApikeyFromReq(ctx)
	if apikey == "" {
		return 0, false, berror.WithMessage(berror.ErrParams, "not support auth type")
	}
	uid, err = openapi.GetMemberID(ctx, apikey)
	return uid, false, err
}

func getApikeyFromReq(ctx *types.Ctx) string {
	if ctx.IsGet() {
		return cast.UnsafeBytesToString(ctx.QueryArgs().Peek("api_key"))
	} else {
		return util.JsonGetString(ctx.PostBody(), "api_key")
	}
}

func getAccountTypeByUID(ctx context.Context, uid int64) (constant.AccountType, error) {
	if uid == 0 {
		return constant.AccountTypeUnknown, berror.ErrParams
	}

	as, err := user.NewAccountService()
	if err != nil {
		return constant.AccountTypeUnknown, fmt.Errorf("%w,invalid account service", err)
	}
	if as == nil {
		return constant.AccountTypeUnknown, fmt.Errorf("%w,invalid account service", berror.ErrDefault)
	}

	// check uta
	res, err := as.QueryMemberTag(ctx, uid, user.UnifiedTradingTag)
	if err != nil {
		return constant.AccountTypeUnknown, err
	}

	if res == user.UnifiedStateSuccess {
		return constant.AccountTypeUnifiedTrading, nil
	}

	// check uma
	res, err = as.QueryMemberTag(ctx, uid, user.UnifiedMarginTag)
	if err != nil {
		return constant.AccountTypeUnknown, err
	}

	if res == user.UnifiedStateSuccess {
		return constant.AccountTypeUnifiedMargin, nil
	}

	return constant.AccountTypeNormal, nil
}
