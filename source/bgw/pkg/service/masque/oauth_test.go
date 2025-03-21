package masque

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/frameworks/byone/zrpc"

	"bgw/pkg/diagnosis"

	"github.com/smartystreets/goconvey/convey"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/golang/mock/gomock"

	oauthv1 "code.bydev.io/cht/backend-bj/user-service/buf-user-gen.git/pkg/bybit/oauth/v1"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	// 导入生成的模拟包
)

func TestGetOAuthService(t *testing.T) {
	convey.Convey("TestGetOAuthService", t, func() {
		service, err := GetOAuthService()
		convey.So(err, convey.ShouldBeNil)
		convey.So(service, convey.ShouldNotBeNil)

		p := gomonkey.ApplyFuncReturn(zrpc.NewClient, nil, errors.New("asas"))
		defer p.Reset()

		defaultOauthService = nil
		doNewOauthService()
		convey.So(defaultOauthService, convey.ShouldBeNil)
		defaultOauthService = service.(*OauthService)
	})
}

func TestOAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// 创建模拟对象
	mockConn := pool.NewMockConn(ctrl)
	mockOAuthClient := NewMockOAuthPrivateServiceClient(ctrl)

	// 设置模拟的行为
	conn := &grpc.ClientConn{} // 这里可以根据需要设置模拟的 conn 对象
	mockConn.EXPECT().Client().Return(conn).AnyTimes()
	mockConn.EXPECT().Close().Return(nil).AnyTimes()
	// expectedResp := &oauthv1.OAuthResponse{
	// 	MemberId: 1,
	// 	ExtInfo:  nil,
	// 	Scope:    nil,
	// 	ClientId: "client123",
	// }
	mockOAuthClient.EXPECT().OAuth(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock err")).AnyTimes()
	patch := gomonkey.ApplyFuncReturn(oauthv1.NewOAuthPrivateServiceClient, mockOAuthClient)
	defer patch.Reset()

	oauthService, err := GetOAuthService()

	token := "your-access-token"
	resp, err := oauthService.OAuth(context.Background(), token)

	// 断言结果是否符合预期
	assert.Error(t, err)
	assert.Nil(t, resp)

}

func TestOauthDiagnosis(t *testing.T) {
	convey.Convey("Oauth Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		defer p.Reset()

		dig := oauthDiagnose{}
		convey.So(dig.Key(), convey.ShouldEqual, "oauth-private")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp["grpc"], convey.ShouldEqual, result)
		convey.So(err, convey.ShouldBeNil)
	})
}
