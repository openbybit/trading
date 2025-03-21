package masque

import (
	"context"
	"errors"
	"testing"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/mod/pkg/bplatform"
	"git.bybit.com/svc/stub/pkg/svc/masquerade"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
)

func TestSetMasqueService(t *testing.T) {
	d := defaultMasqService
	SetMasqueService(d)
	assert.Equal(t, d, defaultMasqService)
}

func TestMasqplatf(t *testing.T) {
	assert.Equal(t, masquerade.Platform_PLATFORM_H5, masqplatf(bplatform.H5))
	assert.Equal(t, masquerade.Platform_PLATFORM_PCWEB, masqplatf(bplatform.PCWeb))
	assert.Equal(t, masquerade.Platform_PLATFORM_APP, masqplatf(bplatform.AndroidAPP))
	assert.Equal(t, masquerade.Platform_PLATFORM_UNSPECIFIED, masqplatf("asas"))
}

func TestGetMasqueService(t *testing.T) {
	convey.Convey("TestGetMasqueService", t, func() {
		service, err := GetMasqueService()
		convey.So(err, convey.ShouldBeNil)
		convey.So(service, convey.ShouldNotBeNil)

		p := gomonkey.ApplyFuncReturn(zrpc.NewClient, nil, errors.New("asas"))
		defer p.Reset()

		defaultMasqService = nil
		Init(Config{})
		convey.So(defaultMasqService, convey.ShouldBeNil)
		defaultMasqService = service.(*MasqueService)
	})
}

func TestMasqueTokenInvoke(t *testing.T) {
	convey.Convey("TestMasqueTokenInvoke", t, func() {

		service, err := GetMasqueService()
		convey.So(err, convey.ShouldBeNil)
		convey.So(service, convey.ShouldNotBeNil)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		masqueradeClient := NewMockMasqueradeClient(ctrl)
		masqueradeClient.EXPECT().Auth(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock auth error"))

		applyFunc := gomonkey.ApplyFunc(masquerade.NewMasqueradeClient, func(cc ggrpc.ClientConnInterface) masquerade.MasqueradeClient {
			return masqueradeClient
		})
		defer applyFunc.Reset()

		invoke, err := service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", Auth)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		masqueradeClient.EXPECT().Auth(gomock.Any(), gomock.Any()).Return(nil, nil)
		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", Auth)
		convey.So(err, convey.ShouldBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		masqueradeClient.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock refreshToken error"))
		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", RefreshToken)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		masqueradeClient.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(nil, nil)
		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", RefreshToken)
		convey.So(err, convey.ShouldBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		masqueradeClient.EXPECT().WeakAuth(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock weak error"))
		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", WeakAuth)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		masqueradeClient.EXPECT().WeakAuth(gomock.Any(), gomock.Any()).Return(nil, nil)
		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", WeakAuth)
		convey.So(err, convey.ShouldBeNil)
		convey.So(invoke, convey.ShouldBeNil)

		invoke, err = service.MasqueTokenInvoke(context.Background(), string(bplatform.PCWeb), "token", "originUrl", "invalidType")
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldEqual, errInvalidMasqType)
		convey.So(invoke, convey.ShouldBeNil)
	})
}

func TestDiagnosis(t *testing.T) {
	convey.Convey("masq Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		defer p.Reset()

		dig := masqDiagnose{}
		convey.So(dig.Key(), convey.ShouldEqual, "masq")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp["grpc"], convey.ShouldEqual, result)
		convey.So(err, convey.ShouldBeNil)
	})
}
