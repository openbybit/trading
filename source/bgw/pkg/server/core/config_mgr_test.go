package core

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigEntry_dirty(t *testing.T) {
	Convey("test dirty", t, func() {
		var ce *configEntry
		res := ce.dirty(nil)
		So(res, ShouldBeTrue)

		ce = &configEntry{}
		res = ce.dirty(nil)
		So(res, ShouldBeFalse)

		ce.version = &ResourceEntry{}
		ce2 := &configEntry{}
		ce2.version = &ResourceEntry{
			Checksum: "22",
		}
		res = ce.dirty(ce2)
	})
}

func TestConfigManager_OnEvent(t *testing.T) {
	Convey("test ConfigManager", t, func() {
		cm := newConfigureManager(context.Background())
		err := cm.init()
		So(err, ShouldBeNil)

		ce := &configEntry{}
		ce.config = &AppConfig{}
		ce.version = &ResourceEntry{}
		err = cm.set("test_key", ce)

		res := cm.get("test_key")
		So(res, ShouldNotBeNil)

		vs := cm.Values()
		So(len(vs), ShouldEqual, 0)

		err = cm.remove("test_key")
		So(err, ShouldBeNil)

		err = cm.remove("test_key_2")
		So(err, ShouldBeNil)

		cm.addListener()

		ty := cm.GetEventType()
		So(ty, ShouldNotBeNil)

		p := cm.GetPriority()
		So(p, ShouldEqual, 10)

		err = cm.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &versionChangeEvent{
			BaseEvent: &observer.BaseEvent{},
		}

		e.Source = &AppVersion{}
		err = cm.OnEvent(e)
		So(err, ShouldNotBeNil)

		e.action = observer.EventTypeDel
		err = cm.OnEvent(e)
		So(err, ShouldBeNil)
	})
}
