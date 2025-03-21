package mock

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"git.bybit.com/svc/stub/pkg/pb/api/user"

	"bgw/pkg/service/openapi"
)

type User struct {
	MemberId    int64
	LoginSecret string
	Err         error
}

func (u *User) GetAPIKey(ctx context.Context, apiKey, xOriginFrom string) (*openapi.MemberLogin, error) {
	return &openapi.MemberLogin{
		MemberId:    u.MemberId,
		LoginSecret: u.LoginSecret,
		ExtInfo: &user.MemberLoginExt{
			Flag: string(sign.TypeHmac),
		}}, nil
}

func (u *User) VerifyAPIKey(ctx context.Context, apikey, xOriginFrom string) (*openapi.MemberLogin, error) {
	return &openapi.MemberLogin{
		MemberId:    u.MemberId,
		LoginSecret: u.LoginSecret,
		ExtInfo: &user.MemberLoginExt{
			Flag: string(sign.TypeHmac),
		}}, u.Err
}
