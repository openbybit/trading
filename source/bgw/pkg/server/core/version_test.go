package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer/dispatcher"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config_center"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"github.com/tj/assert"
)

func TestDecodeMetadata(t *testing.T) {
	vc := &versionController{}
	v := vc.decodeMetadata("xxx", "asas")
	assert.Nil(t, v)

	v = vc.decodeMetadata("current", "current")
	assert.Nil(t, v)

	v = vc.decodeMetadata("current", "{}")
	assert.NotNil(t, v)

}

func TestGetDescVersion(t *testing.T) {
	v := AppVersion{}
	s := v.GetDescVersion()
	assert.Equal(t, "", s)

	v = AppVersion{Version: AppVersionEntry{
		Resources: [2]ResourceEntry{{}, {Checksum: "as", LastTime: time.Now()}},
	}}
	s = v.GetDescVersion()
	assert.NotEqual(t, "", s)

	assert.Equal(t, "/BGW/current", (&AppVersion{}).GetEtcdKey())
	assert.Equal(t, "/BGW", (&AppVersion{}).Path())
}

func TestGetS3Key(t *testing.T) {
	v := &AppVersion{}
	k := v.GetS3Key()
	assert.Equal(t, "", k)
	tt := time.Now()
	v = &AppVersion{Version: AppVersionEntry{
		Resources: [2]ResourceEntry{{}, {Checksum: "as", LastTime: tt}},
	}}
	k = v.GetS3Key()
	assert.Equal(t, "/BGW/"+tt.Format("20060102150405"), k)
}

func TestGet(t *testing.T) {
	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()
	vc := &versionController{
		ctx:        ctx,
		versions:   container.NewConcurrentMap(),
		dispatcher: dispatcher.NewDirectEventDispatcher(ctx),
	}
	v := vc.get("")
	assert.Nil(t, v)
}

func TestRemove(t *testing.T) {
	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()
	vc := &versionController{
		ctx:        ctx,
		versions:   container.NewConcurrentMap(),
		dispatcher: dispatcher.NewDirectEventDispatcher(ctx),
	}
	err := vc.remove(&AppVersion{})
	assert.Nil(t, err)
}

func TestSet(t *testing.T) {
	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()
	vc := &versionController{
		ctx:        ctx,
		versions:   container.NewConcurrentMap(),
		dispatcher: dispatcher.NewDirectEventDispatcher(ctx),
	}
	err := vc.set(&AppVersion{})
	assert.Nil(t, err)
}

func TestLoopEvent(t *testing.T) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second)
	vc := &versionController{
		ctx: ctx,
	}
	cf()
	vc.loopEvent()

	ctx, cf = context.WithTimeout(context.Background(), 2*time.Second)
	vc = &versionController{
		ctx: ctx,
	}
	p := gomonkey.ApplyFuncReturn(time.NewTicker, time.NewTicker(time.Second)).
		ApplyPrivateMethod(reflect.TypeOf(vc), "load", func(prefix string) error {
			return nil
		})
	vc.loopEvent()
	p.Reset()

	ctx, cf = context.WithTimeout(context.Background(), 2*time.Second)
	vc = &versionController{
		ctx: ctx,
	}
	p = gomonkey.ApplyFuncReturn(time.NewTicker, time.NewTicker(time.Second)).
		ApplyPrivateMethod(reflect.TypeOf(vc), "load", func(prefix string) error {
			return errors.New("1212")
		})
	vc.loopEvent()
	p.Reset()
}

func TestDecode(t *testing.T) {
	a := AppVersion{}
	err := a.Decode(`{"namespace":"123","group":"111"}`)
	assert.NoError(t, err)
}

func TestGetDescVersionEntry(t *testing.T) {
	a := AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{Checksum: "xxxx"},
			{Checksum: "xxxx2"},
		}},
	}
	ss := a.GetDescChecksum()
	assert.Equal(t, "xxxx2", ss)
}

func TestGetConfigVersionEntry(t *testing.T) {
	vv := &AppVersionEntry{Resources: [2]ResourceEntry{
		{Checksum: "xxxx"},
		{Checksum: "xxxx2"},
	}}
	a := AppVersion{
		Version: *vv,
	}
	v := a.GetConfigVersionEntry()
	assert.Equal(t, &vv.Resources[0], v)
}

func TestGetCurrentVersion(t *testing.T) {
	vv := &AppVersionEntry{Resources: [2]ResourceEntry{
		{Checksum: "xxxx"},
		{Checksum: "xxxx2"},
	}}
	a := AppVersion{
		Version: *vv,
	}
	v := a.GetCurrentVersion()
	assert.Equal(t, vv, v)
}

func TestGetConfigChecksum(t *testing.T) {
	a := AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{Checksum: "xxxx"},
			{Checksum: "xxxx2"},
		}},
	}
	ss := a.GetConfigChecksum()
	assert.Equal(t, "xxxx", ss)
}

func TestEncode(t *testing.T) {
	a := AppVersion{}

	aa := a.Encode()
	assert.NotEqual(t, "", aa)
	p := gomonkey.ApplyFuncReturn(util.JsonMarshal, nil, errors.New("xxx"))
	aa = a.Encode()
	assert.Equal(t, "", aa)
	p.Reset()
}

func TestIsLocalGroup(t *testing.T) {
	vc := &versionController{}
	v := &AppVersion{
		Namespace: "xxx",
	}
	b := vc.isLocalGroup(v)
	assert.Equal(t, constant.BGW_GROUP, v.Group)
	assert.Equal(t, false, b)

	v = &AppVersion{
		Namespace: "",
		Group:     "123",
	}
	b = vc.isLocalGroup(v)
	assert.Equal(t, "123", v.Group)
	assert.Equal(t, false, b)

	v = &AppVersion{
		Namespace: "123",
		Group:     "123",
	}
	vc.group = "123"
	vc.namespace = "123"
	b = vc.isLocalGroup(v)
	assert.Equal(t, "123", v.Group)
	assert.Equal(t, true, b)
}

func TestVersionLoad(t *testing.T) {
	vc := newVersionController(context.Background())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := config_center.NewMockConfigure(ctrl)
	vc.configure = cfg

	cfg.EXPECT().GetChildren(gomock.Any(), gomock.Eq("/xxx")).Return(nil, nil, errors.New("xxx"))
	err := vc.load("/xxx")
	assert.EqualError(t, err, "xxx")

	// cfg.EXPECT().GetChildren(gomock.Any(), gomock.Eq("/xxx")).Return(nil, nil, nil)
	// err = vc.load("/xxx")
	// assert.EqualError(t, err, "xxx")
}

func TestVersionAddHistory(t *testing.T) {
	a := assert.New(t)
	version := NewAppVersion()

	for i := 0; i <= 10; i++ {
		version.SetCurrentResource(ResourceConfig, cast.ToString(i), time.Now())
		version.AddHistory(&version.Version)
	}

	a.Equal(version.History[0].Resources[0].Checksum, "10")
	for i, h := range version.History {
		t.Logf("%d: %s\n", i, h.Resources[0].Checksum)
	}

	version.SetCurrentResource(ResourceConfig, "11", time.Now())
	version.AddHistory(&version.Version)
	a.Equal(version.History[0].Resources[0].Checksum, "11")
	a.Equal(version.History[1].Resources[0].Checksum, "10")
}

func TestVersionController_Keys(t *testing.T) {
	Convey("test VersionController Keys", t, func() {
		vc := &versionController{
			versions: container.NewConcurrentMap(),
		}

		ks := vc.Keys()
		So(len(ks), ShouldEqual, 0)

		p := vc.GetPriority()
		So(p, ShouldEqual, -1)
		ty := vc.GetEventType()
		So(ty, ShouldNotBeNil)
	})
}

func TestAppVersion(t *testing.T) {
	Convey("test AppVersion", t, func() {
		av := &AppVersion{
			Version: AppVersionEntry{},
		}
		d := av.GetDesc()
		So(d, ShouldNotBeNil)

		c := av.GetConfig()
		So(c, ShouldNotBeNil)
	})
}

// func TestVersionToURL(t *testing.T) {
// 	a := assert.New(t)
// 	version := NewAppVersion()
// 	version.Version.LastTime = time.Now()
// 	version.App = "spot"
// 	version.Module = "user"
// 	version.SetCurrentResource(ResourceConfig, "111", time.Now())
// 	version.AddHistory(&version.Version)

// 	url, err := version.ToURL()
// 	a.Nil(err)
// 	a.NotNil(url)
// 	t.Log(url)
// }
