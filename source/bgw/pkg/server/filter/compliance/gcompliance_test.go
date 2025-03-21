package compliance

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/agiledragon/gomonkey/v2"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var mockErr = errors.New("mock err")

func TestMsgHandler(t *testing.T) {
	Convey("test msg handler", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().HandleUserWhiteListEvent(gomock.Any()).Return(mockErr)
		mockWall.EXPECT().HandleUserKycEvent(gomock.Any()).Return(mockErr)
		mockWall.EXPECT().HandleStrategyEvent(gomock.Any()).Return(mockErr)
		mockWall.EXPECT().HandleSiteConfigEvent(gomock.Any()).Return(mockErr)

		gw = mockWall

		msg := &gkafka.Message{}
		handleWhiteMsg(context.Background(), msg)
		handleUserKycMsg(context.Background(), msg)
		handleStrategyMSg(context.Background(), msg)
		handleSiteConfigMsg(context.Background(), msg)

		kafkaErr := &gkafka.ConsumerError{}
		onErr(kafkaErr)
	})
}

type mockRes struct{}

func (m *mockRes) GetEndPointExec() string {
	return ""
}

func (m *mockRes) GetEndPointsArgs() gcompliance.EndpointsArgs {
	return gcompliance.EndpointsArgs{}
}

func TestMarshalComplianceResult(t *testing.T) {
	Convey("test marshalComplianceResult", t, func() {
		_, err := marshalComplianceResult(&mockRes{})
		So(err, ShouldBeNil)
	})
}

func TestInitComplianceService(t *testing.T) {
	Convey("test init compliance service", t, func() {

		err := initComplianceService()
		So(err, ShouldBeNil)

		p := gomonkey.ApplyFuncReturn(zrpc.NewClient, nil, errors.New("asas"))
		defer p.Reset()
		err = initComplianceService()
		So(err, ShouldNotBeNil)
	})
}

func TestDiagnosis(t *testing.T) {
	Convey("Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		defer p.Reset()
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)

		dig := diagnose{}
		So(dig.Key(), ShouldEqual, "compliance_wall")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		So(resp, ShouldNotBeNil)
		So(resp["kafka_whitelist_topic"], ShouldEqual, result)
		So(resp["kafka_kyc_type_topic"], ShouldEqual, result)
		So(resp["kafka_strategy_topic"], ShouldEqual, result)
		So(resp["kafka_site_topic"], ShouldEqual, result)
		So(resp["grpc"], ShouldEqual, result)
		So(err, ShouldBeNil)
	})
}
