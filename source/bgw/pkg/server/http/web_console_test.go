package http

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/etcd"
	retcd "bgw/pkg/remoting/etcd"
	"context"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/server/core"

	"github.com/tj/assert"

	j "encoding/json"
)

func TestStaticPreCheck(t *testing.T) {
	console := newWebConsole(context.Background())

	cfg, err := console.staticPreCheck("xx", "xxx")
	assert.EqualError(t, err, "While parsing config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `xxx` into map[string]interface {}")
	assert.Nil(t, cfg)

	ac := &core.AppConfig{}
	yml, _ := util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.EqualError(t, err, "config fill error, nil app or module")
	assert.Nil(t, cfg)

	ac = &core.AppConfig{
		App:    "asaas",
		Module: "sas",
	}
	yml, _ = util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.EqualError(t, err, "config fill error, module name must suffix with -http")
	assert.Nil(t, cfg)

	ac = &core.AppConfig{
		App:    "asaas",
		Module: "sas-http",
		Services: []*core.ServiceConfig{
			{Registry: ""},
		},
	}
	yml, _ = util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.EqualError(t, err, "config fill error, nil registry")
	assert.Nil(t, cfg)

	ac = &core.AppConfig{
		App:    "asaas",
		Module: "sas-http",
		Services: []*core.ServiceConfig{
			{
				Registry: "xxxx",
				Methods: []*core.MethodConfig{
					{
						Path: "as",
					},
					{
						Path: "as",
					},
				},
			},
		},
	}
	yml, _ = util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.EqualError(t, err, "config fill error, nil http method")
	assert.Nil(t, cfg)

	ac = &core.AppConfig{
		App:    "asaas",
		Module: "sas-http",
		Services: []*core.ServiceConfig{
			{
				Registry: "xxxx",
				Methods: []*core.MethodConfig{
					{
						HttpMethod: "POST",
						Path:       "as",
					},
					{
						HttpMethod: "POST",
						Path:       "as",
					},
				},
			},
		},
	}
	yml, _ = util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.EqualError(t, err, "method+path exists: POSTas")
	assert.Nil(t, cfg)

	ac = &core.AppConfig{
		App:    "asaas",
		Module: "sas-http",
		Services: []*core.ServiceConfig{
			{
				Registry: "xxxx",
				Methods: []*core.MethodConfig{
					{
						HttpMethod: "POST",
						Path:       "as",
					},
					{
						HttpMethod: "GET",
						Path:       "as",
					},
				},
			},
			{
				Registry: "xxxx2",
				Methods: []*core.MethodConfig{
					{
						HttpMethod: "POST",
						Path:       "as2",
					},
					{
						HttpMethod: "GET",
						Path:       "as2",
					},
				},
			},
		},
	}
	yml, _ = util.YamlMarshal(ac)

	cfg, err = console.staticPreCheck("xx", string(yml))
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "xxxx", cfg.Services[0].Registry)
	assert.Equal(t, "xxxx2", cfg.Services[1].Registry)
	assert.Equal(t, constant.HttpProtocol, cfg.Services[0].Protocol)
	assert.Equal(t, constant.HttpProtocol, cfg.Services[1].Protocol)

	assert.Equal(t, "POST", cfg.Services[0].Methods[0].HttpMethod)
	assert.Equal(t, "POST", cfg.Services[1].Methods[0].HttpMethod)
	assert.Equal(t, "as", cfg.Services[0].Methods[0].Path)
	assert.Equal(t, "as2", cfg.Services[1].Methods[0].Path)

	assert.Equal(t, "GET", cfg.Services[0].Methods[1].HttpMethod)
	assert.Equal(t, "GET", cfg.Services[1].Methods[1].HttpMethod)
	assert.Equal(t, "as", cfg.Services[0].Methods[1].Path)
	assert.Equal(t, "as2", cfg.Services[1].Methods[1].Path)

}

func TestUnlock(t *testing.T) {

	ec, err := retcd.NewConfigClient(context.Background())
	if err != nil {
		t.FailNow()
	}
	lc, err := newLocker(ec.GetRawClient(), "/BGW/test_ut")
	if err != nil {
		t.FailNow()
	}
	uf, err := lc.tryLock(context.Background(), "asas", time.Second*10)
	defer uf()
}

func TestSetVersion(t *testing.T) {
	gmetric.Init("TestSetVersion")
	console := newWebConsole(context.Background())
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := etcd.NewMockClient(ctrl)
	console.etcdClient = mc
	pp := gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {})
	defer pp.Reset()

	p := gomonkey.ApplyFuncReturn(j.Marshal, nil, errors.New("xxx"))
	err := console.setVersion(&core.AppVersion{})
	assert.EqualError(t, err, "xxx")
	p.Reset()

	mc.EXPECT().Put(gomock.Any(), gomock.Any()).Return(errors.New("xxxxx"))
	err = console.setVersion(&core.AppVersion{})
	assert.EqualError(t, err, "xxxxx")

	mc.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
	mc.EXPECT().Delete(gomock.Any()).Return(errors.New("xzxz"))
	err = console.setVersion(&core.AppVersion{})
	assert.NoError(t, err)

	p.Reset()

}

func TestBuildVersion(t *testing.T) {
	gmetric.Init("TestBuildVersion")
	console := newWebConsole(context.Background())
	p := gomonkey.ApplyFuncReturn(util.YamlMarshal, nil, errors.New("xxx"))
	err := console.buildVersion(&core.AppConfig{})
	assert.EqualError(t, err, "xxx")
	p.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := etcd.NewMockClient(ctrl)
	console.etcdClient = mc

	mc.EXPECT().Get(gomock.Any()).Return("", errors.New("xxxx"))
	err = console.buildVersion(&core.AppConfig{})
	assert.EqualError(t, err, "xxxx")

	mc.EXPECT().Get(gomock.Any()).Return("asas", nil)
	err = console.buildVersion(&core.AppConfig{})
	assert.EqualError(t, err, "readObjectStart: expect { or n, but found a, error found in #1 byte of ...|asas|..., bigger context ...|asas|...")

	ver := &core.AppVersion{
		Version: core.AppVersionEntry{
			Resources: [2]core.ResourceEntry{
				{
					Checksum: "8a80554c91d9fca8acb82f023de02f11",
				},
			},
		},
	}
	mc.EXPECT().Get(gomock.Any()).Return(util.ToJSONString(ver), nil)
	err = console.buildVersion(&core.AppConfig{})
	assert.NoError(t, err)

	ver = &core.AppVersion{
		Version: core.AppVersionEntry{
			Resources: [2]core.ResourceEntry{
				{
					Checksum: "xxxx",
				},
			},
		},
	}
	p = gomonkey.ApplyFuncReturn(newLocker, nil, errors.New("xxxx"))
	mc.EXPECT().Get(gomock.Any()).Return(util.ToJSONString(ver), nil)
	mc.EXPECT().GetRawClient()
	err = console.buildVersion(&core.AppConfig{})
	assert.EqualError(t, err, "xxxx")
	p.Reset()

	ver = &core.AppVersion{
		Version: core.AppVersionEntry{
			Resources: [2]core.ResourceEntry{
				{
					Checksum: "xxxx11",
				},
			},
		},
	}
	nacosCfg := config_center.NewMockConfigure(ctrl)
	console.nacosCfg = nacosCfg
	mc.EXPECT().Get(gomock.Any()).Return(util.ToJSONString(ver), nil)
	mc.EXPECT().GetRawClient()
	nacosCfg.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("xxxx"))

	lc := &locker{}
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(console), "setVersion", func(version *core.AppVersion) error {
		return nil
	}).ApplyFuncReturn(newLocker, lc, nil).ApplyPrivateMethod(reflect.TypeOf(lc), "tryLock", func(ctx context.Context, key string, leaseExpire time.Duration) (unlockFunc unlockFunc, err error) {
		return func() error {
			return errors.New("xxx")
		}, nil
	})
	err = console.buildVersion(&core.AppConfig{})
	assert.EqualError(t, err, "xxxx")

	p.Reset()
	ver = &core.AppVersion{
		Version: core.AppVersionEntry{
			Resources: [2]core.ResourceEntry{
				{
					Checksum: "xxxxm",
				},
			},
		},
	}
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(console), "setVersion", func(version *core.AppVersion) error {
		return nil
	}).ApplyFuncReturn(newLocker, lc, nil).ApplyPrivateMethod(reflect.TypeOf(lc), "tryLock", func(ctx context.Context, key string, leaseExpire time.Duration) (unlockFunc unlockFunc, err error) {
		return func() error {
			return errors.New("xxx")
		}, nil
	})
	mc.EXPECT().Get(gomock.Any()).Return(util.ToJSONString(ver), nil)
	mc.EXPECT().GetRawClient()
	nacosCfg.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Eq("{}\n")).Return(nil)
	err = console.buildVersion(&core.AppConfig{})
	assert.NoError(t, err)
	p.Reset()

	ver = &core.AppVersion{
		Version: core.AppVersionEntry{
			Resources: [2]core.ResourceEntry{
				{
					Checksum: "xxxxm",
				},
			},
		},
	}
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(console), "setVersion", func(version *core.AppVersion) error {
		return errors.New("xzxzx")
	}).ApplyFuncReturn(newLocker, lc, nil).ApplyPrivateMethod(reflect.TypeOf(lc), "tryLock", func(ctx context.Context, key string, leaseExpire time.Duration) (unlockFunc unlockFunc, err error) {
		return func() error {
			return errors.New("xxx")
		}, nil
	})
	mc.EXPECT().Get(gomock.Any()).Return(util.ToJSONString(ver), nil)
	mc.EXPECT().GetRawClient()
	nacosCfg.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Eq("{}\n")).Return(nil)
	err = console.buildVersion(&core.AppConfig{})
	assert.NoError(t, err)
	p.Reset()

}

func TestCheckConfigPath(t *testing.T) {
	gmetric.Init("TestCheckConfigPath")
	console := newWebConsole(context.Background())
	err := console.checkConfigPath(&core.MethodConfig{}, &unique{})
	assert.EqualError(t, err, "config fill error, nil http method")

	err = console.checkConfigPath(&core.MethodConfig{
		HttpMethod: "POST",
	}, &unique{})
	assert.EqualError(t, err, "config fill error, nil path or paths")

	err = console.checkConfigPath(&core.MethodConfig{
		HttpMethod:  bhttp.HTTPMethodAny,
		HttpMethods: []string{bhttp.HTTPMethodAny, "POST"},
		Path:        "1212",
	}, &unique{})
	assert.EqualError(t, err, "method config fill error, HttpMethods not support any method")

	err = console.checkConfigPath(&core.MethodConfig{
		HttpMethod:  "POST",
		HttpMethods: []string{"GET"},
		Path:        "1212",
	}, &unique{
		uniqueRegister: make(map[string]struct{}),
	})
	assert.NoError(t, err)

	err = console.checkConfigPath(&core.MethodConfig{
		HttpMethod:  "POST",
		HttpMethods: []string{"GET"},
		Path:        "1212",
		Paths:       []string{"123", "123"},
	}, &unique{
		uniqueRegister: make(map[string]struct{}),
	})
	assert.EqualError(t, err, "method+path exists: POST123")

	err = console.checkConfigPath(&core.MethodConfig{
		HttpMethod:  "POST",
		HttpMethods: []string{"GET", "GET"},
		Path:        "1212",
		Paths:       []string{"123", "123"},
	}, &unique{
		uniqueRegister: make(map[string]struct{}),
	})
	assert.EqualError(t, err, "method+path exists: GET1212")
}

func TestHttpVersion(t *testing.T) {
	a := assert.New(t)

	console := newWebConsole(context.Background())
	a.NotNil(console)

	err := console.init()
	a.NoError(err)
}

func TestSetHttpVersion(t *testing.T) {
	a := assert.New(t)

	console := newWebConsole(context.Background())
	a.NotNil(console)

	err := console.init()
	a.NoError(err)

	file := "infra.hello-http"
	data, err := console.nacosCfg.Get(context.TODO(), file)
	a.NoError(err)

	a.NotNil(data)

	event := &observer.DefaultEvent{
		Key:    file,
		Action: observer.EventTypeUpdate,
		Value:  data,
	}

	err = console.OnEvent(event)
	a.NoError(err)

}

func TestWebConsole_OnEvent(t *testing.T) {
	Convey("test webConsole onEvent", t, func() {
		wb := newWebConsole(context.Background())
		_ = wb.init()
		ty := wb.GetEventType()
		So(ty, ShouldBeNil)
		p := wb.GetPriority()
		So(p, ShouldEqual, 0)

		err := wb.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &observer.DefaultEvent{}
		e.Key = BGWHttpListenFiles
		e.Value = `listen_http_files:
  - file1
  - file1
  - file2`
		err = wb.OnEvent(e)
		So(err, ShouldBeNil)

		e = &observer.DefaultEvent{}
		e.Key = "xxx"
		e.Value = `xxx`
		err = wb.OnEvent(e)
		So(err, ShouldBeNil)

		pa := gomonkey.ApplyPrivateMethod(reflect.TypeOf(wb), "buildVersion", func(ac *core.AppConfig) error {
			return errors.New("xxx")
		})
		defer pa.Reset()
		e = &observer.DefaultEvent{}
		e.Key = "xxx"
		e.Value = `app: 123
module: 1212-http`
		err = wb.OnEvent(e)
		So(err, ShouldBeNil)

	})
}
