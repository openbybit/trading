package getcd

import (
	"context"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestEtcd(t *testing.T) {
	convey.Convey("option", t, func() {
		convey.Convey("WithName", func() {
			getcdOption := &Options{}
			nameFunc := WithName("name")
			nameFunc(getcdOption)
			convey.So(getcdOption.Name, convey.ShouldEqual, "name")
		})

		convey.Convey("WithUsername", func() {
			getcdOption := &Options{}
			nameFunc := WithUsername("username")
			nameFunc(getcdOption)
			convey.So(getcdOption.Username, convey.ShouldEqual, "username")
		})

		convey.Convey("WithTimeout", func() {
			getcdOption := &Options{}
			nameFunc := WithTimeout(time.Second * 5)
			nameFunc(getcdOption)
			convey.So(getcdOption.Timeout, convey.ShouldEqual, time.Second*5)
		})

		convey.Convey("WithEndpoints", func() {
			getcdOption := &Options{}
			nameFunc := WithEndpoints("address1", "address2")
			nameFunc(getcdOption)
			convey.So(getcdOption.Endpoints, convey.ShouldResemble, []string{"address1", "address2"})
		})

		convey.Convey("WithHeartbeat", func() {
			getcdOption := &Options{}
			nameFunc := WithHeartbeat(10)
			nameFunc(getcdOption)
			convey.So(getcdOption.Heartbeat, convey.ShouldEqual, 10)
		})

		convey.Convey("WithPassword", func() {
			getcdOption := &Options{}
			nameFunc := WithPassword("123")
			nameFunc(getcdOption)
			convey.So(getcdOption.Password, convey.ShouldEqual, "123")
		})

		convey.Convey("NewClient", func() {
			ctx := context.TODO()
			_, err := NewClient(ctx)
			convey.ShouldEqual(err, ErrETCDEndpoints)

			gCtx, cancel := context.WithCancel(ctx)
			cli, err := NewClient(gCtx, WithEndpoints("k8s-istiosys-bgwingre-b8ffef1e78-28540305a7ddd534.elb.ap-southeast-1.amazonaws.com:2379"))
			convey.So(err, convey.ShouldBeNil)
			convey.So(cli, convey.ShouldNotBeNil)
			_ = cli.GetCtx()
			_ = cli.GetEndPoints()
			convey.ShouldEqual(cli.Valid(), true)

			err = cli.Put("/abc", "123")
			convey.So(err, convey.ShouldBeNil)
			rst, err := cli.Get("/abc")
			convey.So(err, convey.ShouldBeNil)
			convey.ShouldEqual(rst, "123")

			rst, v, err := cli.GetValAndRev("/abc")
			convey.So(err, convey.ShouldBeNil)
			convey.ShouldEqual(rst, "123")
			t.Log(rst, v)

			err = cli.Delete("/abc")
			convey.So(err, convey.ShouldBeNil)

			err = cli.RegisterTemp("/abc", "123")
			convey.So(err, convey.ShouldBeNil)

			ch, err := cli.WatchWithOption("/abc")
			convey.So(err, convey.ShouldBeNil)
			convey.So(ch, convey.ShouldNotBeNil)

			ch, err = cli.Watch("/abc")
			convey.So(err, convey.ShouldBeNil)
			convey.So(ch, convey.ShouldNotBeNil)

			ch, err = cli.WatchWithPrefix("/abc")
			convey.So(err, convey.ShouldBeNil)
			convey.So(ch, convey.ShouldNotBeNil)

			ks, vs, err := cli.GetChildrenKVList("/ab")
			convey.So(err, convey.ShouldBeNil)
			t.Log(ks, vs)

			ks, vs, err = cli.GetChildren("/ab")
			convey.So(err, convey.ShouldBeNil)
			t.Log(ks, vs)

			err = cli.Update("/abc", "123")
			convey.So(err, convey.ShouldBeNil)

			err = cli.UpdateWithRev("/abc", "123", 13794)
			convey.ShouldBeError(err, ErrCompareFail)

			err = cli.Create("/abc", "123")
			convey.ShouldBeError(err, ErrCompareFail)

			err = cli.BatchCreate([]string{"/ab", "/ac"}, []string{"123", "345"})
			convey.So(err, convey.ShouldBeNil)

			err = cli.Delete("/abc")
			convey.So(err, convey.ShouldBeNil)
			err = cli.Delete("/ab")
			convey.So(err, convey.ShouldBeNil)
			err = cli.Delete("/ac")
			convey.So(err, convey.ShouldBeNil)

			cli.Close()
			cancel()
		})
	})
}
