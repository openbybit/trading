package core

import (
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/constant"
)

// func TestToURL(t *testing.T) {
// 	a := assert.New(t)
// 	demo := NewConfig("demo")
// 	app := new(AppConfig)
// 	err := demo.Unmarshal(app)
// 	app.Integrate()
// 	a.Nil(err)
// 	a.NotEmpty(app.Services)
// 	svc := app.Services[0]
// 	a.NotEmpty(svc.Methods)
// 	method := svc.Methods[0]
// 	a.NotEmpty(method.Name)
// 	a.NotNil(method.service)
// 	t.Logf("%#v", app)
// 	t.Log(method)
// }

func TestServiceConfigRegistry(t *testing.T) {
	a := assert.New(t)

	cases := []ServiceConfig{
		{
			Registry:  "test",
			Namespace: constant.NACOS_DEFAULT_NAMESPACE,
		},
		{
			Registry:  "test",
			Namespace: "",
		},
		{
			Registry:  "test",
			Namespace: "dev",
		},
	}

	for i := range cases {
		url := cases[i].GetRegistry("")
		a.Equal(url.Addr, cases[i].Registry)
		a.Equal(url.Protocol, constant.NacosProtocol)
		a.Equal(url.GetParam(constant.NAMESPACE_KEY, ""), cases[i].Namespace)
	}
}

func TestParseGroupMeta(t *testing.T) {
	data := `{"groups": {"category": "linear, option", "account": "normal", "default": "true"}}`
	res, err := ParseSelectorMeta(data)
	if err != nil {
		t.Error(err)
	}

	routes := res.Groups.GetRoutes()
	s, _ := json.Marshal(routes)
	fmt.Println(string(s), res.Groups.Default)
	// [{"category":"linear","account":"normal","default":""},{"category":"option","account":"normal","default":""}] true
}

func TestParseGroupMetaArray(t *testing.T) {
	data := `{"groups": [{"category": "linear, option", "account": "normal", "default": "true"}, {"category": "inverse"}]}`
	res, err := ParseSelectorMeta(data)
	if err != nil {
		t.Error(err)
	}

	routes := res.Groups.GetRoutes()
	s, _ := json.Marshal(routes)
	fmt.Println(string(s), res.Groups.Default)
	// [{"category":"linear","account":"normal","default":""},{"category":"option","account":"normal","default":""},{"category":"inverse","account":"","default":""}] true
}

func Test_ParseSelectorMeta(t *testing.T) {
	a := assert.New(t)

	groupsSrc := `{"groups":{"category":"option"}}`
	metas, err := ParseSelectorMeta(groupsSrc)
	a.NoError(err)
	t.Logf("%+v\n", metas.Groups)
	t.Log(metas.SelectKeys)
	t.Log("-----------------")

	metas, err = ParseSelectorMeta("")
	a.NoError(err)
	t.Logf("%+v\n", metas)
	t.Log("-----------------")

	selectKeysSrc := `{"select_keys":["_sp_business"]}`
	metas, err = ParseSelectorMeta(selectKeysSrc)
	a.NoError(err)
	t.Logf("%+v\n", metas.Groups)
	t.Log(metas.SelectKeys)
	t.Log("-----------------")
}

// func BenchmarkString(b *testing.B) {
// 	r := RouteKey{
// 		AppName:    "abc",
// 		ModuleName: "efg",
// 	}
// 	for i := 0; i < b.N; i++ {
// 		_ = fmt.Sprintf("%s.%s", r.AppName, r.ModuleName)
// 	}
// }

// func BenchmarkString1(b *testing.B) {
// 	r := RouteKey{
// 		AppName:    "abc",
// 		ModuleName: "efg",
// 	}
// 	for i := 0; i < b.N; i++ {
// 		_ = r.AsModule()
// 	}
// }

func TestServiceConfig_Key(t *testing.T) {
	Convey("test service config", t, func() {
		sc := &ServiceConfig{}
		sc.Filters = make([]Filter, 1)
		fs := sc.GetFilters()
		So(len(fs), ShouldEqual, 1)

		f := &Filter{}
		f.Args = "-- -- --args=12"
		as := f.GetArgs()
		So(len(as), ShouldEqual, 1)
	})
}

func TestMethodConfig_GetCategory(t *testing.T) {
	Convey("test method config", t, func() {
		sc := &ServiceConfig{}
		sc.App = &AppConfig{}
		sc.Filters = make([]Filter, 1)
		mc := &MethodConfig{}
		mc.AllowWSS = true
		mc.Filters = make([]Filter, 1)
		mc.SetService(sc)
		_ = mc.Service()
		fs := mc.GetFilters()
		So(len(fs), ShouldNotEqual, 0)

		mc.HttpMethod = "GET"
		mc.HttpMethods = []string{"GET", "POST", "HTTP_METHOD_ANY"}
		ms := mc.GetMethod()
		So(len(ms), ShouldEqual, 7)

		mc.Path = "/v5"
		mc.Paths = []string{"v3", "v4"}
		ps := mc.GetPath()
		So(len(ps), ShouldEqual, 3)

		r := mc.GetAllowWSS()
		So(r, ShouldBeTrue)

		mc.Category = "1"
		c := mc.GetCategory()
		So(c, ShouldEqual, "1")

		rk := mc.RouteKey()
		So(rk, ShouldNotBeNil)

		mc.Selector = "s"
		s := mc.GetSelector()
		So(s, ShouldEqual, "s")

		mc.SelectorMeta = "s"
		s = mc.GetSelectorMeta()
		So(s, ShouldEqual, "s")

		mc.LoadBalanceMeta = "s"
		mc.GetLBMeta()
		So(s, ShouldEqual, "s")
	})
}
