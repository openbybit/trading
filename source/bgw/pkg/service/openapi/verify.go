package openapi

import (
	"context"
	"time"

	"bgw/pkg/common/berror"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func (o *openapiService) VerifyAPIKey(ctx context.Context, apikey, xOriginFrom string) (member *MemberLogin, err error) {
	member, err = o.GetAPIKey(ctx, apikey, xOriginFrom)
	if err != nil {
		return
	}

	if err = o.checkExpired(member); err != nil {
		return
	}
	glog.Debug(ctx, "VerifyAPIKey", glog.Any("memberLogin.MemberId", member.MemberId),
		glog.Any("memberLogin.LoginSecret", member.LoginSecret),
		glog.Any("memberLogin.ExtInfo.Permissions", member.ExtInfo.Permissions),
		glog.Any("limits", member.ExtInfo.Limits),
	)
	return member, nil
}

func (o *openapiService) checkExpired(member *MemberLogin) error {
	if member.GetExtInfo().ExpiredTimeE0 > 0 && member.GetExtInfo().ExpiredTimeE0 <= time.Now().Unix() {
		return berror.ErrOpenAPIApiKeyExpire
	}
	return nil
}
