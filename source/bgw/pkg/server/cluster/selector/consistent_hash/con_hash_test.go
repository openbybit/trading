package consistent_hash

import (
	"context"
	"log"
	"testing"

	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"

	. "github.com/smartystreets/goconvey/convey"
)

func BenchmarkConHashSelect(b *testing.B) {
	ctx := context.Background()
	meta := metadata.NewMetadata()
	meta.UID = 12345
	meta.Route.Group = "groupa"
	ctx = metadata.ContextWithMD(ctx, meta)
	ctx = metadata.ContextWithSelectMetas(ctx, []string{"uid"})

	ins := make([]registry.ServiceInstance, 0, 6)
	for i := 0; i < 6; i++ {
		ins = append(ins, &registry.DefaultServiceInstance{
			Host: "127.0.0.1",
			Port: i,
		})
	}

	lb := New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := lb.Select(ctx, ins)
			if err != nil {
				log.Println(err.Error())
			}
		}
	})
}

func TestConHashSelect(t *testing.T) {

	Convey("test conhash select", t, func() {
		lb := New()

		ctx := context.Background()
		meta := metadata.NewMetadata()
		meta.UID = 123458900
		meta.Route.Group = "groupa"
		ctx = metadata.ContextWithMD(ctx, meta)
		ctx = metadata.ContextWithSelectMetas(ctx, []string{"uid"})

		ins1 := make([]registry.ServiceInstance, 0, 6)
		for i := 0; i < 6; i++ {
			ins1 = append(ins1, &registry.DefaultServiceInstance{
				Host: "127.0.0.1",
				Port: i,
			})
		}

		Convey("test stability", func() {

			n, err := lb.Select(ctx, ins1)
			So(err, ShouldBeNil)
			log.Println(n.GetID())

			for i := 0; i < 100; i++ {
				m, _ := lb.Select(ctx, ins1)
				So(m.GetID(), ShouldEqual, n.GetID())
			}

			ins2 := ins1[:3]
			x, err := lb.Select(ctx, ins2)
			So(err, ShouldBeNil)
			log.Println(x.GetID())

			for i := 0; i < 100; i++ {
				y, _ := lb.Select(ctx, ins2)
				So(y.GetID(), ShouldEqual, x.GetID())
			}

			m, err := lb.Select(ctx, ins1)
			So(err, ShouldBeNil)
			So(m.GetID(), ShouldEqual, n.GetID())

			m, err = lb.Select(ctx, nil)
			So(m, ShouldBeNil)
			So(err, ShouldNotBeNil)

			ins0 := []registry.ServiceInstance{&registry.DefaultServiceInstance{}}
			m, err = lb.Select(ctx, ins0)
			So(m, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
	})
}

func TestConHash_Extract(t *testing.T) {
	Convey("test Extract", t, func() {
		lb := &ConHash{}
		meta := &cluster.ExtractConf{}
		_, err := lb.Extract(meta)
		So(err, ShouldBeNil)

		meta.SelectKeys = []string{"uid"}
		_, err = lb.Extract(meta)
		So(err, ShouldBeNil)
	})
}

func TestConHash_Inject(t *testing.T) {
	Convey("test Extract", t, func() {
		lb := &ConHash{}
		_, err := lb.Inject(context.Background(), nil)
		So(err, ShouldNotBeNil)
		meta := []string{}
		_, err = lb.Inject(context.Background(), meta)
		So(err, ShouldNotBeNil)

		meta = []string{"uid"}
		_, err = lb.Inject(context.Background(), meta)
		So(err, ShouldBeNil)
	})
}
