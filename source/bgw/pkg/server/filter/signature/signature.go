package signature

import (
	"context"
	"sync"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	"git.bybit.com/gtd/gopkg/solutions/risksign.git"
	"google.golang.org/protobuf/proto"
)

var once sync.Once

func Init() {
	filter.Register(filter.SignatureFilterKey, new)
}

type signatureKey struct{}

func new() filter.Filter {
	return &signatureKey{}
}

// GetName returns the name of the filter
func (*signatureKey) GetName() string {
	return filter.SignatureFilterKey
}

// Do will call next handler
func (s *signatureKey) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		md := metadata.MDFromContext(ctx)
		signature, err := s.getSignature(ctx, md.Route.Registry)

		if err != nil {
			return berror.NewInterErr("signature filter", err.Error())
		}

		md.Intermediate.RiskSign = signature

		return next(ctx)
	}
}

// Init will init the filter
func (s *signatureKey) Init(ctx context.Context, args ...string) (err error) {
	if len(args) == 0 {
		// skip must filter
		return nil
	}
	once.Do(func() {
		defaultKeyKeeper = newPrivateKeyKeeper()
		err = defaultKeyKeeper.buildListen(ctx)
	})
	return
}

func (s *signatureKey) getSignature(ctx context.Context, appID string) (string, error) {
	appName := defaultKeyKeeper.GetAppName()
	signKey, err := defaultKeyKeeper.GetSignKey(ctx, appID)
	if err != nil {
		return "", err
	}
	info, err := risksign.WithSignInfo(appName, signKey)
	if err != nil {
		return "", err
	}
	marshal, err := proto.Marshal(info)
	if err != nil {
		return "", err
	}

	return string(marshal), nil
}
