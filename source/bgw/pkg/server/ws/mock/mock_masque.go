package mock

import (
	"context"

	"bgw/pkg/service/masque"
	"git.bybit.com/svc/stub/pkg/svc/common"
)

type Masq struct {
	Err    *common.Error
	Uid    int64
	RpcErr error
}

func (m *Masq) MasqueTokenInvoke(_ context.Context, platform, token, originUrl string, typ masque.MasqueType) (*masque.AuthResponse, error) {
	return &masque.AuthResponse{
		Error:  m.Err,
		UserId: m.Uid,
	}, m.RpcErr
}
