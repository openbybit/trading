package random

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"

	"bgw/pkg/registry"
)

func TestNew(t *testing.T) {
	convey.Convey("TestNew", t, func() {
		selector := New()
		convey.So(selector, convey.ShouldNotBeNil)
	})
}

func TestRandomLoadBalance_Select(t *testing.T) {
	convey.Convey("TestRandomLoadBalance_Select", t, func() {
		rlb := &randomLoadBalance{}
		_, err := rlb.Select(context.Background(), []registry.ServiceInstance{})
		convey.So(err, convey.ShouldNotBeNil)

		instance, err := rlb.Select(context.Background(), []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "10"},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(instance, convey.ShouldNotBeNil)

		instance, err = rlb.Select(context.Background(), []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "10"},
			},
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "1"},
			}})
		rs := New()
		ctx := context.Background()

		_, err = rs.Select(ctx, nil)
		convey.So(err, convey.ShouldNotBeNil)

		ins := make([]registry.ServiceInstance, 0)
		ins = append(ins, &registry.DefaultServiceInstance{ID: "126"})
		res, err := rs.Select(ctx, ins)
		convey.So(err, convey.ShouldBeNil)
		convey.So(res.GetID(), convey.ShouldEqual, "126")
	})
}

func TestRandomLoadBalance_Select2(t *testing.T) {
	convey.Convey("test RandomLoadBalanc select", t, func() {
		rs := New()
		ins := []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Weight: 1,
			},
			&registry.DefaultServiceInstance{
				Weight: 2,
			},
		}

		tar, err := rs.Select(context.Background(), ins)
		convey.So(tar, convey.ShouldNotBeNil)
		convey.So(err, convey.ShouldBeNil)
	})
}
