package etcd

import (
	"bgw/pkg/config_center/etcd"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"

	"bgw/pkg/registry"
)

var root = "/config/var/"

func TestNewETCDServiceDiscovery(t *testing.T) {
	convey.Convey("TestNewETCDServiceDiscovery", t, func() {
		discovery, err := NewETCDServiceDiscovery(context.Background(), root)
		convey.So(err, convey.ShouldBeNil)
		convey.So(discovery, convey.ShouldNotBeNil)
	})
}

func TestDestroy(t *testing.T) {
	convey.Convey("TestDestroy", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := etcd.NewMockClient(ctrl)
		mockClient.EXPECT().Delete(gomock.Any()).Return(nil).Times(1)
		mockClient.EXPECT().Close().Times(2)

		esd := &etcdServiceDiscovery{
			client:            mockClient,
			registryInstances: []registry.ServiceInstance{&registry.DefaultServiceInstance{}},
		}
		esd.Destroy()

		mockClient.EXPECT().Delete(gomock.Any()).Return(errors.New("mock error")).Times(1)
		esd.Destroy()
	})
}

func TestRegister(t *testing.T) {
	convey.Convey("TestRegister", t, func() {
		discovery, err := NewETCDServiceDiscovery(context.Background(), root+"TestRegister")
		convey.So(err, convey.ShouldBeNil)
		convey.So(discovery, convey.ShouldNotBeNil)
		err = discovery.Register(&registry.DefaultServiceInstance{ServiceName: "TestRegister"})
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestUpdate(t *testing.T) {
	convey.Convey("TestUpdate", t, func() {
		discovery, err := NewETCDServiceDiscovery(context.Background(), root+"TestUpdate")
		convey.So(err, convey.ShouldBeNil)
		convey.So(discovery, convey.ShouldNotBeNil)
		err = discovery.Update(&registry.DefaultServiceInstance{ServiceName: "TestUpdate"})
		convey.So(err, convey.ShouldBeNil)

		convey.Convey("TestUpdate when unregister error", func() {
			applyMethod := gomonkey.ApplyMethod(reflect.TypeOf(discovery), "Unregister", func(discovery *etcdServiceDiscovery, instance registry.ServiceInstance) error {
				return errors.New("mock error")
			})
			defer applyMethod.Reset()
			err = discovery.Update(&registry.DefaultServiceInstance{ServiceName: "TestUpdate"})
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func TestUnregister(t *testing.T) {
	convey.Convey("TestUnregister", t, func() {
		discovery, err := NewETCDServiceDiscovery(context.Background(), root+"TestUnregister")
		convey.So(err, convey.ShouldBeNil)
		convey.So(discovery, convey.ShouldNotBeNil)
		err = discovery.Unregister(&registry.DefaultServiceInstance{ServiceName: "TestUnregister"})
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestGetInstances(t *testing.T) {
	convey.Convey("TestGetInstances", t, func() {
		convey.Convey("TestGetInstances when GetChildren return error", func() {
			discovery, err := NewETCDServiceDiscovery(context.Background(), root+"TestGetInstances")
			convey.So(err, convey.ShouldBeNil)
			convey.So(discovery, convey.ShouldNotBeNil)
			instances := discovery.GetInstances("TestGetInstances")
			convey.So(instances, convey.ShouldBeNil)
		})

		convey.Convey("TestGetInstances when GetChildren return data", func() {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClient := etcd.NewMockClient(ctrl)
			expectedKl := []string{"instance1", "instance2"}
			expectedVl := []string{"host1:1234", "host2:5678"}
			mockClient.EXPECT().GetChildren(gomock.Any(), gomock.Any()).Return(expectedKl, expectedVl, nil)
			esd := &etcdServiceDiscovery{
				client: mockClient,
			}
			instances := esd.GetInstances("TestGetInstances")
			convey.So(len(instances), convey.ShouldEqual, 2)
		})
	})
}

func TestAddListener(t *testing.T) {
	convey.Convey("TestAddListener", t, func() {
		discovery, err := NewETCDServiceDiscovery(context.Background(), root+"TestAddListener")
		convey.So(err, convey.ShouldBeNil)
		convey.So(discovery, convey.ShouldNotBeNil)

		controller := gomock.NewController(t)
		defer controller.Finish()
		listener := registry.NewMockServiceListener(controller)
		err = discovery.AddListener(listener)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestEtcdServiceDiscovery_DispatchEventByServiceName(t *testing.T) {
	convey.Convey("TestEtcdServiceDiscovery_DispatchEventByServiceName", t, func() {
		discovery, _ := NewETCDServiceDiscovery(context.Background(), root+"TestAddListener")
		err := discovery.DispatchEventByServiceName(registry.ServiceMeta{})
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestEtcdServiceDiscovery_DispatchEventForInstances(t *testing.T) {
	convey.Convey("TestEtcdServiceDiscovery_DispatchEventForInstances", t, func() {
		discovery, _ := NewETCDServiceDiscovery(context.Background(), root+"TestAddListener")
		err := discovery.DispatchEventForInstances(registry.ServiceMeta{}, []registry.ServiceInstance{})
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestEtcdServiceDiscovery_DispatchEvent(t *testing.T) {
	convey.Convey("TestEtcdServiceDiscovery_DispatchEvent", t, func() {
		discovery, _ := NewETCDServiceDiscovery(context.Background(), root+"TestAddListener")
		err := discovery.DispatchEvent(&registry.ServiceInstancesChangedEvent{})
		convey.So(err, convey.ShouldBeNil)
	})
}
