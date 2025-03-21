package masque

//go:generate mockgen -destination=oauth_mock.go -source=oauth.go -package=masque OauthIface

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.bydev.io/frameworks/byone/zrpc"

	"bgw/pkg/common/berror"
	"bgw/pkg/diagnosis"
	"bgw/pkg/service"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/config"

	oauthv1 "code.bydev.io/cht/backend-bj/user-service/buf-user-gen.git/pkg/bybit/oauth/v1"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var (
	defaultOauthService *OauthService
	oauthOnce           sync.Once
)

type OauthService struct {
	client ggrpc.ClientConnInterface
}

// OauthIface is the interface of account service
type OauthIface interface {
	OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error)
}

// GetOAuthService returns the default masque service
func GetOAuthService() (OauthIface, error) {
	var err error
	if defaultOauthService == nil {
		oauthOnce.Do(func() {
			doNewOauthService()
		})
	}
	if defaultOauthService == nil {
		gmetric.IncDefaultError("oauth", "empty_oauth_service")
		return nil, fmt.Errorf("empty oauth service: %w", err)
	}
	return defaultOauthService, nil
}

func doNewOauthService() {
	rpcClient, err := zrpc.NewClient(config.Global.Oauth, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), "dial oauth failed,error=%v", err)
		galert.Error(context.Background(), "dial oauth failed,error=%v", galert.WithField("error", err))
		return
	}
	defaultOauthService = &OauthService{client: rpcClient.Conn()}
	_ = diagnosis.Register(&oauthDiagnose{
		cfg: config.Global.Oauth,
		svc: defaultOauthService,
	})
}

func (o *OauthService) OAuth(ctx context.Context, token string) (*oauthv1.OAuthResponse, error) {
	m := oauthv1.NewOAuthPrivateServiceClient(o.client)

	glog.Debug(ctx, "oauth grpc", glog.String("token", token))

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	oresp, err := m.OAuth(ctx, &oauthv1.OAuthRequest{BearerToken: token})
	if err != nil {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrOauthInvokeFailed, "oauth Auth error", err.Error())
	}
	return oresp, nil
}

type oauthDiagnose struct {
	cfg zrpc.RpcClientConf
	svc *OauthService
}

func (o *oauthDiagnose) Key() string {
	return "oauth-private"
}

func (o *oauthDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	return resp, nil
}
