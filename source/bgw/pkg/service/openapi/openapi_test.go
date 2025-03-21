package openapi

import (
	"context"
	"errors"
	"sync"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/consts/euser"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"git.bybit.com/svc/stub/pkg/svc/common"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/coocood/freecache"
	"github.com/golang/mock/gomock"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc"

	"bgw/pkg/config"
	"bgw/pkg/service/openapi/mock"
)

func Test_Init(t *testing.T) {
	Convey("test init", t, func() {
		SetOpenapiService(nil)
		cfg := Config{}
		err := Init(cfg)
		So(err, ShouldNotBeNil)

		cfg.PrivateKey = "123"
		cfg.CacheSize = 0
		cfg.RpcConf = config.Global.UserServicePrivate
		cfg.KafkaConf = config.Global.KafkaCli
		err = Init(cfg)
		So(err, ShouldBeNil)

		patch := gomonkey.ApplyFunc(zrpc.NewClient, func(c zrpc.RpcClientConf, options ...zrpc.ClientOption) (zrpc.Client, error) {
			return &mockCli{}, nil
		})
		defer patch.Reset()
		err = Init(cfg)
		So(err, ShouldBeNil)
	})
}

func Test_GetOpenapiService(t *testing.T) {
	Convey("test GetOpenapiService", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()

		SetOpenapiService(nil)
		os, err := GetOpenapiService()
		So(os, ShouldNotBeNil)
		So(err, ShouldBeNil)

		patch1 := gomonkey.ApplyFunc(getPrivateKey, func() (string, error) { return "key", nil })
		defer patch1.Reset()

		openapiOnce = sync.Once{}
		SetOpenapiService(nil)
		os, err = GetOpenapiService()
		So(os, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

type mockCli struct{}

func (m *mockCli) Conn() grpc.ClientConnInterface {
	return nil
}

func TestOpenapiService_getAPIKey(t *testing.T) {
	Convey("test getAPIKey", t, func() {
		ctrl := gomock.NewController(t)
		mockClient := mock.NewMockMemberInternalClient(ctrl)
		patch := gomonkey.ApplyFunc(user.NewMemberInternalClient, func(grpc.ClientConnInterface) user.MemberInternalClient { return mockClient })
		defer patch.Reset()
		mockClient.EXPECT().GetOpenAPIMemberLoginV2(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock err"))

		os := &openapiService{}
		res, err := os.getAPIKey(context.Background(), "key", "xoriginfrom")
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)

		patch1 := gomonkey.ApplyFunc((*openapiService).getSign, func(*openapiService, *user.GetOpenApiMemberLoginRequest, string) (string, error) {
			return "sign", err
		})
		defer patch1.Reset()

		resp := &OpenAPIMemberLoginResponse{}
		mockClient.EXPECT().GetOpenAPIMemberLoginV2(gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
		res, err = os.getAPIKey(context.Background(), "key", "xoriginfrom")
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)

		resp.Error = &common.Error{}
		res, err = os.getAPIKey(context.Background(), "key", "xoriginfrom")
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func TestOpenapiService_GetAPIKey(t *testing.T) {
	Convey("test GetAPIKey", t, func() {
		os := &openapiService{
			cache: freecache.NewCache(1000),
		}
		patch := gomonkey.ApplyFunc((*openapiService).getAPIKey, func(o *openapiService, c context.Context, key string, from string) (*OpenAPIMemberLoginResponse, error) {
			if key == "apikey" {
				return nil, errors.New("mock err")
			}
			if key == "apikey259" {
				return &OpenAPIMemberLoginResponse{}, nil
			}
			if key == "apikey345" {
				return &OpenAPIMemberLoginResponse{
					MemberLogin: &user.MemberLogin{Status: euser.MemberLoginStatus_MEMBER_LOGIN_STATUS_VERIFIED},
				}, nil
			}
			return nil, nil
		})
		defer patch.Reset()

		_, err := os.GetAPIKey(context.Background(), "apikey", "from")
		So(err, ShouldNotBeNil)

		_, err = os.GetAPIKey(context.Background(), "apikey259", "from")
		So(err, ShouldNotBeNil)

		_, err = os.GetAPIKey(context.Background(), "apikey345", "from")
		So(err, ShouldBeNil)
	})
}

func TestOpenapiService_getSign(t *testing.T) {
	Convey("test getSign", t, func() {
		os := &openapiService{
			privateKey: "1234",
		}

		_, err := os.getSign(&user.GetOpenApiMemberLoginRequest{}, "123435")
		So(err, ShouldNotBeNil)
	})
}

func Test_apiOnErr(t *testing.T) {
	Convey("test apiOnErr", t, func() {
		apiOnErr(&gkafka.ConsumerError{})
	})
}

func TestOpenapiService_HandleApikeyMessage(t *testing.T) {
	Convey("test HandleApikeyMessage", t, func() {
		os := &openapiService{
			cache: freecache.NewCache(100),
		}
		msg := &gkafka.Message{}
		os.HandleApikeyMessage(context.Background(), msg)

		val, _ := jsoniter.Marshal(ApiKeyMessage{Keys: []string{"key1"}})
		msg.Value = val
		os.HandleApikeyMessage(context.Background(), msg)
	})
}
