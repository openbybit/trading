package nacos

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"

	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"
	"github.com/tj/assert"
)

func TestGetNacosConfig(t *testing.T) {
	a := assert.New(t)

	convey.Convey("TestGetNacosConfig", t, func() {

		cfg, err := GetNacosConfig("")
		convey.So(err, convey.ShouldBeNil)
		convey.So(cfg, convey.ShouldNotBeNil)

		client, err := gnacos.NewConfigClient(cfg)

		convey.So(err, convey.ShouldBeNil)
		convey.So(client, convey.ShouldNotBeNil)

		o := vo.ConfigParam{
			DataId:  "test_nacos_config",
			Group:   "test",
			Content: "hello world",
		}
		convey.Convey("TestPublishConfig", func() {
			success, err := client.PublishConfig(o)
			convey.So(err, convey.ShouldBeNil)
			convey.So(success, convey.ShouldBeTrue)
		})

		convey.Convey("TestGetConfig", func() {
			content, err := client.GetConfig(vo.ConfigParam{
				DataId: o.DataId,
				Group:  o.Group,
				OnChange: func(namespace, group, dataId, data string) {
					a.Equal(o.DataId, dataId, "data id changed")
					a.NotEmpty(o.Content, data)
					t.Logf("data changed to: %s", data)
				},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(content, convey.ShouldEqual, o.Content)
		})

		convey.Convey("TestRegisterInstance", func() {
			naming, err := gnacos.NewNamingClient(cfg)
			convey.So(err, convey.ShouldBeNil)
			convey.So(naming, convey.ShouldNotBeNil)

			registerSuccess, err := naming.RegisterInstance(vo.RegisterInstanceParam{
				Ip:          "127.0.0.1",
				Port:        5555,
				Enable:      true,
				Ephemeral:   false,
				ServiceName: "foo.bar",
				Healthy:     true,
				Metadata: map[string]string{
					"foo": "bar",
				},
			})
			convey.So(err, convey.ShouldBeNil)
			convey.So(registerSuccess, convey.ShouldBeTrue)
		})
	})

}
