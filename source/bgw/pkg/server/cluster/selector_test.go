package cluster

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/registry"
	"bgw/pkg/server/metadata"
)

func TestSelectorFunc_Select(t *testing.T) {
	Convey("test SelectorFunc Select", t, func() {
		var f SelectorFunc
		f = func(c context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
			if len(ins) == 0 {
				return nil, mockErr
			}
			return ins[0], nil
		}

		ins := []registry.ServiceInstance{&registry.DefaultServiceInstance{}}

		_, err := f.Select(context.Background(), ins)
		So(err, ShouldBeNil)
		Register("test selector", f)

		_ = GetSelector(context.Background(), "no exit selector")
		l := GetSelector(context.Background(), "test selector")
		So(l, ShouldNotBeNil)
	})
}

func Test_SwimLaneSelector(t *testing.T) {
	Convey("test SwimLaneSelector", t, func() {
		ctx := context.Background()
		res := SwimLaneSelector(ctx, nil)
		So(res, ShouldBeNil)
	})
}

func Test_azAware(t *testing.T) {
	Convey("test az aware", t, func() {
		ins := make([]registry.ServiceInstance, 0)
		res := azAware(ins)
		So(len(res), ShouldEqual, 0)

		ins = append(ins, &registry.DefaultServiceInstance{
			Metadata: map[string]string{"az": "az1"},
		})

		patch := gomonkey.ApplyFunc(env.AvailableZoneID, func() string { return "az1" })
		res = azAware(ins)
		So(len(res), ShouldEqual, 1)
		So(res[0].GetMetadata().GetAZName(), ShouldEqual, "az1")
		patch.Reset()

		patch = gomonkey.ApplyFunc(env.AvailableZoneID, func() string { return "az2" })
		res = azAware(ins)
		So(len(res), ShouldEqual, 1)
		patch.Reset()
	})
}

func Test_cloudAware(t *testing.T) {
	Convey("test cloud aware", t, func() {
		ins := make([]registry.ServiceInstance, 0)
		res := cloudAware(ins)
		So(len(res), ShouldEqual, 0)

		ins = append(ins, &registry.DefaultServiceInstance{Cluster: "tencent"})

		res = cloudAware(ins)
		So(len(res), ShouldEqual, 1)

		support = true
		res = cloudAware(ins)
		So(len(res), ShouldEqual, 1)

		cloudName = "tencent"
		patch2 := gomonkey.ApplyFunc(GetDegradeCloudMap, func() (map[string]struct{}, bool) {
			return make(map[string]struct{}), false
		})
		res = cloudAware(ins)
		So(len(res), ShouldEqual, 1)

		ins = append(ins[1:], &registry.DefaultServiceInstance{Cluster: DefaultClusterName})
		res = cloudAware(ins)
		So(len(res), ShouldEqual, 1)
		patch2.Reset()

		patch2 = gomonkey.ApplyFunc(GetDegradeCloudMap, func() (map[string]struct{}, bool) {
			return map[string]struct{}{"aws": {}}, true
		})
		defer patch2.Reset()
		res = cloudAware(ins)
		So(len(res), ShouldEqual, 1)
		So(res[0].GetCluster(), ShouldEqual, "DEFAULT")

		res = LocalAware(ins)
		So(len(res), ShouldEqual, 1)
	})
}

func Test_SwimLaneSelector2(t *testing.T) {
	Convey("test SwimLaneSelector2", t, func() {
		ins := []registry.ServiceInstance{
			&registry.DefaultServiceInstance{},
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{"lane-env": "swim1"},
			},
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{"lane-env": "base"},
			},
		}

		patch := gomonkey.ApplyFunc(env.IsProduction, func() bool { return true })
		res := SwimLaneSelector(context.Background(), ins)
		So(len(res), ShouldEqual, 3)
		patch.Reset()

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Intermediate.LaneEnv = "swim1"
		res = SwimLaneSelector(ctx, ins)

		md.Intermediate.LaneEnv = "swim2"
		res = SwimLaneSelector(ctx, ins)

		ins1 := ins[:1]
		res = SwimLaneSelector(ctx, ins1)

		ins2 := ins1[0:]
		res = SwimLaneSelector(ctx, ins2)
	})
}
