package masque

//go:generate mockgen -destination=masque_mock.go -source=masque.go -package=user MasqueIface

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/mod/pkg/bplatform"
	"git.bybit.com/svc/stub/pkg/svc/cmd"
	"git.bybit.com/svc/stub/pkg/svc/masquerade"
	"google.golang.org/grpc/metadata"

	"bgw/pkg/common/berror"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

const (
	masqueGatewayKey   = "gateway"
	masqueGatewayVal   = "bgw"
	masqueWSGatewayVal = "bgws"
	masquePathKey      = "origin-url"
)

// MasqueType is masque type
type MasqueType = string

// nolint
const (
	Auth         MasqueType = "auth"
	RefreshToken MasqueType = "refreshToken"
	WeakAuth     MasqueType = "weakAuth"
)

var (
	errInvalidMasqConfig = errors.New("remote config of masque error")
	errInvalidMasqType   = errors.New("invalid type of masque error")
)

var (
	defaultMasqService MasqueIface
	masqOnce           sync.Once
)

// Plateform is the masquerade platform
type Plateform = masquerade.Platform

// AuthResponse is the response of auth
type AuthResponse = masquerade.AuthResponse

// MasqueService is a service for masque
type MasqueService struct {
	client ggrpc.ClientConnInterface
}

// MasqueIface is the interface of masque service
type MasqueIface interface {
	MasqueTokenInvoke(ctx context.Context, platform, token, originUrl string, typ MasqueType) (*AuthResponse, error)
}

type Config struct {
	RpcConf zrpc.RpcClientConf
}

func Init(conf Config) error {
	if conf.RpcConf.Nacos.Key == "" {
		conf.RpcConf.Nacos.Key = "masq"
	}

	rpcClient, err := zrpc.NewClient(conf.RpcConf, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), "new masq client fail, error=%v", err)
		galert.Error(context.Background(), "new masq client fail", galert.WithField("error", err))
		return err
	}
	defaultMasqService = &MasqueService{client: rpcClient.Conn()}
	_ = diagnosis.Register(&masqDiagnose{
		cfg: config.Global.Masq,
		svc: defaultMasqService.(*MasqueService),
	})
	return nil
}

func SetMasqueService(s MasqueIface) {
	defaultMasqService = s
}

// GetMasqueService returns the default masque service
func GetMasqueService() (MasqueIface, error) {
	var err error
	if defaultMasqService == nil {
		masqOnce.Do(func() {
			conf := Config{
				RpcConf: config.Global.Masq,
			}
			err = Init(conf)
		})
	}
	if defaultMasqService == nil {
		gmetric.IncDefaultError("masque", "empty_masq_service")
		return nil, fmt.Errorf("empty masq service: %w", err)
	}
	return defaultMasqService, nil
}

// MasqueTokenInvoke invokes masque token
func (ms *MasqueService) MasqueTokenInvoke(ctx context.Context, platform, token, originUrl string, t MasqueType) (*AuthResponse, error) {
	m := masquerade.NewMasqueradeClient(ms.client)

	var err error
	var resp *AuthResponse

	req := &masquerade.TokenRequest{
		Platform: masqplatf(bplatform.Client(platform)),
		Token:    token,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	gw := masqueGatewayVal
	if platform == "" {
		gw = masqueWSGatewayVal
	}
	ctx = metadata.AppendToOutgoingContext(ctx, masqueGatewayKey, gw, masquePathKey, originUrl)
	switch t {
	case Auth:
		if resp, err = m.Auth(ctx, req); err != nil {
			return nil, berror.NewUpStreamErr(berror.UpstreamErrMasqInvokeFailed, "masq Auth error", err.Error())
		}
	case RefreshToken:
		if resp, err = m.RefreshToken(ctx, req); err != nil {
			return nil, berror.NewUpStreamErr(berror.UpstreamErrMasqInvokeFailed, "masq RefreshToken error", err.Error())
		}
	case WeakAuth:
		if resp, err = m.WeakAuth(ctx, req); err != nil {
			return nil, berror.NewUpStreamErr(berror.UpstreamErrMasqInvokeFailed, "masq WeakAuth error", err.Error())
		}
	default:
		return nil, errInvalidMasqType
	}
	return resp, nil
}

func masqplatf(client bplatform.Client) Plateform {
	// 虽然两者是二进制兼容的，这里防一手其他业务/设计上不可能的值钻进来 internal crm等
	switch client.CMDPlatform() {
	case cmd.Platform_PLATFORM_H5:
		return masquerade.Platform_PLATFORM_H5
	case cmd.Platform_PLATFORM_PCWEB:
		return masquerade.Platform_PLATFORM_PCWEB
	case cmd.Platform_PLATFORM_APP:
		return masquerade.Platform_PLATFORM_APP
	default:
		return masquerade.Platform_PLATFORM_UNSPECIFIED
	}
}

type masqDiagnose struct {
	cfg zrpc.RpcClientConf
	svc *MasqueService
}

func (ms *masqDiagnose) Key() string {
	return "masq"
}

func (ms *masqDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, ms.cfg)
	return resp, nil
}
