package diagnosis

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"context"
	"errors"
	"strings"
	"sync"
)

type Diagnosis interface {
	Key() string

	Diagnose(ctx context.Context) (interface{}, error)
}

type module struct {
	diagnosisMap map[string]Diagnosis
	lock         sync.Mutex
}

var m = newDiagnosis()

func InitAdmin() {
	// curl 'http://localhost:6480/admin?cmd=diagnosisKeys 返回所有注册的key
	gapp.RegisterAdmin("diagnosisKeys", "diagnosis dependency service", m.onDiagnosisKeys)
	// curl 'http://localhost:6480/admin?cmd=diagnosis&keys keys为注册的模块key，多个逗号分割，不传则诊断全部key
	gapp.RegisterAdmin("diagnosis", "diagnosis dependency service", m.onDiagnosis)
	// curl 'http://localhost:6480/admin?cmd=diagnosisUpstream&registries=${registries}&group=DEMO_GROUP|DEFAULT_GROUP&namespace='
	// registries为下游服务注册nacos的注册名，多个逗号分割，group为注册的group，默认DEFAULT_GROUP，namespace为服务注册的namespace，不传默认
	gapp.RegisterAdmin("diagnosisUpstream", "diagnosis upstream service", m.onDiagnosisUpstream)
}

func Register(d Diagnosis) error {
	return m.Register(d)
}

func newDiagnosis() *module {
	return &module{diagnosisMap: make(map[string]Diagnosis)}
}

func (m *module) Register(d Diagnosis) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	dd, ok := m.diagnosisMap[d.Key()]
	if ok {
		msg := "diagnosis already reg"
		n := "name"
		glog.Error(context.Background(), msg, glog.String(n, dd.Key()))
		galert.Error(context.Background(), msg, galert.WithField(n, dd.Key()))
		return errors.New(msg)
	}
	m.diagnosisMap[d.Key()] = d
	return nil
}

func (m *module) onDiagnosisKeys(_ gapp.AdminArgs) (interface{}, error) {
	return wrapResp(func(data map[string]interface{}) {
		var ds []string
		m.lock.Lock()
		for _, d := range m.diagnosisMap {
			ds = append(ds, d.Key())
		}
		m.lock.Unlock()
		data["keys"] = ds
	}), nil
}

func (m *module) filterDiagnosis(args gapp.AdminArgs) []Diagnosis {
	p := args.GetStringBy("keys")
	var ds []Diagnosis
	m.lock.Lock()
	for k, v := range m.diagnosisMap {
		if p == "" || strings.Contains(p, k) {
			ds = append(ds, v)
		}
	}
	m.lock.Unlock()
	return ds
}

func (m *module) onDiagnosis(args gapp.AdminArgs) (interface{}, error) {
	return wrapResp(func(data map[string]interface{}) {
		ds := m.filterDiagnosis(args)
		for _, d := range ds {
			k := d.Key()
			r, err := diagnoseWithTimeout(d)
			if err != nil {
				data[k] = err.Error()
			} else {
				data[k] = r
			}
		}
	}), nil
}

func (m *module) onDiagnosisUpstream(args gapp.AdminArgs) (interface{}, error) {
	return wrapResp(func(data map[string]interface{}) {
		ctx := context.Background()
		p := args.GetStringBy("registries")
		if p == "" {
			return
		}
		namespace := args.GetStringBy("namespace")
		group := args.GetStringBy("group")
		rs := strings.Split(p, ",")
		if namespace == "" {
			namespace = config.GetRegistryNamespace()
		}
		if group == "" {
			group = constant.DEFAULT_GROUP
		}
		for _, r := range rs {
			errs := DiagnoseGrpcUpstream(ctx, r, namespace, group)
			data[r] = errs
		}
	}), nil
}

func wrapResp(handle func(data map[string]interface{})) map[string]interface{} {
	resp := make(map[string]interface{})
	resp["success"] = true
	resp["code"] = 0
	resp["msg"] = "请求成功"
	data := make(map[string]interface{})
	resp["data"] = data
	handle(data)
	return resp
}
