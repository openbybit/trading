package user

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"git.bybit.com/svc/stub/pkg/svc/common"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc"

	"bgw/pkg/common/constant"
	"bgw/pkg/diagnosis"
	"bgw/pkg/service/user/mock"
)

func TestDiagnosis(t *testing.T) {
	Convey("Us Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		defer p.Reset()

		dig := usDiagnose{}
		So(dig.Key(), ShouldEqual, "user-service-private")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		So(resp, ShouldNotBeNil)
		So(resp["kafka"], ShouldEqual, result)
		So(resp["grpc"], ShouldEqual, result)
		So(err, ShouldBeNil)
	})
}

func TestMemberTags(t *testing.T) {
	Convey("test member tag", t, func() {
		RegisterMemberTag("")
		RegisterMemberTag("tag1")
		RegisterMemberTag("tag1")
		ts := readAllTags()
		So(len(ts), ShouldEqual, 4)
		ts = readMemberTags()
		So(len(ts), ShouldEqual, 2)
	})
}

func TestNewAccountService(t *testing.T) {
	Convey("test NewAccountService", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(a, b string) {})
		defer patch.Reset()
		_, err := NewAccountService()
		So(err, ShouldBeNil)

		patch1 := gomonkey.ApplyFunc(zrpc.NewClient,
			func(c zrpc.RpcClientConf, options ...zrpc.ClientOption) (zrpc.Client, error) {
				return &mockZrpcClient{}, nil
			})
		defer patch1.Reset()
		accountOnce = sync.Once{}
		_, err = NewAccountService()
		So(err, ShouldBeNil)

		temp := accountService
		accountService = nil
		_, err = NewAccountService()
		So(err, ShouldNotBeNil)
		accountService = temp
	})
}

type mockZrpcClient struct{}

func (m *mockZrpcClient) Conn() grpc.ClientConnInterface {
	return &grpc.ClientConn{}
}

func TestAccountService_GetAccountID(t *testing.T) {
	Convey("test get accountID", t, func() {
		patch := gomonkey.ApplyFunc((*AccountService).getAccountID,
			func(*AccountService, context.Context, int64, int32, int32) (int64, error) {
				return 1234, nil
			})
		defer patch.Reset()

		id, err := accountService.GetAccountID(context.Background(), 123, constant.AppTypeFUTURES, 1)
		So(id, ShouldEqual, 123)
		So(err, ShouldBeNil)

		id, err = accountService.GetAccountID(context.Background(), 123, "empty", 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldBeNil)

		id, err = accountService.GetAccountID(context.Background(), 123, "spot", 1)
		So(id, ShouldEqual, 1234)
		So(err, ShouldBeNil)

		id, err = accountService.GetAccountID(context.Background(), 123, "spot", 1)
		So(id, ShouldEqual, 1234)
		So(err, ShouldBeNil)

		ids, errs := accountService.GetBizAccountIDByApps(context.Background(), 123, 1, "spot")
		So(ids[0], ShouldEqual, 1234)
		So(errs[0], ShouldBeNil)
	})
}

func TestAccountService_getAccountID(t *testing.T) {
	Convey("test getAccountID", t, func() {
		id, err := accountService.getAccountID(context.Background(), 123, 1, -1)
		So(id, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		ctrl := gomock.NewController(t)
		mockClient := mock.NewMockAccountInternalClient(ctrl)
		patch := gomonkey.ApplyFunc(user.NewAccountInternalClient, func(grpc.ClientConnInterface) user.AccountInternalClient { return mockClient })
		defer patch.Reset()

		mockClient.EXPECT().GetAccountIDSByMemberID(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock err"))
		id, err = accountService.getAccountID(context.Background(), 123, 1, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		resp := &user.GetAccountByMemberResponse{}
		resp.Error = &common.Error{}
		mockClient.EXPECT().GetAccountIDSByMemberID(gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
		id, err = accountService.getAccountID(context.Background(), 123, 1, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		resp.Error = nil
		id, err = accountService.getAccountID(context.Background(), 123, 1, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		resp.Accounts = []*user.Account{{AccountId: -1}}
		id, err = accountService.getAccountID(context.Background(), 123, 1, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		resp.Accounts = []*user.Account{{AccountId: 2}}
		id, err = accountService.getAccountID(context.Background(), 123, 1, 1)
		So(id, ShouldEqual, 2)
		So(err, ShouldBeNil)

		// GetUnifiedMarginAccountID
		key := fmt.Sprintf("%dmember_tag_%s", 123, UnifiedMarginTag)
		_ = accountService.accountCache.Set([]byte(key), []byte("1111"), 1000)
		id, err = accountService.GetUnifiedMarginAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldBeNil)

		_ = accountService.accountCache.Set([]byte(key), []byte(UnifiedStateSuccess), 1000)
		_ = accountService.accountCache.Del([]byte(fmt.Sprintf("%dunified_margin%d", 123, 1)))
		id, err = accountService.GetUnifiedMarginAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 2)
		So(err, ShouldBeNil)

		id, err = accountService.GetUnifiedMarginAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 2)
		So(err, ShouldBeNil)

		// GetUnifiedTradingAccountID
		key = fmt.Sprintf("%dmember_tag_%s", 123, UnifiedTradingTag)
		_ = accountService.accountCache.Set([]byte(key), []byte("1111"), 1000)
		id, err = accountService.GetUnifiedTradingAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 0)
		So(err, ShouldBeNil)

		_ = accountService.accountCache.Set([]byte(key), []byte(UnifiedStateSuccess), 1000)
		_ = accountService.accountCache.Del([]byte(fmt.Sprintf("%dunified_trading%d", 123, 1)))
		id, err = accountService.GetUnifiedTradingAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 2)
		So(err, ShouldBeNil)

		id, err = accountService.GetUnifiedTradingAccountID(context.Background(), 123, 1)
		So(id, ShouldEqual, 2)
		So(err, ShouldBeNil)
	})
}

func TestAccountService_QueryMemberTag(t *testing.T) {
	Convey("test QueryMemberTag", t, func() {
		tg, err := accountService.QueryMemberTag(context.Background(), -1, "tag")
		So(tg, ShouldEqual, "")
		So(err, ShouldBeNil)

		ctrl := gomock.NewController(t)
		mockClient := mock.NewMockMemberInternalClient(ctrl)
		patch := gomonkey.ApplyFunc(user.NewMemberInternalClient, func(grpc.ClientConnInterface) user.MemberInternalClient { return mockClient })
		defer patch.Reset()

		mockClient.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("mock err"))
		tg, err = accountService.QueryMemberTag(context.Background(), 123, "tag")
		So(tg, ShouldEqual, "")
		So(err, ShouldNotBeNil)

		resp := &user.QueryMemberTagResponse{}
		resp.Error = &common.Error{}
		mockClient.EXPECT().QueryMemberTag(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
		tg, err = accountService.QueryMemberTag(context.Background(), 123, "tag")
		So(tg, ShouldEqual, "")
		So(err, ShouldNotBeNil)

		resp.Error = nil
		resp.Result = &user.MemberTagInfo{
			TagInfo: map[string]string{UnifiedTradingTag: UnifiedStateSuccess},
		}
		tg, err = accountService.QueryMemberTag(context.Background(), 456, UnifiedMarginTag)
		So(tg, ShouldEqual, UnifiedStateSuccess)
		So(err, ShouldBeNil)

	})
}

func TestAccountService_HandleMemberTagMessage(t *testing.T) {
	Convey("test HandleMemberTagMessage", t, func() {
		msg := &gkafka.Message{}
		msg.Value = []byte("1234")
		accountService.HandleMemberTagMessage(context.Background(), msg)

		m := MemberMessage{
			MemberID:      123,
			UpsertTagInfo: map[string]string{UnifiedTradingTag: UnifiedStateSuccess, "tag": "val"},
		}
		RegisterMemberTag("tag")
		d, _ := jsoniter.Marshal(&m)
		msg.Value = d
		accountService.HandleMemberTagMessage(context.Background(), msg)

		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(string, string) {})
		defer patch.Reset()

		m = MemberMessage{
			MemberID:      123,
			UpsertTagInfo: map[string]string{copytradeUpgrade: copytradeUpgradeSuccess},
		}
		d, _ = jsoniter.Marshal(&m)
		msg.Value = d
		accountService.HandleMemberTagMessage(context.Background(), msg)
	})
}

func TestAccountOnErr(t *testing.T) {
	Convey("test accountOnErr", t, func() {
		accountOnErr(&gkafka.ConsumerError{})
	})
}
