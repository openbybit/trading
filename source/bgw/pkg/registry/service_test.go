package registry

import (
	"bgw/pkg/common/constant"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetEndPoints(t *testing.T) {
	convey.Convey("TestGetEndPoints", t, func() {
		instance := DefaultServiceInstance{}
		points := instance.GetEndPoints()
		convey.So(points, convey.ShouldBeNil)

		convey.Convey("TestGetEndPoints with wrong json", func() {
			instance.Metadata = Metadata{
				constant.SERVICE_INSTANCE_ENDPOINTS: "{\"ip\":\"1\",\"port\":\"2\",\"protocol\":\"3\"},{\"ip\":\"4\",\"port\":\"5\",\"protocol\":\"6\"}]",
			}
			points = instance.GetEndPoints()
			convey.So(points, convey.ShouldBeNil)
		})

		convey.Convey("TestGetEndPoints with right json", func() {
			instance.Metadata = Metadata{
				constant.SERVICE_INSTANCE_ENDPOINTS: "[{\"ip\":\"1\",\"port\":\"2\",\"protocol\":\"3\"},{\"ip\":\"4\",\"port\":\"5\",\"protocol\":\"6\"}]",
			}
			points = instance.GetEndPoints()
			convey.So(points, convey.ShouldNotBeNil)
		})
	})
}

func TestGetAddress(t *testing.T) {
	convey.Convey("TestGetAddress", t, func() {
		instance := DefaultServiceInstance{Address: map[string]string{constant.HttpProtocol: "1"}}
		addr := instance.GetAddress(constant.HttpProtocol)
		convey.So(addr, convey.ShouldEqual, "1")

		instance.Port = -1
		addr = instance.GetAddress(constant.GrpcProtocol)
		convey.So(addr, convey.ShouldEqual, "")

		instance.Host = "127.0.0.1"
		instance.Port = 9091
		addr = instance.GetAddress(constant.GrpcProtocol)
		convey.So(addr, convey.ShouldEqual, "127.0.0.1:9091")
	})
}

func TestGetName(t *testing.T) {
	convey.Convey("TestGetName", t, func() {
		instance := ServiceMeta{Name: "test"}
		name := instance.GetName()
		convey.So(name, convey.ShouldEqual, "test")

		instance.Namespace = "namespace"
		instance.Group = "group"
		name = instance.GetName()
		convey.So(name, convey.ShouldEqual, "test-namespace-group")
	})
}

func TestGetId(t *testing.T) {
	convey.Convey("TestGetId", t, func() {
		instance := DefaultServiceInstance{ID: "test"}
		convey.So(instance.GetID(), convey.ShouldEqual, "test")

		instance.ID = ""
		instance.Port = -1
		instance.Host = "0.0.0.0"
		convey.So(instance.GetID(), convey.ShouldEqual, instance.GetHost())

		instance.ID = ""
		instance.Port = 9091
		convey.So(instance.GetID(), convey.ShouldEqual, "0.0.0.0:9091")
	})
}

func TestDefaultServiceInstance_GetWeight(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetWeight", t, func() {
		instance := DefaultServiceInstance{Weight: 1}
		convey.So(instance.GetWeight(), convey.ShouldEqual, 1)

		instance.Weight = 0
		convey.So(instance.GetWeight(), convey.ShouldEqual, -1)
	})
}

func TestDefaultServiceInstance_GetMetadata(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetMetadata", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.GetMetadata(), convey.ShouldNotBeNil)
	})
}

func TestDefaultServiceInstance_Copy(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_Copy", t, func() {
		instance := DefaultServiceInstance{}
		copy := instance.Copy(&Endpoint{Port: 9091})
		convey.So(copy, convey.ShouldNotBeNil)
		convey.So(copy.GetPort(), convey.ShouldEqual, 9091)
	})
}

func TestServiceInstancesChangedEvent_String(t *testing.T) {
	convey.Convey("TestServiceInstancesChangedEvent_String", t, func() {
		event := ServiceInstancesChangedEvent{}
		convey.So(event.String(), convey.ShouldNotBeNil)

		event.Service = ServiceMeta{Name: "test", Namespace: "namespace", Group: "group"}
		convey.So(event.String(), convey.ShouldContainSubstring, "test-namespace-group")

	})
}

func TestDefaultServiceInstance_IsEnable(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_IsEnable", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.IsEnable(), convey.ShouldBeFalse)

		instance.Enable = true
		convey.So(instance.IsEnable(), convey.ShouldBeTrue)
	})
}

func TestDefaultServiceInstance_IsHealthy(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_IsHealthy", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.IsHealthy(), convey.ShouldBeFalse)

		instance.Healthy = true
		convey.So(instance.IsHealthy(), convey.ShouldBeTrue)
	})
}

func TestDefaultServiceInstance_GetPort(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetPort", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.GetPort(), convey.ShouldEqual, 0)

		instance.Port = 9091
		convey.So(instance.GetPort(), convey.ShouldEqual, 9091)
	})
}

func TestDefaultServiceInstance_GetHost(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetHost", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.GetHost(), convey.ShouldEqual, "")

		instance.Host = ""
		convey.So(instance.GetHost(), convey.ShouldEqual, "")
	})
}

func TestDefaultServiceInstance_GetServiceName(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetServiceName", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.GetServiceName(), convey.ShouldEqual, "")

		instance.ServiceName = "test"
		convey.So(instance.GetServiceName(), convey.ShouldEqual, "test")
	})
}

func TestServiceMeta_String(t *testing.T) {
	convey.Convey("TestServiceMeta_String", t, func() {
		meta := ServiceMeta{}
		convey.So(meta.String(), convey.ShouldEqual, "--")

		meta.Name = "test"
		convey.So(meta.String(), convey.ShouldContainSubstring, "test")
	})
}

func TestNewServiceInstancesChangedEvent(t *testing.T) {
	convey.Convey("TestNewServiceInstancesChangedEvent", t, func() {
		event := NewServiceInstancesChangedEvent(ServiceMeta{Name: "test"}, nil)
		convey.So(event, convey.ShouldNotBeNil)
	})
}

func TestDefaultServiceInstance_GetCluster(t *testing.T) {
	convey.Convey("TestDefaultServiceInstance_GetCluster", t, func() {
		instance := DefaultServiceInstance{}
		convey.So(instance.GetCluster(), convey.ShouldEqual, "")

		instance.Cluster = "test"
		convey.So(instance.GetCluster(), convey.ShouldEqual, "test")
	})
}
