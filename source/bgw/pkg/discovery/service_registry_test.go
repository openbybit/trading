package discovery

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/container"

	"bgw/pkg/registry/etcd"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"

	"bgw/pkg/registry"

	"github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
)

func setupTest(t *testing.T) (*MockServiceDiscovery, *gomock.Controller, *gomonkey.Patches, func()) {
	controller := gomock.NewController(t)
	discovery := NewMockServiceDiscovery(controller)
	patches := gomonkey.NewPatches()

	ServiceRegistryModule := &serviceRegistry{
		ctx:          context.Background(),
		serviceNames: container.NewSet(),
	}
	patch := patches.ApplyPrivateMethod(reflect.TypeOf(ServiceRegistryModule), "getRegistry", func(*serviceRegistry, *common.URL) registry.ServiceDiscovery {
		return discovery
	})

	teardown := func() {
		patch.Reset()
		controller.Finish()
	}

	return discovery, controller, patches, teardown
}

func TestGetRegistry(t *testing.T) {
	convey.Convey("TestGetRegistry", t, func() {
		sr := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}

		url, _ := common.NewURL("TestGetRegistry",
			common.WithProtocol(constant.DNSProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)
		convey.Convey("TestGetRegistry by dns", func() {
			serviceDiscovery := sr.getRegistry(url)
			convey.So(serviceDiscovery, convey.ShouldNotBeNil)
		})

		convey.Convey("TestGetRegistry by etcd", func() {
			url.Protocol = constant.EtcdProtocol
			serviceDiscovery := sr.getRegistry(url)
			convey.So(serviceDiscovery, convey.ShouldNotBeNil)

			applyFunc := gomonkey.ApplyFunc(etcd.NewETCDServiceDiscovery, func(ctx context.Context, root string) (registry.ServiceDiscovery, error) {
				return nil, errors.New("mock error")
			})
			defer applyFunc.Reset()
			serviceDiscovery = sr.getRegistry(url)
			convey.So(serviceDiscovery, convey.ShouldBeNil)
		})

	})
}

func TestNewServiceRegistry(t *testing.T) {
	convey.Convey("TestNewServiceRegistry", t, func() {
		serviceRegistry := NewServiceRegistry(context.Background())
		convey.So(serviceRegistry, convey.ShouldNotBeNil)
	})
}

func TestWatch(t *testing.T) {
	convey.Convey("TestWatch", t, func() {
		serviceRegistryModule := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		err := serviceRegistryModule.Watch(context.Background(), nil)
		convey.So(err, convey.ShouldNotBeNil)

		url, err := common.NewURL("open-contract-core-test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)
		convey.So(err, convey.ShouldBeNil)
		convey.So(url, convey.ShouldNotBeNil)

		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)

		err = serviceRegistryModule.Watch(context.Background(), url)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestServices(t *testing.T) {
	convey.Convey("TestServices", t, func() {
		serviceRegistryModule := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		convey.So(serviceRegistryModule, convey.ShouldNotBeNil)

		services := serviceRegistryModule.Services()
		convey.So(services, convey.ShouldNotBeNil)
		convey.So(len(services), convey.ShouldEqual, 0)

		url, _ := common.NewURL("open-contract-core-test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)
		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)

		err := serviceRegistryModule.Watch(context.Background(), url)
		convey.So(err, convey.ShouldBeNil)

		services = serviceRegistryModule.Services()
		convey.So(len(services), convey.ShouldEqual, 1)
		convey.So(services[0], convey.ShouldEqual, registry.ServiceMeta{Name: "open-contract-core-test", Namespace: "unify-test-1", Group: "DEFAULT_GROUP"})
	})
}

func TestGetInstances(t *testing.T) {
	convey.Convey("TestGetInstances", t, func() {
		var url *common.URL
		serviceRegistryModule := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}

		convey.Convey("TestGetInstances when url is nil", func() {
			instances := serviceRegistryModule.GetInstances(url)
			convey.So(len(instances), convey.ShouldEqual, 0)
		})

		url, _ = common.NewURL("test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)

		ins := serviceRegistryModule.GetInstances(url)
		convey.So(len(ins), convey.ShouldEqual, 0)

		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)
		var defaultServiceInstance = &registry.DefaultServiceInstance{
			ID:          "test",
			ServiceName: "test",
		}
		discovery.EXPECT().GetInstances(gomock.Any()).Return([]registry.ServiceInstance{defaultServiceInstance})

		err := serviceRegistryModule.Watch(context.Background(), url)
		convey.So(err, convey.ShouldBeNil)

		ins = serviceRegistryModule.GetInstances(url)
		convey.So(len(ins), convey.ShouldEqual, 1)
		convey.So(ins[0], convey.ShouldEqual, defaultServiceInstance)
	})
}

func TestGetAllInstances(t *testing.T) {
	convey.Convey("TestGetAllInstances", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		newServiceRegistry.allInstances.Store("test", 1)
		newServiceRegistry.allInstances.Store(registry.ServiceMeta{}, 1)
		newServiceRegistry.allInstances.Store(registry.ServiceMeta{}, []registry.ServiceInstance{})

		instances := newServiceRegistry.GetAllInstances()
		convey.So(len(instances), convey.ShouldEqual, 1)
	})
}

func TestGetServiceNames(t *testing.T) {
	convey.Convey("TestGetServiceNames", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		serviceNames := newServiceRegistry.GetServiceNames()
		convey.So(serviceNames.Empty(), convey.ShouldBeTrue)

		url, _ := common.NewURL("open-contract-core-test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)

		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)

		_ = newServiceRegistry.Watch(context.Background(), url)

		serviceNames = newServiceRegistry.GetServiceNames()
		convey.So(serviceNames.Empty(), convey.ShouldBeFalse)
	})
}

func TestRemoveListener(t *testing.T) {

	convey.Convey("TestRemoveListener", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}

		url, _ := common.NewURL("open-contract-core-test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)
		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)

		_ = newServiceRegistry.Watch(context.Background(), url)
		convey.So(newServiceRegistry.serviceNames.Size(), convey.ShouldEqual, 1)

		newServiceRegistry.RemoveListener(registry.ServiceMeta{Name: "open-contract-core-test", Namespace: "unify-test-1", Group: "DEFAULT_GROUP"})
		convey.So(newServiceRegistry.serviceNames.Size(), convey.ShouldEqual, 0)

	})

}

func TestOnEvent(t *testing.T) {

	gmetric.Init("test")

	convey.Convey("TestOnEvent", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		listener := func(service string, ins []string) error {
			log.Println("ins listen ")
			return nil
		}
		newServiceRegistry.AddInsListener(listener)
		convey.So(newServiceRegistry, convey.ShouldNotBeNil)

		convey.Convey("TestNonServiceInstanceEvent", func() {
			err := newServiceRegistry.OnEvent(&observer.DefaultEvent{
				Key:    "testkey",
				Action: observer.EventTypeUpdate,
				Value:  "testdata",
			})
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("TestServiceInstanceEvent", func() {
			serviceMeta := registry.ServiceMeta{Name: "open-contract-core-test", Namespace: "unify-test-1", Group: "DEFAULT_GROUP"}
			serviceInstances := []registry.ServiceInstance{&registry.DefaultServiceInstance{ID: "test", ServiceName: "test"}}
			newServiceRegistry.OnEvent(registry.NewServiceInstancesChangedEvent(serviceMeta, serviceInstances))

		})

	})
}

func TestGetEventType(t *testing.T) {
	convey.Convey("TestGetEventType", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		convey.So(newServiceRegistry.GetEventType(), convey.ShouldNotBeNil)
	})
}

func TestGetPriority(t *testing.T) {
	convey.Convey("TestGetPriority", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		convey.So(newServiceRegistry.GetPriority(), convey.ShouldEqual, 0)
	})
}

func TestAccept(t *testing.T) {
	convey.Convey("TestAccept", t, func() {
		newServiceRegistry := &serviceRegistry{
			ctx:          context.Background(),
			serviceNames: container.NewSet(),
		}
		convey.So(newServiceRegistry.Accept(&observer.DefaultEvent{}), convey.ShouldBeFalse)

		serviceMeta := registry.ServiceMeta{Name: "open-contract-core-test", Namespace: "unify-test-1", Group: "DEFAULT_GROUP"}
		serviceInstances := []registry.ServiceInstance{&registry.DefaultServiceInstance{ID: "test", ServiceName: "test"}}
		convey.So(newServiceRegistry.Accept(registry.NewServiceInstancesChangedEvent(serviceMeta, serviceInstances)), convey.ShouldBeFalse)

		url, _ := common.NewURL("open-contract-core-test",
			common.WithProtocol(constant.NacosProtocol),
			common.WithNamespace("unify-test-1"),
			common.WithGroup(constant.DEFAULT_GROUP),
		)
		discovery, _, _, teardown := setupTest(t)
		defer teardown()
		discovery.EXPECT().AddListener(gomock.Any()).Return(nil)

		_ = newServiceRegistry.Watch(context.Background(), url)

		convey.So(newServiceRegistry.Accept(registry.NewServiceInstancesChangedEvent(serviceMeta, serviceInstances)), convey.ShouldBeTrue)

	})
}

func TestGetInstancesNoCache(t *testing.T) {
	convey.Convey("TestGetInstancesNoCache", t, func() {

		convey.Convey("url is nil", func() {
			serviceRegistryModule := &serviceRegistry{
				ctx:          context.Background(),
				serviceNames: container.NewSet(),
			}
			ins := serviceRegistryModule.GetInstancesNoCache(nil)
			convey.So(ins, convey.ShouldBeNil)
		})
		convey.Convey("getRegistry return nil", func() {
			serviceRegistryModule := &serviceRegistry{
				ctx:          context.Background(),
				serviceNames: container.NewSet(),
			}
			url, _ := common.NewURL("test",
				common.WithProtocol("sasas"),
			)
			ins := serviceRegistryModule.GetInstancesNoCache(url)
			convey.So(len(ins), convey.ShouldEqual, 0)
		})
		convey.Convey("GetInstances return ins len == 0", func() {
			serviceRegistryModule := &serviceRegistry{
				ctx:          context.Background(),
				serviceNames: container.NewSet(),
			}
			url, _ := common.NewURL("test",
				common.WithProtocol(constant.NacosProtocol),
				common.WithNamespace("unify-test-1"),
				common.WithGroup(constant.DEFAULT_GROUP),
			)
			ins := serviceRegistryModule.GetInstancesNoCache(url)
			convey.So(len(ins), convey.ShouldEqual, 0)
		})
	})
}

func TestServiceRegistry_getInstances(t *testing.T) {
	convey.Convey("test ServiceRegistry getInstances", t, func() {
		sr := &serviceRegistry{}
		serv := registry.ServiceMeta{}
		ins := []registry.ServiceInstance{&registry.DefaultServiceInstance{}}
		sr.allInstances.Store(serv, ins)
		ins = sr.getInstances(serv)
		convey.So(len(ins), convey.ShouldEqual, 1)
	})
}

func TestServiceRegistry_AddInsListener(t *testing.T) {
	convey.Convey("test ServiceRegistry AddInsListener", t, func() {
		sr := NewServiceRegistry(context.Background())
		listener := func(service string, ins []string) error {
			log.Println("ins listen ")
			return nil
		}
		sr.AddInsListener(listener)
	})
}
