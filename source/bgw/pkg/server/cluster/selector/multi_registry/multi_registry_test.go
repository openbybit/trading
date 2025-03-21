package multi_registry

import (
	"bgw/pkg/common"
	"bgw/pkg/common/types"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/dynconfig"
	"context"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"
	"reflect"
	"testing"

	"bgw/pkg/common/util"

	"github.com/tj/assert"
)

func TestUnmarshal(t *testing.T) {
	a := assert.New(t)
	lbMeta := `{"duplicate_registry":{"registry":"trading-express"}}`
	registries := &registryMeta{}
	err := util.JsonUnmarshal([]byte(lbMeta), registries)
	a.NoError(err)
	t.Log(registries)
}

func TestSelector_SetDiscovery(t *testing.T) {
	convey.Convey("TestSelector_SetDiscovery", t, func() {
		s := &selector{}
		s.SetDiscovery(func(*common.URL) []registry.ServiceInstance {
			return nil
		})
	})
}

func TestSelector_Inject(t *testing.T) {
	convey.Convey("TestSelector_Inject", t, func() {
		s := &selector{}
		ctx := context.Background()

		convey.Convey("TestSelector_Inject invalid metas", func() {
			ctx, err := s.Inject(ctx, "invalid metas")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(ctx, convey.ShouldBeNil)
		})

		convey.Convey("TestSelector_Inject valid metas", func() {
			service := &service{
				Registry: "trading-express",
			}
			ctx, err := s.Inject(ctx, service)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ctx, convey.ShouldNotBeNil)

			rctx := fasthttp.RequestCtx{}
			ctx, err = s.Inject(&rctx, service)
			convey.So(err, convey.ShouldBeNil)
			convey.So(ctx, convey.ShouldBeNil)
		})
	})
}

func TestSelector_Extract(t *testing.T) {
	convey.Convey("TestSelector_Extract", t, func() {
		s := &selector{}
		_, err := s.Extract(&cluster.ExtractConf{
			Group:           "",
			Namespace:       "",
			LoadBalanceMeta: "",
		})
		convey.So(err, convey.ShouldNotBeNil)

		sc, err := s.Extract(&cluster.ExtractConf{
			Group:           "",
			Namespace:       "",
			LoadBalanceMeta: "{\"duplicate_registry\": {\"registry\": \"trading-express\"}}",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(sc, convey.ShouldNotBeNil)
	})
}

func TestSelector_urlFromContext(t *testing.T) {
	convey.Convey("TestSelector_urlFromContext", t, func() {
		s := &selector{}
		ctx := context.Background()
		u := s.urlFromContext(ctx)
		convey.So(u, convey.ShouldBeNil)

		ctx = contextWithSelectMetas(ctx, &service{
			Registry: "trading-express",
		})
		u = s.urlFromContext(ctx)
		convey.So(u, convey.ShouldNotBeNil)
	})
}

func TestSelector_selectZoneInstance(t *testing.T) {
	convey.Convey("TestSelector_selectZoneInstance", t, func() {
		instance, err := selectZoneInstance("test", []registry.ServiceInstance{})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(instance, convey.ShouldBeNil)

		instance, err = selectZoneInstance("test", []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "10"},
			},
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "1"},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(instance, convey.ShouldNotBeNil)
	})
}

func TestSelector_Select(t *testing.T) {
	convey.Convey("TestSelector_Select", t, func() {
		s := &selector{}
		ctx := types.Ctx{}

		convey.Convey("TestSelector_Select uid is 0", func() {
			_, err := s.Select(&ctx, []registry.ServiceInstance{})
			convey.So(err, convey.ShouldNotBeNil)

			GetPartition := gomonkey.ApplyMethod(reflect.TypeOf(registry.Metadata{}), "GetPartition", func(registry.Metadata) int {
				return 0
			})
			defer GetPartition.Reset()

			instance, err := s.Select(&ctx, []registry.ServiceInstance{
				&registry.DefaultServiceInstance{
					Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "10"},
				},
				&registry.DefaultServiceInstance{
					Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "1"},
				},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(instance, convey.ShouldNotBeNil)
		})

		convey.Convey("TestSelector_Select vip and zone raft", func() {
			metadata.ContextWithMD(&ctx, &metadata.Metadata{UID: 1234})
			checkZone := gomonkey.ApplyMethod(reflect.TypeOf(&dynconfig.VIPIdsListLoader{}), "CheckZone", func(*dynconfig.VIPIdsListLoader, int64) string {
				return "vipids"
			})
			_, err := s.Select(&ctx, []registry.ServiceInstance{})
			convey.So(err, convey.ShouldNotBeNil)

			urlFromContext := gomonkey.ApplyPrivateMethod(reflect.TypeOf(&selector{}), "urlFromContext", func(*selector, context.Context) *common.URL {
				url := &common.URL{}
				url.Addr = "127.0.0.1:8081"
				return url
			})
			s.SetDiscovery(func(*common.URL) []registry.ServiceInstance {
				return []registry.ServiceInstance{
					&registry.DefaultServiceInstance{
						Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "10"},
					},
					&registry.DefaultServiceInstance{
						Metadata: map[string]string{registry.RoleKey: string(registry.RoleTypeLeader), registry.ZoneKey: "test", registry.TermKey: "1"},
					},
				}
			})

			_, err = s.Select(&ctx, []registry.ServiceInstance{})
			convey.So(err, convey.ShouldNotBeNil)
			checkZone.Reset()
			urlFromContext.Reset()

			checkZone = gomonkey.ApplyMethod(reflect.TypeOf(&dynconfig.VIPIdsListLoader{}), "CheckZone", func(*dynconfig.VIPIdsListLoader, int64) string {
				return ""
			})
			checkZone = gomonkey.ApplyMethod(reflect.TypeOf(&dynconfig.GrayIdsListLoader{}), "CheckZone", func(*dynconfig.GrayIdsListLoader, int64) string {
				return "grayids"
			})
			_, err = s.Select(&ctx, []registry.ServiceInstance{})
			checkZone.Reset()
			convey.So(err, convey.ShouldNotBeNil)
		})

	})
}
