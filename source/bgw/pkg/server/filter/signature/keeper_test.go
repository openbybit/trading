package signature

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPrivateKeyKeeper_OnEvent(t *testing.T) {
	Convey("test new private keykeeper", t, func() {
		k := newPrivateKeyKeeper()
		pk := k.(*privateKeyKeeper)
		pk.signKey["mock"] = "111"
		pk.appKey["mock"] = "111"
		err := pk.OnEvent(nil)
		So(err, ShouldBeNil)
		e := &observer.DefaultEvent{}
		err = pk.OnEvent(e)
		So(err, ShouldBeNil)
		e.Value = "123"
		err = pk.OnEvent(e)
		So(err, ShouldBeNil)
		val := `
		app_name: bgw
		app_key:
		  - app_id: user
			key: 123
          - app_id: mock
			key: 123
		`
		e.Value = val
		err = pk.OnEvent(e)
		So(err, ShouldBeNil)
		e1 := &observer.DefaultEvent{}
		val1 := `
        app_name:
        app_key:
        `
		e1.Value = val1
		err = pk.OnEvent(e1)
		So(err, ShouldBeNil)

		_ = pk.GetEventType()
		p := pk.GetPriority()
		So(p, ShouldEqual, 0)
	})
}

func TestPrivateKeyKeeper_GetSignKey(t *testing.T) {
	Convey("test get sign key", t, func() {
		k := &privateKeyKeeper{
			signKey: map[string]string{"user": "123"},
			appKey:  map[string]string{"cht": "123"},
		}

		key, err := k.GetSignKey(context.Background(), "user")
		So(err, ShouldBeNil)
		So(key, ShouldEqual, "123")

		key, err = k.GetSignKey(context.Background(), "cht")
		So(err, ShouldBeNil)
	})
}
