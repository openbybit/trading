package getcd

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"github.com/smartystreets/goconvey/convey"
)

func TestNewEventListener(t *testing.T) {
	convey.Convey("NewEventListener", t, func() {
		gCtx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		cli, err := NewClient(gCtx, WithEndpoints("k8s-istiosys-bgwingre-b8ffef1e78-28540305a7ddd534.elb.ap-southeast-1.amazonaws.com:2379"))
		convey.So(err, convey.ShouldBeNil)
		convey.So(cli, convey.ShouldNotBeNil)

		en := NewEventListener(gCtx, cli)
		tl := &testListen{}
		en.ListenWithChildren("/ab", tl)
		go en.Wait()
		time.Sleep(time.Second)

		en.Listen("/ad", tl)
		en.Listen("/ac", tl)
		_ = cli.Put("/abc", "gg")
		_ = cli.Put("/ad", "gg")
		_ = cli.Put("/ac", "hh")
		_ = cli.Delete("/ad")
		_ = cli.Delete("/ac")
		_ = cli.Delete("/abc")

		time.Sleep(time.Second)
	})
}

type testListen struct{}

func (t *testListen) OnEvent(e observer.Event) error {
	fmt.Println(e.String())
	fmt.Println(e.GetSource())
	return nil
}

func (t *testListen) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

func (t *testListen) GetPriority() int {
	return 0
}
