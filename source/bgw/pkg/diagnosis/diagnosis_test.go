package diagnosis

import (
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"sort"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	convey.Convey("RegisterDuplicate", t, func() {
		err := Register(&mockDiagnosis{
			k: "aaa",
		})
		convey.So(err, convey.ShouldBeNil)
		err = Register(&mockDiagnosis{
			k: "aaa",
		})
		convey.So(err.Error(), convey.ShouldEqual, "diagnosis already reg")
	})
}

func TestAdmin(t *testing.T) {
	InitAdmin()
	_ = Register(&mockDiagnosis{
		k: "bbb",
	})
	_ = Register(&mockDiagnosis{
		k: "ccc",
		r: "ddd",
	})
	_ = Register(&mockDiagnosis{
		k: "ddd",
		e: errors.New("xxx"),
	})
	convey.Convey("diagnosis admin", t, func() {

		convey.Convey("diagnosis onDiagnosisKeys", func() {

			r, err := m.onDiagnosisKeys(gapp.AdminArgs{})

			resp := r.(map[string]interface{})
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp["success"], convey.ShouldEqual, true)
			convey.So(resp["code"], convey.ShouldEqual, 0)
			convey.So(resp["msg"], convey.ShouldEqual, "请求成功")
			data := resp["data"].(map[string]interface{})
			sort.SliceStable(data["keys"], func(i, j int) bool {
				a := data["keys"].([]string)
				return strings.Compare(a[i], a[j]) > 0
			})
			mm := map[string]struct{}{"bbb": {}, "ccc": {}, "ddd": {}, "aaa": {}}
			sss := data["keys"].([]string)
			for i := range sss {
				_, ok := mm[sss[i]]
				convey.So(ok, convey.ShouldBeTrue)
			}
		})

		convey.Convey("diagnosis onDiagnosis", func() {

			// 有参数情况，诊断参数中的key
			r, err := m.onDiagnosis(gapp.AdminArgs{Options: map[string]string{"keys": "ccc"}})

			resp := r.(map[string]interface{})
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp["success"], convey.ShouldEqual, true)
			convey.So(resp["code"], convey.ShouldEqual, 0)
			convey.So(resp["msg"], convey.ShouldEqual, "请求成功")
			data := resp["data"].(map[string]interface{})
			convey.So(data["ddd"], convey.ShouldBeNil)
			convey.So(data["ccc"], convey.ShouldEqual, "ddd")
			convey.So(data["bbb"], convey.ShouldBeNil)

			// 无参数情况，诊断所有
			r, err = m.onDiagnosis(gapp.AdminArgs{})

			resp = r.(map[string]interface{})
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp["success"], convey.ShouldEqual, true)
			convey.So(resp["code"], convey.ShouldEqual, 0)
			convey.So(resp["msg"], convey.ShouldEqual, "请求成功")
			data = resp["data"].(map[string]interface{})
			convey.So(data["ddd"], convey.ShouldEqual, "xxx")
			convey.So(data["ccc"], convey.ShouldEqual, "ddd")
			convey.So(data["bbb"], convey.ShouldBeNil)
		})

		convey.Convey("diagnosis onDiagnosisUpstream", func() {
			// 什么都不传
			r, err := m.onDiagnosisUpstream(gapp.AdminArgs{})

			resp := r.(map[string]interface{})
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp["success"], convey.ShouldEqual, true)
			convey.So(resp["code"], convey.ShouldEqual, 0)
			convey.So(resp["msg"], convey.ShouldEqual, "请求成功")
			data := resp["data"].(map[string]interface{})
			convey.So(len(data), convey.ShouldEqual, 0)

			// 只指定registries
			rr := NewResult(errors.New("xxxx"))
			p := gomonkey.ApplyFuncReturn(DiagnoseGrpcUpstream, rr)
			defer p.Reset()
			r, err = m.onDiagnosisUpstream(gapp.AdminArgs{Options: map[string]string{"registries": "aaa,bbb"}})

			resp = r.(map[string]interface{})
			convey.So(resp, convey.ShouldNotBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp["success"], convey.ShouldEqual, true)
			convey.So(resp["code"], convey.ShouldEqual, 0)
			convey.So(resp["msg"], convey.ShouldEqual, "请求成功")
			data = resp["data"].(map[string]interface{})
			convey.So(len(data), convey.ShouldEqual, 2)
			convey.So(data["aaa"].(Result), convey.ShouldEqual, rr)
			convey.So(data["bbb"].(Result), convey.ShouldEqual, rr)
		})
	})
}
