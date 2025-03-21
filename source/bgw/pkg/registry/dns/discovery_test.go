package dns

import (
	"context"
	"github.com/agiledragon/gomonkey/v2"
	"net"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"

	"bgw/pkg/registry"
)

func TestNewDNSDiscovery(t *testing.T) {
	convey.Convey("TestNewDNSDiscovery", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		convey.So(discovery, convey.ShouldNotBeNil)
	})
}

func TestDestroy(t *testing.T) {
	convey.Convey("TestDestroy", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		err := discovery.Destroy()
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestRegister(t *testing.T) {
	convey.Convey("TestRegister", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		err := discovery.Register(nil)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestUpdate(t *testing.T) {
	convey.Convey("TestUpdate", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		err := discovery.Update(nil)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestUnregister(t *testing.T) {
	convey.Convey("TestUnregister", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		err := discovery.Unregister(nil)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestGetInstances(t *testing.T) {
	convey.Convey("TestGetInstances", t, func() {
		dnsDiscovery := NewDNSDiscovery(context.Background(), "")
		instances := dnsDiscovery.GetInstances("dns://service.public")
		convey.So(len(instances), convey.ShouldEqual, 1)
		convey.So(instances[0].GetAddress(""), convey.ShouldEqual, "dns://service.public")

		dnsDiscovery = NewDNSDiscovery(context.Background(), "test")
		instances = dnsDiscovery.GetInstances("dns://service.public")
		convey.So(len(instances), convey.ShouldEqual, 0)

		applyMethod := gomonkey.ApplyMethod(reflect.TypeOf(&lookup{}), "LookupSRV", func(l *lookup, name string) ([]net.SRV, error) {
			return []net.SRV{
				{
					Target: "service.public",
					Port:   8080,
					Weight: 12,
				},
			}, nil
		})
		defer applyMethod.Reset()
		instances = dnsDiscovery.GetInstances("dns://service.public")
		convey.So(len(instances), convey.ShouldEqual, 1)

	})
}

func TestAddListener(t *testing.T) {
	convey.Convey("TestAddListener", t, func() {
		controller := gomock.NewController(t)
		defer controller.Finish()
		listener := registry.NewMockServiceListener(controller)
		listener.EXPECT().GetEventType().Return(reflect.TypeOf(&registry.ServiceInstancesChangedEvent{}))

		err := NewDNSDiscovery(context.Background(), "").AddListener(listener)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestDispatchEventByServiceName(t *testing.T) {
	convey.Convey("TestDispatchEventByServiceName", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		convey.So(discovery.DispatchEventByServiceName(registry.ServiceMeta{}), convey.ShouldBeNil)
	})
}

func TestDispatchEventForInstances(t *testing.T) {
	convey.Convey("TestDispatchEventForInstances", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		convey.So(discovery.DispatchEventForInstances(registry.ServiceMeta{}, nil), convey.ShouldBeNil)
	})
}

func TestDispatchEvent(t *testing.T) {
	convey.Convey("TestDispatchEvent", t, func() {
		discovery := NewDNSDiscovery(context.Background(), "")
		convey.So(discovery.DispatchEvent(&registry.ServiceInstancesChangedEvent{}), convey.ShouldBeNil)
	})
}
