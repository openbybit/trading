package gray

import (
	"log"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v3"
)

func TestConfigMarshal(t *testing.T) {
	Convey("test config marshal", t, func() {
		var val = `log:
  deadline: "2023-10-21 23:59:59"
  version: 0
  clusters: 
       openapi: 
       - strategy: uid # 采用枚举值
         value: [123456, 2314321, 434534265]
       siteapi: 
       - strategy: uid # 采用枚举值
         value: [123456, 2314321, 434534265]`

		d := make(map[string]*tagCfg, 0)

		err := yaml.Unmarshal([]byte(val), &d)
		if err != nil {
			log.Printf("err: %s", err.Error())
		} else {
			log.Printf("cfg: %v", d)
		}

		temp := make([]*strategy, 0)
		temp = append(temp, &strategy{Strags: "uid", Value: []any{123, 456}})
		strags := &Strategies{}
		*strags = temp

		tcfg := &tagCfg{
			Version:  1,
			Deadline: "48H",
			Clusters: map[string]*Strategies{"openapi": strags},
		}

		d["log"] = tcfg
		d["log2"] = tcfg

		out, err := yaml.Marshal(d)
		if err != nil {
			log.Printf("err: %s", err.Error())
		} else {
			log.Println("\n" + string(out))
		}
	})
}

func TestGrayCfg_RegisterGrayer(t *testing.T) {
	Convey("test grayCfg RegisterGrayer", t, func() {
		g := &grayer{
			tag:     "ut",
			cluster: "test",
		}

		globalCfg.RegisterGrayer(g)
	})
}

func TestGrayCfg_OnEvent(t *testing.T) {
	Convey("test grayCfg OnEvent", t, func() {
		err := globalCfg.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &observer.DefaultEvent{}
		e.Key = "ut"
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// 缺省，不会报错
		e.Value = `log:
  deadline: "2023-10-21 23:59:59"
  version: 1
  clusters:`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// 缺省，不会报错
		e.Value = `log:
  deadline: "2023-10-21 23:59:59"
  clusters:`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// 缺省，不报错
		e.Value = `log:
  deadline: "2023-10-21 23:59:59"
  version: 1
  clusters: 
    openapi: # 集群名称
    - strategy: uid # 采用枚举值
      value: [123456, 2314321, 434534265]
    - strategy: uid_tail # 尾号
    - strategy: service
      value: [routing://uta-engine, open-contract-core]
    - strategy: path   
      value: [v5/order/create, v5/order/cancel]`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// 类型不对，报错
		e.Value = `log:
  deadline: "2023-10-21 23:59:59"
  version: 1
  clusters: 
    openapi: # 集群名称
    - strategy: uid # 采用枚举值
      value: 123456`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// deadline为空, 走默认48h
		e.Value = `log:
  deadline:
  version: 1
  clusters: 
    openapi: # 集群名称
    - strategy: uid # 采用枚举值
      value: [123456]`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)

		// deadline为错误，走默认48h
		e.Value = `ut:
  deadline: "2023-10-21"
  version: 1
  clusters: 
    openapi: # 集群名称
    - strategy: uid # 采用枚举值
      value: [123456]`
		err = globalCfg.OnEvent(e)
		So(err, ShouldBeNil)
	})
}
