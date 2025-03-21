package smp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"bgw/pkg/diagnosis"
	"bgw/pkg/discovery"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	_ "code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/kafka"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestDiagnosis(t *testing.T) {
	convey.Convey("smp Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcUpstream, result)
		defer p.Reset()

		dig := diagnose{}
		convey.So(dig.Key(), convey.ShouldEqual, impServer)
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp["kafka"], convey.ShouldEqual, result)
		convey.So(resp["grpc"], convey.ShouldEqual, result)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestNewDiscovery(t *testing.T) {
	convey.Convey("newDiscovery", t, func() {
		url, _ := common.NewURL(impServer,
			common.WithProtocol(constant.NacosProtocol),
			common.WithGroup(constant.DEFAULT_GROUP),
			common.WithNamespace(config.GetRegistryNamespace()),
		)
		g := newDiscovery(context.Background(), url)
		convey.So(g, convey.ShouldNotBeNil)

		addrs := g(context.Background(), impServer, config.GetRegistryNamespace(), constant.DEFAULT_GROUP)
		sr := discovery.NewServiceRegistry(context.Background())
		ins := sr.GetInstances(url)
		convey.So(len(addrs), convey.ShouldEqual, len(ins))
	})
}

func TestConsumeKafka(t *testing.T) {
	gmetric.Init("TestConsumeKafka")
	convey.Convey("on err", t, func() {
		// 没有返回无法assert
		onErr(&kafka.ConsumerError{})
	})
	convey.Convey("on handleSmpMsg", t, func() {
		g, err := GetGrouper(context.Background())
		convey.So(g, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldBeNil)

		p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(g), "HandleMsg", func(msg []byte) error {
			return errors.New("xxx")
		})
		defer p.Reset()
		// 没有返回无法assert
		handleSmpMsg(context.Background(), &gkafka.Message{})
	})
}

func TestGetGrouper(t *testing.T) {
	gmetric.Init("TestGetGrouper")
	convey.Convey("GetGrouper", t, func() {
		g, err := GetGrouper(context.Background())
		convey.So(g, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestOnSmpAdmin(t *testing.T) {
	gmetric.Init("TestOnSmpAdmin")
	convey.Convey("OnSmpAdmin", t, func() {
		g, err := GetGrouper(context.Background())
		convey.So(g, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldBeNil)

		p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(g), "GetGroup", func(ctx context.Context, uid int64) (group int32, err error) {
			return 1, errors.New("xxx")
		})
		defer p.Reset()
		r, err := OnSmpAdmin(gapp.AdminArgs{Params: []string{"123"}})
		convey.So(r, convey.ShouldBeNil)
		convey.So(err, convey.ShouldNotBeNil)

		p.ApplyPrivateMethod(reflect.TypeOf(g), "GetGroup", func(ctx context.Context, uid int64) (group int32, err error) {
			return 1, nil
		})
		r, err = OnSmpAdmin(gapp.AdminArgs{Params: []string{"123"}})
		convey.So(r, convey.ShouldNotBeNil)
		convey.So(r.(resp).GroupID, convey.ShouldEqual, 1)
		convey.So(err, convey.ShouldBeNil)
	})
}
