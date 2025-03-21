package etcd

import (
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"context"
	"github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

func TestNewEtcdConfigure(t *testing.T) {
	convey.Convey("TestNewEtcdConfigure", t, func() {
		etcdConfigure, err := NewEtcdConfigure(context.Background())
		convey.So(err, convey.ShouldBeNil)
		convey.So(etcdConfigure, convey.ShouldNotBeNil)
	})
}

func TestGet(t *testing.T) {
	convey.Convey("TestGet", t, func() {
		etcdConfigure, err := NewEtcdConfigure(context.Background())
		etcdKey := "/bgw_test/test"
		err = etcdConfigure.Del(context.Background(), etcdKey)
		_, err = etcdConfigure.Get(context.Background(), etcdKey)
		convey.So(err, convey.ShouldNotBeNil)
		err = etcdConfigure.Put(context.Background(), etcdKey, "test")
		convey.So(err, convey.ShouldBeNil)

		data, err := etcdConfigure.Get(context.Background(), etcdKey)
		convey.So(err, convey.ShouldBeNil)
		convey.So(data, convey.ShouldEqual, "test")
	})
}

func TestGetChildren(t *testing.T) {
	convey.Convey("TestGetChildren", t, func() {
		etcdConfigure, _ := NewEtcdConfigure(context.Background())
		keyPrefix := "/bgw_test/TestGetChildren"

		//clear old data if exist
		_ = etcdConfigure.Del(context.Background(), keyPrefix+"/test1")

		_, _, err := etcdConfigure.GetChildren(context.Background(), keyPrefix)
		convey.So(err, convey.ShouldNotBeNil)

		_ = etcdConfigure.Put(context.Background(), keyPrefix+"/test1", "test1")

		keys, valuse, err := etcdConfigure.GetChildren(context.Background(), keyPrefix)
		convey.So(err, convey.ShouldBeNil)
		convey.So(keys, convey.ShouldContain, keyPrefix+"/test1")
		convey.So(keys, convey.ShouldHaveLength, 1)
		convey.So(valuse, convey.ShouldContain, "test1")
		convey.So(valuse, convey.ShouldHaveLength, 1)
	})
}

func TestListen(t *testing.T) {
	convey.Convey("TestListen", t, func() {
		etcdConfigure, _ := NewEtcdConfigure(context.Background())
		keyPrefix := "/bgw_test/TestListen"

		//clear old data if exist
		_ = etcdConfigure.Del(context.Background(), keyPrefix+"/test1")

		dataListener := &NoopListener{
			Ctx:   context.TODO(),
			Event: make(chan observer.Event),
		}
		err := etcdConfigure.Listen(context.Background(), keyPrefix+"/test1", dataListener)
		convey.So(err, convey.ShouldBeNil)

		// NOTICE:  direct listen will lose create msg
		time.Sleep(time.Second)
		_ = etcdConfigure.Put(context.Background(), keyPrefix+"/test1", "test1")

		msg := <-dataListener.Event
		mm := msg.(*observer.DefaultEvent)
		convey.So(mm.Value, convey.ShouldEqual, "test1")
	})
}

type NoopListener struct {
	Ctx context.Context
	observer.EventListener
	Event chan observer.Event
}

func (n *NoopListener) OnEvent(event observer.Event) error {
	glog.Debug(n.Ctx, "receive data change event", glog.Any("event", event))
	n.Event <- event
	return nil
}

func (n *NoopListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&observer.DefaultEvent{})
}

func (n *NoopListener) GetPriority() int {
	return 999
}
