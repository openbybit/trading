package cluster

import (
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
)

var mockErr = errors.New("mock err")

func Test_InitCloudCfg(t *testing.T) {
	Convey("test InitCloudCfg", t, func() {
		InitCloudCfg()

		//mock nacos.NewNacosConfigure failed
		newNacosConfigure := gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(context.Context, ...nacos.Options) (config_center.Configure, error) {
			return nil, mockErr
		})
		doInitCloudCfg()
		newNacosConfigure.Reset()

		//mock nacos listen failed
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		nacosConfigure := config_center.NewMockConfigure(ctrl)
		nacosConfigure.EXPECT().Listen(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockErr)

		newNacosConfigure = gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(context.Context, ...nacos.Options) (config_center.Configure, error) {
			return nacosConfigure, nil
		})
		doInitCloudCfg()
		newNacosConfigure.Reset()
	})
}

func TestGetDegradeCloudMap(t *testing.T) {
	Convey("test GetDegradeCloudMap", t, func() {
		cloudCfg = &CloudDegrade{
			globalCfg: make(map[string]map[string][]string),
			appCfg:    make(map[string]map[string][]string),
		}

		_, _ = GetDegradeCloudMap()
	})
}

func TestCloudDegrade_GetDegradeMap(t *testing.T) {
	Convey("test CloudDegrade GetDegradeMap", t, func() {
		cd := &CloudDegrade{}
		appCfg := map[string]map[string][]string{}
		appCfg["tencent"] = map[string][]string{DegradeCloudNamesKeys: {"aws"}}
		appCfg["aliyun"] = map[string][]string{}
		cd.appCfg = appCfg

		cd.globalCfg = map[string]map[string][]string{}

		cloud = "tencent"
		m, ok := cd.GetDegradeMap()
		So(len(m), ShouldEqual, 1)
		So(ok, ShouldBeTrue)

		cloud = "aliyun"
		m, ok = cd.GetDegradeMap()
		So(len(m), ShouldEqual, 0)
		So(ok, ShouldBeFalse)

		cloud = "aws"
		m, ok = cd.GetDegradeMap()
		So(len(m), ShouldEqual, 0)
		So(ok, ShouldBeFalse)

		cd.globalCfg = cd.appCfg
		cd.appCfg = map[string]map[string][]string{}

		cloud = "tencent"
		m, ok = cd.GetDegradeMap()
		So(len(m), ShouldEqual, 1)
		So(ok, ShouldBeTrue)

		cloud = "aliyun"
		m, ok = cd.GetDegradeMap()
		So(len(m), ShouldEqual, 0)
		So(ok, ShouldBeFalse)

		cloud = "aws"
		m, ok = cd.GetDegradeMap()
		So(len(m), ShouldEqual, 0)
		So(ok, ShouldBeFalse)
	})
}

func TestCloudDegrade_OnEvent(t *testing.T) {
	Convey("test CloudDegrade OnEvent", t, func() {
		e1 := &observer.DefaultEvent{}
		cd := &CloudDegrade{
			globalCfg: make(map[string]map[string][]string),
			appCfg:    make(map[string]map[string][]string),
		}

		err := cd.OnEvent(e1)
		So(err, ShouldBeNil)

		e2 := &observer.DefaultEvent{}
		e2.Key = globalCloudCfgDataID
		e2.Value = "12345"

		patch := gomonkey.ApplyFunc(yaml.Unmarshal, func([]byte, any) error { return mockErr })
		err = cd.OnEvent(e2)
		So(err, ShouldNotBeNil)
		patch.Reset()

		e2.Value = `cloud:
  tencent:
     allow-downgrade-cloud:
      - aws`
		err = cd.OnEvent(e2)
		So(err, ShouldBeNil)
		So(cd.globalCfg["tencent"]["allow-downgrade-cloud"][0], ShouldEqual, "aws")

		e2.Key = appCloudCfgDataID
		err = cd.OnEvent(e2)
		So(err, ShouldBeNil)
		So(cd.appCfg["tencent"]["allow-downgrade-cloud"][0], ShouldEqual, "aws")
	})
}
