package openapi

import (
	"bytes"
	"context"
	"crypto"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/gtd/gopkg/solutions/risksign.git"
	"git.bybit.com/svc/stub/pkg/pb/api/consts/euser"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/coocood/freecache"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"bgw/pkg/common/kafkaconsume"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

type OpenAPIMemberLoginListResponse = user.GetOpenApiMemberLoginListResponse
type OpenAPIMemberLoginResponse = user.GetOpenApiMemberLoginResponse
type MemberLogin = user.MemberLogin

const (
	defaultOpenapiCacheSize   = 256
	openapiCacheExpireSeconds = 120 * 3600
)

var (
	_ OpenAPIServiceIface = &openapiService{}
)

var (
	errPrivateKeyEmpty = errors.New("user service private key is empty")
)

// OpenAPIServiceIface is the interface that provides the OpenAPI methods
type OpenAPIServiceIface interface {
	GetAPIKey(ctx context.Context, apiKey, xOriginFrom string) (*MemberLogin, error)
	VerifyAPIKey(ctx context.Context, apikey, xOriginFrom string) (*MemberLogin, error)
}

var (
	defaultOpenapiService OpenAPIServiceIface
	openapiOnce           sync.Once
)

type openapiService struct {
	client ggrpc.ClientConnInterface
	// kafka cfg
	cache      *freecache.Cache
	privateKey string
}

type Config struct {
	RpcConf    zrpc.RpcClientConf
	KafkaConf  kafka.UniversalClientConfig
	CacheSize  int
	PrivateKey string
}

func Init(conf Config) error {
	if conf.PrivateKey == "" {
		return fmt.Errorf("empty user service private key")
	}

	if conf.CacheSize <= 0 {
		conf.CacheSize = defaultOpenapiCacheSize
	}

	if conf.RpcConf.Nacos.Key == "" {
		conf.RpcConf.Nacos.Key = "user-service-private"
	}

	rpcClient, err := zrpc.NewClient(conf.RpcConf, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), "openapi service new user-service-private client failed, error=%v", err)
		galert.Error(context.Background(), "openapi service new user-service-private client failed", galert.WithField("error", err))
		return err
	}

	svc := &openapiService{
		client:     rpcClient.Conn(),
		cache:      freecache.NewCache(conf.CacheSize * 1024 * 1024),
		privateKey: conf.PrivateKey,
	}
	// start to consume kafka topic of apikey once
	svc.kafkaConsume(conf.KafkaConf)
	defaultOpenapiService = svc
	registerAdmin()
	return nil
}

func SetOpenapiService(s OpenAPIServiceIface) {
	defaultOpenapiService = s
}

// GetOpenapiService creates a new OpenapiService
func GetOpenapiService() (OpenAPIServiceIface, error) {
	var err error
	var privateKey string
	if defaultOpenapiService == nil {
		openapiOnce.Do(func() {
			privateKey, err = getPrivateKey()
			if err != nil {
				ctx := context.Background()
				msg := "GetOpenapiService getPrivateKey error"
				galert.Error(ctx, msg+err.Error())
				glog.Error(ctx, msg, glog.String("error", err.Error()), glog.String("group", config.GetGroup()), glog.String("version", constant.GetAppName()))
				return
			}
			// openapi cache
			size := config.Global.Data.CacheSize.OpenapiCacheSize
			if size < defaultOpenapiCacheSize {
				size = defaultOpenapiCacheSize
			}

			conf := Config{
				RpcConf:    config.Global.UserServicePrivate,
				KafkaConf:  config.Global.KafkaCli,
				CacheSize:  size,
				PrivateKey: privateKey,
			}

			err = Init(conf)
		})
	}
	if defaultOpenapiService == nil {
		gmetric.IncDefaultError("openapi", "empty_openapi_service")
		return nil, fmt.Errorf("empty openapi service: %w", err)
	}
	return defaultOpenapiService, nil
}

// getPrivateKey decrypt private key
func getPrivateKey() (string, error) {
	usersCfg := &config.Global.ComponentConfig.User
	if usersCfg == nil {
		return "", errors.New("user config is nil")
	}
	key := usersCfg.GetOptions("private_key", "")
	return gsechub.Decrypt(key)
}

func (o *openapiService) kafkaConsume(kconf kafka.UniversalClientConfig) {
	ctx := context.Background()
	kafkaconsume.AsyncHandleKafkaMessage(ctx, constant.EventEffectAPIKeyCache, kconf, o.HandleApikeyMessage, apiOnErr)
}

// getAPIKey get member login info from openapi service
func (o *openapiService) getAPIKey(ctx context.Context, key, xOriginFrom string) (*OpenAPIMemberLoginResponse, error) {
	req := &user.GetOpenApiMemberLoginRequest{
		LoginName:   key,
		XOriginFrom: xOriginFrom,
	}
	// if sign is not empty, add signature params to context
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign, err := o.getSign(req, timestamp)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var ctx1 = service.GetContext(ctx)
	if err == nil {
		openApiSignData := map[string]string{
			"app_id":    constant.Name,
			"signature": sign,
			"timestamp": timestamp,
		}
		md := metadata.New(openApiSignData)
		ctx1 = metadata.NewOutgoingContext(ctx1, md)
	}

	var resp *OpenAPIMemberLoginResponse
	if resp, err = user.NewMemberInternalClient(o.client).GetOpenAPIMemberLoginV2(ctx1, req); err != nil {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetOpenAPIMemberLoginV2 error", err.Error(), req.String())
	}
	if resp.Error != nil {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetOpenAPIMemberLoginV2 resp error", resp.Error.String(), req.String())
	}
	return resp, nil
}

// GetAPIKey get member login info by api-key
// check local cache first
func (o *openapiService) GetAPIKey(ctx context.Context, apikey, xOriginFrom string) (*MemberLogin, error) {
	data, err := o.cache.Get([]byte(apikey))
	if err == nil {
		glog.Debug(ctx, "openapi apikey cache hit", glog.String("key", apikey), glog.String("xOriginFrom", xOriginFrom))
		msg := &user.MemberLogin{}
		err = proto.Unmarshal(data, msg)
		if err != nil {
			return nil, err
		}
		glog.Debug(ctx, "openapi apikey cache hit", glog.String("key", apikey), glog.String("xOriginFrom", xOriginFrom),
			glog.Any("msg", msg))
		return msg, nil
	}
	glog.Debug(ctx, "openapi apikey cache not hit", glog.String("key", apikey), glog.String("xOriginFrom", xOriginFrom))

	resp, err := o.getAPIKey(ctx, apikey, xOriginFrom)
	if err != nil {
		return nil, err
	}

	if resp.MemberLogin == nil || resp.MemberLogin.Status != euser.MemberLoginStatus_MEMBER_LOGIN_STATUS_VERIFIED {
		return nil, berror.ErrOpenAPIApiKey
	}

	data, err = proto.Marshal(resp.MemberLogin)
	if err != nil {
		glog.Error(ctx, "openapi apikey cache not hit, Marshal error", glog.String("key", apikey), glog.Any("member_login", resp.MemberLogin))
		return resp.MemberLogin, nil
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
	_ = o.cache.Set([]byte(apikey), data, openapiCacheExpireSeconds+random)
	return resp.MemberLogin, nil
}

// getSign generate signature with request user api
func (o *openapiService) getSign(req *user.GetOpenApiMemberLoginRequest, timestamp string) (string, error) {
	payload, err := proto.Marshal(req)
	if err != nil {
		return "", err
	}
	if o.privateKey == "" {
		return "", errPrivateKeyEmpty
	}
	appInfo := strings.Join([]string{constant.Name, timestamp}, "")
	signSVC, err := risksign.New(o.privateKey, crypto.SHA256)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.Write(payload)
	buf.Write([]byte(appInfo))
	sign, err := signSVC.Rsa2Sign(buf.String())
	if err != nil {
		return "", err
	}
	return sign, nil
}

func apiOnErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "api key consumer err "+err.Error())
	}
}

type ApiKeyMessage struct {
	Operation int32    `json:"operation"`      // 操作类型
	Source    string   `json:"source"`         // 来源 用于日志记录
	MemberID  int64    `json:"member_id"`      // 用户id
	Keys      []string `json:"keys,omitempty"` // 数组参数(可选)
	Time      int64    `json:"time"`           // 通知时间 用于日志记录
}

func (o *openapiService) HandleApikeyMessage(ctx context.Context, msg *gkafka.Message) {
	var apiKeyMsg ApiKeyMessage
	if err := util.JsonUnmarshal(msg.Value, &apiKeyMsg); err != nil {
		glog.Error(ctx, "HandleApikeyMessage Unmarshal error", glog.String("error", err.Error()))
		return
	}
	glog.Info(ctx, "apikey msg", glog.Any("keys", apiKeyMsg), glog.Int64("offset", msg.Offset))

	for _, key := range apiKeyMsg.Keys {
		glog.Debug(ctx, "del apikey msg", glog.String("key", key))
		o.cache.Del([]byte(key))
	}
}
