package mock

import (
	"context"

	"bgw/pkg/service/ban"
)

type Ban struct {
	LoginBanType int32
	Err          error
}

func (m *Ban) GetMemberStatus(_ context.Context, uid int64) (*ban.UserStatusWrap, error) {
	return &ban.UserStatusWrap{
		LoginBanType: m.LoginBanType,
	}, m.Err
}

func (m *Ban) CheckStatus(_ context.Context, uid int64) (*ban.UserStatusWrap, error) {
	return &ban.UserStatusWrap{}, nil
}

func (m *Ban) VerifyTrade(_ context.Context, uid int64, app string, status *ban.UserStatusWrap, opts ...ban.Option) (bool, error) {
	return false, nil
}
