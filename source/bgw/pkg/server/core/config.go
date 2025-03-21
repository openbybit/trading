package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"bgw/pkg/common"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	"github.com/spf13/viper"
)

type AppConfig struct {
	App      string           `json:"app,omitempty" yaml:"app,omitempty"`
	Module   string           `json:"module,omitempty" yaml:"module,omitempty"`
	Services []*ServiceConfig `json:"services,omitempty" yaml:"services,omitempty"`
	AppCfg   metadata.AppCfg  `json:"appCfg,omitempty" yaml:"appCfg,omitempty"`
}

// Key get key
func (a *AppConfig) Key() string {
	return a.App + "." + a.Module
}

type FilterConfig struct {
	Name string   `json:"name,omitempty" yaml:"name,omitempty"`
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`
}

type ServiceConfig struct {
	App *AppConfig `json:"-" yaml:"-"`

	// service discovery related
	Registry  string `json:"registry,omitempty" yaml:"registry,omitempty"`
	Namespace string `json:"namespace,omitempty" xml:"namespace,omitempty" yaml:"namespace,omitempty"`
	Group     string `json:"group,omitempty" xml:"group,omitempty" yaml:"group,omitempty"`

	// grpc dynamic message related
	Protocol   string          `json:"protocol,omitempty" xml:"protocol,omitempty" yaml:"protocol,omitempty"`
	Descriptor string          `json:"descriptor,omitempty" xml:"descriptor,omitempty" yaml:"descriptor,omitempty"`
	Package    string          `json:"package,omitempty" xml:"package,omitempty" yaml:"package,omitempty"`
	Name       string          `json:"interface,omitempty" xml:"interface,omitempty" yaml:"interface,omitempty"`
	Methods    []*MethodConfig `json:"methods,omitempty" xml:"methods,omitempty" yaml:"methods,omitempty"`

	// filter and loadbalance
	Filters         []Filter `json:"filters,omitempty" xml:"filters,omitempty" yaml:"filters,omitempty"`
	Selector        string   `json:"selector,omitempty" xml:"selector,omitempty" yaml:"selector,omitempty"`
	Timeout         int32    `json:"timeout,omitempty" xml:"timeout,omitempty" yaml:"timeout,omitempty"`
	SelectorMeta    string   `json:"selectorMeta,omitempty" xml:"selector_meta" yaml:"selectorMeta,omitempty"`
	AllowWSS        bool     `json:"allow_wss,omitempty" xml:"allow_wss" yaml:"allow_wss,omitempty"`
	LoadBalanceMeta string   `json:"loadBalanceMeta,omitempty" yaml:"loadBalanceMeta,omitempty"`
	Category        string   `json:"category,omitempty" yaml:"category,omitempty"`

	registries map[string]*common.URL // group -> url
	once       sync.Once
}

// GetFilters get filter
func (s *ServiceConfig) GetFilters() []Filter {
	if s != nil && len(s.Filters) > 0 {
		return s.Filters
	}

	return nil
}

const (
	demoAccountGroup = "DEMO_GROUP"
	strict           = "strict" // default
	allToDefault     = "allToDefault"
	defaultOnly      = "defaultOnly"
	demoOnly         = "demoOnly"
)

// GetRegistry get registry from service registry config
// if not fully-qulified-registry name, then use default nacos(NOTE: etcd not support yet)
// otherwise use user specified registry like: dns://test.com
func (s *ServiceConfig) GetRegistry(group string) *common.URL {
	s.once.Do(func() {
		url, _ := common.NewURL(s.Registry,
			common.WithProtocol(constant.NacosProtocol),
			common.WithGroup(s.Group),
			common.WithNamespace(s.Namespace),
		)
		if len(s.registries) == 0 {
			s.registries = map[string]*common.URL{s.Group: url}
		}

		durl, _ := common.NewURL(s.Registry,
			common.WithProtocol(constant.NacosProtocol),
			common.WithGroup(demoAccountGroup),
			common.WithNamespace(s.Namespace),
		)
		s.registries[demoAccountGroup] = durl
	})

	return s.registries[group]
}

func (s *ServiceConfig) Key() string {
	return s.App.Key()
}

type Filter struct {
	Name       string `json:"name,omitempty" xml:"name" yaml:"name"`
	PluginName string `json:"pluginName,omitempty" xml:"pluginName" yaml:"pluginName,omitempty"`
	Args       string `json:"args,omitempty" xml:"args" yaml:"args"`
	Disable    bool   `json:"disable,omitempty" xml:"disable,omitempty" yaml:"disable,omitempty"`
}

// GetArgs get args
func (f Filter) GetArgs() []string {
	tmp := strings.Split(f.Args, "--")
	args := make([]string, 0, 2)
	for _, s := range tmp {
		k := strings.TrimSpace(s)
		if k == "" {
			continue
		}
		args = append(args, "--"+k)
	}
	return args
}

type MethodConfig struct {
	Name            string         `json:"name,omitempty" yaml:"name,omitempty"`
	Version         string         `json:"version,omitempty" yaml:"version,omitempty"`
	Path            string         `json:"path,omitempty" yaml:"path,omitempty"`
	Paths           []string       `json:"paths,omitempty" yaml:"paths,omitempty"`
	Group           string         `json:"group,omitempty" yaml:"group,omitempty"`
	HttpMethod      string         `json:"httpMethod,omitempty" yaml:"httpMethod,omitempty"`
	HttpMethods     []string       `json:"httpMethods,omitempty" yaml:"httpMethods,omitempty"`
	Filters         []Filter       `json:"filters,omitempty" yaml:"filters,omitempty"`
	service         *ServiceConfig `json:"-" yaml:"-"`
	ACL             metadata.ACL   `json:"acl,omitempty" yaml:"acl,omitempty"`
	Timeout         int32          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Selector        string         `json:"selector,omitempty" xml:"selector,omitempty" yaml:"selector,omitempty"`
	SelectorMeta    string         `json:"selectorMeta,omitempty" yaml:"selectorMeta,omitempty"`
	LoadBalanceMeta string         `json:"loadBalanceMeta,omitempty" yaml:"loadBalanceMeta,omitempty"`
	AllowWSS        bool           `json:"allow_wss,omitempty" xml:"allow_wss" yaml:"allow_wss,omitempty"`
	Disable         bool           `json:"disable,omitempty" xml:"disable" yaml:"disable,omitempty"`
	Category        string         `json:"category,omitempty" yaml:"category,omitempty"`
	GroupRouteMode  string         `json:"groupRouteMode,omitempty" yaml:"groupRouteMode,omitempty"`
	Breaker         bool           `json:"breaker,omitempty" yaml:"breaker,omitempty"`
}

type ACL struct {
	Group      string   `json:"group,omitempty" yaml:"group"`
	Permission string   `json:"permission,omitempty" yaml:"permission"`
	AllGroup   bool     `json:"all_group,omitempty" yaml:"all_group"`
	Groups     []string `json:"groups,omitempty" yaml:"groups"`
}

type FilterDefinition struct {
	Name       string `json:"name" yaml:"name"`
	LookupName string `json:"lookup_name" yaml:"lookup_name"`
	PluginPath string `json:"plugin_path" yaml:"plugin_path"`
	Args       string `json:"args" yaml:"args"`
}

// Integrate integrate
func (a *AppConfig) Integrate() {
	for i := 0; i < len(a.Services); i++ {
		service := a.Services[i]
		service.App = a
		for j := 0; j < len(service.Methods); j++ {
			meth := service.Methods[j]
			meth.SetService(service)
		}
	}
}

// Unmarshal get AppConfig from io.Reader
func (a *AppConfig) Unmarshal(data io.Reader, typ string) error {
	v := viper.New()
	v.SetConfigType(typ)

	if err := v.ReadConfig(data); err != nil {
		return err
	}

	if err := v.Unmarshal(a); err != nil {
		return err
	}

	a.Integrate()
	return nil
}

// GetFullQulifiedName return the full qualified name
func (s *ServiceConfig) GetFullQulifiedName() string {
	return fmt.Sprintf("%s.%s", s.Package, s.Name)
}

// GetFilters get all filter
func (m *MethodConfig) GetFilters() []Filter {
	data := make(map[string]Filter, 4)
	var index = make([]string, 0, 4) // record filter order in map

	for _, filter := range m.Service().GetFilters() {
		if !filter.Disable {
			data[filter.Name] = filter
			index = append(index, filter.Name)
		}
	}
	for _, filter := range m.Filters {
		if !filter.Disable {
			data[filter.Name] = filter
			index = append(index, filter.Name)
		} else {
			delete(data, filter.Name)
		}
	}

	var filters = make([]Filter, 0, 2)
	// put the response filter in the customer filters forefront
	if f, ok := data[filter.ResponseFilterKey]; ok {
		filters = append(filters, f)
		delete(data, filter.ResponseFilterKey)
	}
	// delete the request filter in the customer filters, will put the request filter in the customer filter backend
	reqFilter, ok := data[filter.RequestFilterKey]
	if ok {
		delete(data, filter.RequestFilterKey)
	}
	for _, name := range index {
		filter, ok := data[name]
		if ok {
			filters = append(filters, filter)
			delete(data, name)
		}
	}
	if reqFilter.Name != "" && m.service.Protocol != constant.HttpProtocol {
		// request filter do not support http
		filters = append(filters, reqFilter)
	}

	return filters
}

// SetService set service
func (m *MethodConfig) SetService(service *ServiceConfig) {
	m.service = service
}

// Service return service
func (m *MethodConfig) Service() *ServiceConfig {
	return m.service
}

// GetMethod get the slice of method
func (m *MethodConfig) GetMethod() []string {
	meths := make([]string, 0, 2)
	if m.HttpMethod != "" {
		meths = append(meths, m.checkMethod(m.HttpMethod)...)
	}
	for _, method := range m.HttpMethods {
		meths = append(meths, m.checkMethod(method)...)
	}
	return meths
}

func (m *MethodConfig) checkMethod(meth string) []string {
	if meth == bhttp.HTTPMethodAny {
		return bhttp.HttpMethodAnyConfig
	}
	return []string{meth}
}

// GetPath get the slice of path
func (m *MethodConfig) GetPath() []string {
	paths := make([]string, 0, 2)
	if m.Path != "" {
		paths = append(paths, m.Path)
	}
	paths = append(paths, m.Paths...)
	return paths
}

// GetTimeout return the timeout
func (m *MethodConfig) GetTimeout() time.Duration {
	timeout := m.Service().Timeout
	if m.Timeout > 0 {
		timeout = m.Timeout
	}
	if timeout <= 0 {
		timeout = 4 // default
	}
	return time.Duration(timeout) * time.Second
}

// GetAllowWSS return the flag of allow wss
func (m *MethodConfig) GetAllowWSS() bool {
	allowWSS := m.Service().AllowWSS
	if m.AllowWSS {
		allowWSS = m.AllowWSS
	}
	return allowWSS
}

// GetCategory get category
func (m *MethodConfig) GetCategory() string {
	c := m.Service().Category
	if m.Category != "" {
		c = m.Category
	}
	return c
}

// RouteKey key for method, including service name, method name and method httpVerb
// indicate primary route key
func (m *MethodConfig) RouteKey() metadata.RouteKey {
	rk := metadata.RouteKey{
		Protocol:    m.service.Protocol,
		AppName:     m.service.App.App,
		ModuleName:  m.service.App.Module,
		Registry:    m.service.Registry,
		ServiceName: m.service.Name,
		MethodName:  m.Name,
		HttpMethod:  m.HttpMethod,
		Group:       m.Group,
		ACL:         m.ACL,
		Category:    m.GetCategory(),
		AppCfg:      m.Service().App.AppCfg,
		AllApp:      m.Service().App.AppCfg.Mapping,
	}
	// !NOTE: for RouteKey completeness
	// http protocol service name is nil
	if rk.ServiceName == "" {
		if m.Path != "" {
			rk.ServiceName = m.Path
		} else if len(m.Paths) > 0 {
			rk.ServiceName = m.Paths[0]
		}
	}
	// http protocol method name is nil
	if rk.MethodName == "" {
		if m.HttpMethod != "" {
			rk.MethodName = m.HttpMethod
		} else if len(m.HttpMethods) > 0 {
			rk.MethodName = m.HttpMethods[0]
		}
	}

	// http or grpc protocol method is nil when use methods
	if rk.HttpMethod == "" && len(m.HttpMethods) > 0 {
		rk.HttpMethod = m.HttpMethods[0]
	}
	return rk
}

// GetSelector get selector
func (m *MethodConfig) GetSelector() string {
	selector := m.Service().Selector
	if m.Selector != "" {
		selector = m.Selector
	}
	return selector
}

// GetSelectorMeta get selector meta
func (m *MethodConfig) GetSelectorMeta() string {
	metaStr := m.Service().SelectorMeta
	if m.SelectorMeta != "" {
		metaStr = m.SelectorMeta
	}
	return metaStr
}

// GetLBMeta get load balance meta
func (m *MethodConfig) GetLBMeta() string {
	lb := m.Service().LoadBalanceMeta
	if m.LoadBalanceMeta != "" {
		lb = m.LoadBalanceMeta
	}
	return lb
}

// SelectorMeta is the meta for route
type SelectorMeta struct {
	RouteType  string     `json:"route_type,omitempty" yaml:"route_type,omitempty"` // 路由类型,空/default/category/account_type
	Groups     *GroupMeta `json:"groups,omitempty" yaml:"groups,omitempty"`         // route one handler to invoke
	SelectKeys []string   `json:"select_keys" yaml:"select_keys"`                   // select keys for router
}

// 路由信息
type RouteTuple struct {
	Category string `json:"category"`
	Account  string `json:"account"`
	Default  string `json:"default"`
}

func (rr *RouteTuple) Categories() []string {
	return toCategories(rr.Category)
}

// GroupMeta is the meta of group
type GroupMeta struct {
	Default bool
	Routes  []RouteTuple
}

func (g *GroupMeta) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(data, []byte("[")) {
		if err := json.Unmarshal(data, &g.Routes); err != nil {
			return err
		}

		for _, x := range g.Routes {
			if x.Default != "" {
				g.Default = true
				break
			}
		}
	} else {
		t := RouteTuple{}
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}

		g.Routes = append(g.Routes, t)
		if t.Default != "" {
			g.Default = true
		}
	}

	return nil
}

func (g *GroupMeta) GetRoutes() []RouteTuple {
	var result []RouteTuple

	for _, item := range g.Routes {
		// 用于支持多category
		categories := toCategories(item.Category)
		if len(categories) > 0 {
			for _, c := range categories {
				result = append(result, RouteTuple{Category: c, Account: item.Account})
			}
		} else if item.Account != "" {
			result = append(result, RouteTuple{Account: item.Account})
		}
	}

	return result
}

func toCategories(s string) []string {
	tokens := strings.Split(s, ",")
	res := make([]string, 0, len(tokens))
	for _, x := range tokens {
		x = strings.TrimSpace(x)
		if x != "" {
			res = append(res, x)
		}
	}

	return res
}

// ParseSelectorMeta parse select metadata string
func ParseSelectorMeta(metaStr string) (*SelectorMeta, error) {
	if strings.TrimSpace(metaStr) == "" {
		return &SelectorMeta{}, nil
	}
	meta := &SelectorMeta{}
	// !note: 不要使用jsoniter,无法正常解析
	if err := json.Unmarshal([]byte(metaStr), meta); err != nil {
		return nil, err
	}

	return meta, nil
}
