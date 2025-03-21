package openapi

import (
	"context"
	"sync"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWhiteListIpNacosLoader(t *testing.T) {

	t.Run("onEvent cast error", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(observer.NewBaseEvent("123"))
		assert.NoError(t, err)
	})
	t.Run("onEvent value nil", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{})
		assert.Nil(t, err)
	})
	t.Run("onEvent value wrong", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{Value: "123"})
		assert.Nil(t, err)
	})
	t.Run("onEvent user value wrong", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{Value: "123", Key: impIpListNacosKey})
		assert.NotNil(t, err)
	})
	t.Run("onEvent imp value wrong", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{Value: "123", Key: usIpListNacosKey})
		assert.NotNil(t, err)
	})
	t.Run("onEvent value success", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{
			Key:   impIpListNacosKey,
			Value: "{\"data\":[{\"brokerId\":\"Vl000246\",\"ipLists\":\"46.51.242.97,54.178.66.217,43.206.59.180,13.112.179.82\"}],\"version\":\"36036ea4-7b61-41e3-a357-55ecfb8f811e\"}"})
		assert.Nil(t, err)
		err = sw.OnEvent(&observer.DefaultEvent{
			Key:   impIpListNacosKey,
			Value: "{\"data\":[{\"brokerId\":\"Vl000246\",\"ipLists\":\"46.51.242.97,54.178.66.217,43.206.59.180,13.112.179.82\"}]}"})
		assert.Nil(t, err)
	})
	t.Run("GetIpWhiteList", func(t *testing.T) {
		sw := &ipListNaocsLoader{
			ctx:       context.Background(),
			impCache:  &sync.Map{},
			userCache: &sync.Map{},
		}
		err := sw.OnEvent(&observer.DefaultEvent{
			Key:   impIpListNacosKey,
			Value: "{\"data\":[{\"brokerId\":\"Vl000246\",\"ipLists\":\"123\"}],\"version\":\"36036ea4-7b61-41e3-a357-55ecfb8f811e\"}"})
		assert.Nil(t, err)
		err = sw.OnEvent(&observer.DefaultEvent{
			Key:   usIpListNacosKey,
			Value: "{\"data\":[{\"brokerId\":\"Vl000247\",\"ipLists\":\"1234\"}]}"})
		assert.Nil(t, err)
		b, ok := sw.GetIpWhiteList(nil, &user.MemberLoginExt{
			AppId: "Vl000246",
		})
		assert.Equal(t, true, ok)
		assert.Equal(t, "123", b)
		b, ok = sw.GetIpWhiteList(nil, &user.MemberLoginExt{
			AppId: "Vl000247",
		})
		assert.Equal(t, true, ok)
		assert.Equal(t, "1234", b)
		b, ok = sw.GetIpWhiteList(nil, &user.MemberLoginExt{
			AppId: "Vl0002478",
		})
		assert.Equal(t, false, ok)
		assert.Equal(t, "", b)
	})
}

func TestNewIpListNacosLoader(t *testing.T) {
	Convey("test newIpListNacosLoader", t, func() {
		_, err := newIpListNacosLoader(context.Background())
		So(err, ShouldBeNil)

		patch := gomonkey.ApplyFunc(env.IsProduction, func() bool { return true })
		defer patch.Reset()
		_, err = newIpListNacosLoader(context.Background())
		So(err, ShouldBeNil)

		il := &ipListNaocsLoader{}

		ty := il.GetEventType()
		So(ty, ShouldNotBeNil)

		p := il.GetPriority()
		So(p, ShouldEqual, -1)
	})
}
