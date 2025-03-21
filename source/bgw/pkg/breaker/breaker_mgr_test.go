package breaker

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// 共用同一个manager，单测一起运行
var manager = &breakerMgr{
	services: make(map[string]*serviceBreakerMgr),
}

func TestNewBreakerMgr(t *testing.T) {
	Convey("test NewBreakerMgr", t, func() {
		_ = NewBreakerMgr()
	})
}

func TestBreakerMgr_GetOrSet(t *testing.T) {
	Convey("test BreakerMgr GetOrSet", t, func() {
		b := manager.GetOrSet("service_1", "target_1", "method_1")
		So(b, ShouldNotBeNil)
		So(len(manager.services), ShouldEqual, 1)

		b1 := manager.GetOrSet("service_1", "target_1", "method_1")
		So(b1, ShouldNotBeNil)
		So(len(manager.services), ShouldEqual, 1)
		So(b, ShouldEqual, b1)
	})
}

func TestBreakerMgr_OnInstanceRemove(t *testing.T) {
	Convey("test BreakerMgr OnInstanceRemove", t, func() {
		sbm, ok := manager.services["service_1"]
		So(ok, ShouldBeTrue)
		So(len(sbm.breakers), ShouldEqual, 1)
		err := manager.OnInstanceRemove("service_1", []string{"target_1"})
		So(err, ShouldBeNil)
		So(len(manager.services), ShouldEqual, 1)
		sbm, ok = manager.services["service_1"]
		So(ok, ShouldBeTrue)
		So(len(sbm.breakers), ShouldEqual, 0)

		err = manager.OnInstanceRemove("service_2", []string{})
		So(err, ShouldBeNil)
		So(len(manager.services), ShouldEqual, 1)
	})
}

func TestBreakerMgr_OnConfigUpdate(t *testing.T) {
	Convey("test BreakerMgr OnConfigUpdate", t, func() {
		sbm, ok := manager.services["service_1"]
		So(ok, ShouldBeTrue)
		So(len(sbm.breakers), ShouldEqual, 0)
		err := manager.OnConfigUpdate("service_1")
		So(err, ShouldBeNil)
		sbm, ok = manager.services["service_1"]
		So(ok, ShouldBeFalse)
	})
}

func TestServiceBreakerMgr_OnInstanceRemove(t *testing.T) {
	Convey("test ServiceBreakerMgr OnInstanceRemove", t, func() {
		sm := newServiceBreakerMgr("service_1")
		err := sm.onInstanceRemove([]string{"target_1"})
		So(err, ShouldBeNil)

		_ = sm.getOrSet("target_1", "method_1")
		So(len(sm.breakers), ShouldEqual, 1)
		err = sm.onInstanceRemove([]string{})
		So(len(sm.breakers), ShouldEqual, 1)

		err = sm.onInstanceRemove([]string{"target_1"})
		So(err, ShouldBeNil)
		So(len(sm.breakers), ShouldEqual, 0)
	})
}

// 150 ns/op
func BenchmarkBreakerMgr_GetOrSet3(b *testing.B) {
	bgr := NewBreakerMgr()
	service := fmt.Sprintf("service_%d", 0)
	target := fmt.Sprintf("target_%d", 0)
	method := fmt.Sprintf("method_%d", 0)
	for i := 0; i < b.N; i++ {
		_ = bgr.GetOrSet(service, target, method)
	}
}

// 274.4 ns/op
func BenchmarkBreakerMgr_OnInstanceRemove(b *testing.B) {
	bgr := NewBreakerMgr()
	service := fmt.Sprintf("service_%d", 0)
	target := fmt.Sprintf("target_%d", 0)
	method := fmt.Sprintf("method_%d", 0)
	_ = bgr.GetOrSet(service, target, method)
	for i := 0; i < b.N; i++ {
		tar := fmt.Sprintf("target_%d", i)
		_ = bgr.OnInstanceRemove(service, []string{tar})
	}
}
