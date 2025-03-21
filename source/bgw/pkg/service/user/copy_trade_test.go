package user

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

	"bgw/pkg/diagnosis"
	"bgw/pkg/service/user/mock"
)

func TestCpDiagnosis(t *testing.T) {
	Convey("Cp Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		defer p.Reset()

		dig := cpDiagnose{}
		So(dig.Key(), ShouldEqual, "copytrade")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		So(resp, ShouldNotBeNil)
		So(resp["kafka"], ShouldEqual, result)
		So(resp["grpc"], ShouldEqual, result)
		So(err, ShouldBeNil)
	})
}

func TestCopyTradeInfo_Parse(t *testing.T) {
	Convey("test Parse", t, func() {
		ci := CopyTradeInfo{}
		res, err := ci.Parse(context.Background(), "")
		So(res, ShouldBeNil)
		So(err, ShouldBeNil)

		res, err = ci.Parse(context.Background(), "111")
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)

		res, err = ci.Parse(context.Background(), `{"allowGuest": true}`)
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestGetCopyTradeService(t *testing.T) {
	Convey("test GetCopyTradeService", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(a, b string) {})
		defer patch.Reset()
		_, err := GetCopyTradeService(zrpc.RpcClientConf{})
		So(err, ShouldBeNil)

		copyTradeOnce = sync.Once{}
		patch1 := gomonkey.ApplyFunc(zrpc.NewClient,
			func(c zrpc.RpcClientConf, options ...zrpc.ClientOption) (zrpc.Client, error) {
				return &mockZrpcClient{}, nil
			})
		defer patch1.Reset()
		_, err = GetCopyTradeService(zrpc.RpcClientConf{})
		So(err, ShouldBeNil)

		copyTradeService = nil
		_, err = GetCopyTradeService(zrpc.RpcClientConf{})
		So(err, ShouldNotBeNil)
	})
}

func TestCopyTradeService_getCopyTradeData(t *testing.T) {
	Convey("test getCopyTradeData", t, func() {
		cs := &CopyTradeService{
			copytradeCache: freecache.NewCache(100 * 1024),
		}
		res, ok := cs.getCopyTradeData(345)
		So(res, ShouldBeNil)
		So(ok, ShouldBeFalse)

		key := cs.getCopyTradeCacheKey(345)
		_ = cs.copytradeCache.Set([]byte(key), []byte("wrong val"), 1000)

		res, ok = cs.getCopyTradeData(345)
		So(res, ShouldNotBeNil)
		So(ok, ShouldBeFalse)

		err := cs.setCopyTradeData(345, nil)
		So(err, ShouldBeNil)

		err = cs.setCopyTradeData(345, &CopyTrade{LeaderID: 111111})
		So(err, ShouldBeNil)

		res, ok = cs.getCopyTradeData(345)
		So(res, ShouldNotBeNil)
		So(ok, ShouldBeTrue)

		res, err = cs.GetCopyTradeData(context.Background(), 345)
		So(res, ShouldNotBeNil)
		So(ok, ShouldBeTrue)
	})
}

func TestCopyTradeService_queryCopyTradeData(t *testing.T) {
	Convey("test queryCopyTradeData", t, func() {
		cs := &CopyTradeService{
			copytradeCache: freecache.NewCache(100 * 1024),
		}
		ctrl := gomock.NewController(t)
		mockCli := mock.NewMockMemberInternalClient(ctrl)
		patch := gomonkey.ApplyFunc(user.NewMemberInternalClient,
			func(grpc.ClientConnInterface) user.MemberInternalClient {
				return mockCli
			})
		defer patch.Reset()

		mockCli.EXPECT().GetRelationByMemberIDCommon(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock err")).Times(2)
		res, err := cs.queryCopyTradeData(context.Background(), 123)
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)

		res, err = cs.GetCopyTradeData(context.Background(), 123)
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)

		resp := &user.QueryRelationByMemberResponse{}
		resp.Error = &common.Error{}
		mockCli.EXPECT().GetRelationByMemberIDCommon(gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
		res, err = cs.queryCopyTradeData(context.Background(), 123)
		So(res, ShouldBeNil)
		So(err, ShouldNotBeNil)

		resp.Error = nil
		res, err = cs.queryCopyTradeData(context.Background(), 123)
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)

		res, err = cs.GetCopyTradeData(context.Background(), 123)
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)

		resp.Result = map[int64]*user.MemberRelationList{123: &user.MemberRelationList{}}
		res, err = cs.queryCopyTradeData(context.Background(), 123)
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)

		rl1 := &user.MemberRelation{
			OwnerMemberId:  3456,
			TargetMemberId: 2345,
		}
		rl2 := &user.MemberRelation{
			MemberId:           123,
			MemberRelationType: euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_LEADER,
		}
		rl3 := &user.MemberRelation{
			MemberId:           123,
			MemberRelationType: euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_FOLLOWER,
		}
		rl4 := &user.MemberRelation{
			MemberId:           3456,
			MemberRelationType: euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_LEADER,
		}
		rls := &user.MemberRelationList{
			MemberRelations: []*user.MemberRelation{rl1, rl2, rl3, rl4},
		}
		resp.Result = map[int64]*user.MemberRelationList{123: rls}
		res, err = cs.queryCopyTradeData(context.Background(), 123)
		So(res, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestCopyTradeService_handleCopyTradeData(t *testing.T) {
	Convey("test handleCopyTradeData", t, func() {
		cs := &CopyTradeService{
			copytradeCache: freecache.NewCache(1024),
		}
		cs.handleCopyTradeData(context.Background(), &gkafka.Message{})

		d, _ := jsoniter.Marshal(SpecialSubMemberCreateMsg{})
		cs.handleCopyTradeData(context.Background(), &gkafka.Message{Value: d})

		d, _ = jsoniter.Marshal(SpecialSubMemberCreateMsg{MemberRelationType: 4})
		cs.handleCopyTradeData(context.Background(), &gkafka.Message{Value: d})
	})
}

func TestOnErr(t *testing.T) {
	Convey("test onErr", t, func() {
		onErr(&gkafka.ConsumerError{})
	})
}
