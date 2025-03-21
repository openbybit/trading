package diagnosis

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/config_center/etcd"
	"bgw/pkg/registry"
	"bgw/pkg/remoting/redis"
	"code.bydev.io/frameworks/byone/core/discov/nacos"
	n1 "code.bydev.io/frameworks/byone/core/nacos"
	"code.bydev.io/frameworks/byone/core/service"
	redis2 "code.bydev.io/frameworks/byone/core/stores/redis"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
	"context"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestApply(t *testing.T) {
	convey.Convey("apply", t, func() {
		r := apply(func() Result {
			return Result{
				Es: []error{errors.New("xxx")},
			}
		})
		convey.So(r, convey.ShouldNotBeNil)
		convey.So(r.Cost, convey.ShouldNotEqual, "")
		convey.So(r.Errs, convey.ShouldNotBeEmpty)
		convey.So(r.Errs[0], convey.ShouldEqual, "xxx")
	})
}

func TestDiagnoseWithTimeout(t *testing.T) {
	convey.Convey("diagnoseWithTimeout", t, func() {
		r, e := diagnoseWithTimeout(&mockDiagnosis{r: "xxx"})
		convey.So(e, convey.ShouldBeNil)
		convey.So(r, convey.ShouldEqual, "xxx")
	})
}

func TestGetAllInstance(t *testing.T) {
	convey.Convey("getAllInstance", t, func() {
		convey.Convey("err", func() {
			r, e := getAllInstance(context.Background(), "xx a12d=2=1", "public", "DEFAULT_GROUP")
			convey.So(e, convey.ShouldNotBeNil)
			convey.So(r, convey.ShouldBeNil)
		})
		convey.Convey("success", func() {
			r, e := getAllInstance(context.Background(), "user-service-private-up", "bybit-test-1", "DEFAULT_GROUP")
			convey.So(e, convey.ShouldBeNil)
			convey.So(r, convey.ShouldNotBeNil)
			convey.So(len(r), convey.ShouldBeGreaterThan, 0)
		})
	})
}

func TestDial(t *testing.T) {
	convey.Convey("dial", t, func() {
		convey.Convey("err", func() {
			e := Dial(context.Background(), "xxx")
			convey.So(e, convey.ShouldNotBeNil)
		})
		convey.Convey("success", func() {
			s := grpc.NewServer()
			defer s.Stop()
			l, e := net.Listen("tcp", "127.0.0.1:49874")
			defer func(l net.Listener) {
				_ = l.Close()
			}(l)
			convey.So(e, convey.ShouldBeNil)
			go func() {
				_ = s.Serve(l)
			}()
			time.Sleep(time.Second)
			e = Dial(context.Background(), "127.0.0.1:49874")
			convey.So(e, convey.ShouldBeNil)
		})
	})
}

func TestDiagnosisInstances_NotInstanceFound(t *testing.T) {
	convey.Convey("diagnosisInstances", t, func() {
		convey.Convey("not instance", func() {
			es := diagnosisInstances(context.Background(), "12", []registry.ServiceInstance{})
			convey.So(es[0].Error(), convey.ShouldEqual, "12: instance not found")
		})
	})
}

func TestDiagnosisInstances_OneErr(t *testing.T) {
	convey.Convey("diagnosisInstances", t, func() {

		convey.Convey("one err", func() {
			es := diagnosisInstances(context.Background(), "12", []registry.ServiceInstance{
				&registry.DefaultServiceInstance{
					Address: map[string]string{constant.GrpcProtocol: "222sa"},
				},
			})
			convey.So(es[0].Error(), convey.ShouldEqual, "222sa: context deadline exceeded")
		})
	})
}

func TestDiagnoseEtcd(t *testing.T) {
	convey.Convey("DiagnoseEtcd", t, func() {
		p := gomonkey.ApplyFuncReturn(etcd.NewEtcdConfigure, nil, errors.New("xxx"))
		defer p.Reset()
		r := DiagnoseEtcd(context.Background())
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx")

		p.Reset()
		et := &etcd.EtcdConfigure{}
		p.ApplyFuncReturn(etcd.NewEtcdConfigure, et, nil)
		p.ApplyPrivateMethod(reflect.TypeOf(et), "Get", func(ctx context.Context, key string) (string, error) {
			return "", errors.New("xxx")
		})
		r = DiagnoseEtcd(context.Background())
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx")

		p.ApplyPrivateMethod(reflect.TypeOf(et), "Get", func(ctx context.Context, key string) (string, error) {
			return "s", nil
		})
		r = DiagnoseEtcd(context.Background())
		convey.So(len(r.Es), convey.ShouldEqual, 0)
	})
}

func TestDiagnoseRedis(t *testing.T) {
	convey.Convey("DiagnoseRedis", t, func() {
		convey.Convey("redis init failed", func() {
			p := gomonkey.ApplyFuncReturn(redis.NewClient, nil)
			defer p.Reset()

			r := DiagnoseRedis(context.Background())
			convey.So(r.Es[0].Error(), convey.ShouldEqual, "redis init failed")
		})
		convey.Convey("redis error", func() {

			red := &redis2.Redis{}
			p := gomonkey.ApplyFuncReturn(redis.NewClient, red)
			defer p.Reset()

			p.ApplyPrivateMethod(reflect.TypeOf(red), "GetCtx", func(ctx context.Context, key string) (val string, err error) {
				return "", errors.New("xxx")
			})
			r := DiagnoseRedis(context.Background())
			convey.So(r.Es[0].Error(), convey.ShouldEqual, ": xxx")
		})
		convey.Convey("redis success", func() {
			red := &redis2.Redis{}
			p := gomonkey.ApplyFuncReturn(redis.NewClient, red)
			defer p.Reset()

			p.ApplyPrivateMethod(reflect.TypeOf(red), "GetCtx", func(ctx context.Context, key string) (val string, err error) {
				return "123", nil
			})
			r := DiagnoseRedis(context.Background())
			convey.So(len(r.Es), convey.ShouldEqual, 0)
		})
	})
}

func TestDiagnoseGrpcUpstream(t *testing.T) {
	convey.Convey("DiagnoseGrpcUpstream", t, func() {

		// get all instance err
		p := gomonkey.ApplyFuncReturn(getAllInstance, []registry.ServiceInstance{}, errors.New("xxx"))
		defer p.Reset()

		r := DiagnoseGrpcUpstream(context.Background(), "xxx", "123", "456")
		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxx")

		// get all instance success
		p.Reset()
		p.ApplyFuncReturn(getAllInstance, []registry.ServiceInstance{&registry.DefaultServiceInstance{}}, nil)
		p.ApplyFuncReturn(diagnosisInstances, []error{errors.New("xxx2")})
		r = DiagnoseGrpcUpstream(context.Background(), "xxx", "123", "456")
		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx2")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxx2")
	})
}

func TestDiagnoseGrpcDependency_diaStaticAddrErr(t *testing.T) {
	convey.Convey("diaStaticAddrErr", t, func() {
		// dia static address err
		p := gomonkey.ApplyFuncReturn(Dial, errors.New("xxxxx"))
		defer p.Reset()
		r := DiagnoseGrpcDependency(context.Background(), zrpc.RpcClientConf{
			Target: "xxxxx",
		})
		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxxxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxxxx")
	})
}

func TestDiagnoseGrpcDependency_diaStaticAddrSuccess(t *testing.T) {
	convey.Convey("diaStaticAddrSuccess", t, func() {
		p := gomonkey.ApplyFuncReturn(Dial, nil)
		defer p.Reset()
		r := DiagnoseGrpcDependency(context.Background(), zrpc.RpcClientConf{
			Target: "xxxxx",
		})
		convey.So(len(r.Es), convey.ShouldEqual, 0)
		convey.So(len(r.Errs), convey.ShouldEqual, 0)
	})
}

func TestDiagnoseGrpcDependency_getAllInstanceErr(t *testing.T) {
	convey.Convey("getAllInstanceErr", t, func() {

		// get all instance err
		p := gomonkey.ApplyFuncReturn(getAllInstance, []registry.ServiceInstance{}, errors.New("xxx"))
		defer p.Reset()

		r := DiagnoseGrpcDependency(context.Background(), zrpc.RpcClientConf{
			Nacos: nacos.NacosConf{
				NacosConf: n1.NacosConf{
					ServerConfigs: []nacos.ServerConfig{{Address: "xxxx"}},
				},
				Key:   "",
				Group: "",
			},
		})
		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxx")
	})
}

func TestDiagnoseGrpcDependency_getInstanceSuccess(t *testing.T) {
	convey.Convey("getInstanceSuccess", t, func() {
		// get all instance success
		p := gomonkey.ApplyFuncReturn(getAllInstance, []registry.ServiceInstance{&registry.DefaultServiceInstance{}}, nil)
		defer p.Reset()
		p.ApplyFuncReturn(diagnosisInstances, []error{errors.New("xxx2")})
		r := DiagnoseGrpcDependency(context.Background(), zrpc.RpcClientConf{
			Nacos: nacos.NacosConf{
				NacosConf: n1.NacosConf{
					ServerConfigs: []nacos.ServerConfig{{Address: "xxxx"}},
				},
				Key:   "",
				Group: "",
			},
		})
		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxx2")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxx2")
	})
}

func TestDiagnoseKafka_newClientErr(t *testing.T) {
	convey.Convey("newClientErr", t, func() {

		p := gomonkey.ApplyFuncReturn(kafka.NewClient, nil, errors.New("xxxx"))
		defer p.Reset()

		r := DiagnoseKafka(context.Background(), "sss", kafka.UniversalClientConfig{})

		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxxx")
	})
}

func TestDiagnoseKafka_partitionsErr(t *testing.T) {
	convey.Convey("partitionsErr", t, func() {
		c := &mockKafka{
			partitionErr: errors.New("xxxx"),
		}
		p := gomonkey.ApplyFuncReturn(kafka.NewClient, c, nil)
		defer p.Reset()

		r := DiagnoseKafka(context.Background(), "sss", kafka.UniversalClientConfig{})

		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxxx")
	})
}

func TestDiagnoseKafka_newConsumerErr(t *testing.T) {
	convey.Convey("newConsumerErr", t, func() {
		c := &mockKafka{
			newConsumerErr: errors.New("xxxx"),
		}
		p := gomonkey.ApplyFuncReturn(kafka.NewClient, c, nil)
		defer p.Reset()

		r := DiagnoseKafka(context.Background(), "sss", kafka.UniversalClientConfig{})

		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxxx")
	})
}

func TestDiagnoseKafka_getNewestOffsetErr(t *testing.T) {
	convey.Convey("getNewestOffsetErr", t, func() {
		c := &mockKafkaConsumer{
			getNewestOffset: errors.New("xxxx"),
		}
		m := &mockKafka{
			consumer: c,
		}
		p := gomonkey.ApplyFuncReturn(kafka.NewClient, m, nil)
		defer p.Reset()

		r := DiagnoseKafka(context.Background(), "sss", kafka.UniversalClientConfig{})

		convey.So(len(r.Es), convey.ShouldEqual, 1)
		convey.So(len(r.Errs), convey.ShouldEqual, 1)
		convey.So(r.Es[0].Error(), convey.ShouldEqual, "xxxx")
		convey.So(r.Errs[0], convey.ShouldEqual, "xxxx")
	})
}

type mockDiagnosis struct {
	r interface{}
	e error
	k string
}

func (m mockDiagnosis) Key() string {
	if m.k == "" {
		return "mock_diagnosis"
	}
	return m.k
}

func (m mockDiagnosis) Diagnose(_ context.Context) (interface{}, error) {
	return m.r, m.e
}

type mockKafka struct {
	closeErr       error
	partitionErr   error
	getOffsetErr   error
	newConsumerErr error
	consumer       *mockKafkaConsumer
}

func (m *mockKafka) NewProducer() (kafka.Producer, error) {
	panic("not support")
}

func (m *mockKafka) NewAsyncProducer() (kafka.AsyncProducer, error) {
	panic("not support")
}

func (m *mockKafka) NewUnsafeProducer(k string) (kafka.UnsafeProducer, error) {
	panic("not support")
}

func (m *mockKafka) NewConsumerGroup(cc kafka.GroupConfig, handler kafka.Handler) (service.Service, error) {
	panic("not support")
}

func (m *mockKafka) NewConsumer() (kafka.Consumer, error) {
	return m.consumer, m.newConsumerErr
}

func (m *mockKafka) NewUnsafeConsumer(k string) (kafka.UnsafeConsumer, error) {
	panic("not support")
}

func (m *mockKafka) GetOffset(topic string, partition int32, time int64) (int64, error) {
	return 100, m.getOffsetErr
}

func (m *mockKafka) Partitions(topic string) ([]int32, error) {
	return []int32{0}, m.partitionErr
}

func (m *mockKafka) Close() error {
	return m.closeErr
}

type mockKafkaConsumer struct {
	closeErr        error
	getNewestOffset error
}

func (m *mockKafkaConsumer) GetNewestOffset(topic string, partition int32) (int64, error) {
	return 123, m.getNewestOffset
}

func (m *mockKafkaConsumer) ConsumePartition(topic string, partition int32, offset int64, handler kafka.ConsumerHandler, errorHandler kafka.ConsumerErrorHandler) (kafka.PartitionConsumer, error) {
	panic("not support")
}

func (m *mockKafkaConsumer) Close() error {
	return m.closeErr
}
