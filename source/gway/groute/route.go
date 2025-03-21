package groute

import (
	"fmt"
	"strings"
	"time"
)

type Handler interface{}

type Route struct {
	Path       string      `json:"path"`        // 原始path,可能含有*,:,末尾也可能含有/
	Method     string      `json:"method"`      // http method
	AppKey     string      `json:"app_key"`     // AppKey={app}.{module}
	ServerName string      `json:"server_name"` // 服务名
	Type       RouteType   `json:"type"`        // 路由类型
	Values     OrderedList `json:"values"`      // category/account_type列表,有序且不区分大小写
	Account    AccountType `json:"account"`     // account类型,允许同时指定uma,uta
	UpdateTime time.Time   `json:"update_time"` // 最后更新时间
	Handler    Handler     `json:"-"`           // 路由handler回调
	pathType   PathType    `json:"-"`           // 路由类型
	realPath   string      `json:"-"`           // 加工后path
}

func (r *Route) GetPathType() PathType {
	return r.pathType
}

func (r *Route) IsPathType(t PathType) bool {
	return r.pathType == t
}

func (r *Route) IsRouteType(t RouteType) bool {
	return r.Type == t
}

func (r *Route) IsCatetoryDefault() bool {
	return (r.Type == ROUTE_TYPE_CATEGORY || r.Type == ROUTE_TYPE_ACCOUNT_TYPE) && len(r.Values) == 0 && r.Account == AccountTypeUnknown
}

func (r *Route) equal(o *Route) bool {
	return r.AppKey == o.AppKey && r.ServerName == o.ServerName &&
		r.Method == o.Method && r.realPath == o.realPath &&
		r.Account == o.Account && r.Values.Equal(o.Values)
}

func (r *Route) build() error {
	r.Method = strings.TrimPrefix(r.Method, "HTTP_METHOD_")
	for i, x := range r.Values {
		x = strings.TrimSpace(x)
		if x != "" {
			r.Values[i] = strings.ToLower(x)
		}
	}
	r.Values.Sort()

	if strings.ContainsRune(r.Path, ':') {
		r.pathType = PATH_TYPE_PARAMS
	} else if strings.ContainsRune(r.Path, '*') {
		if strings.Index(r.Path, "/*") != len(r.Path)-2 {
			return fmt.Errorf("path: %s,err: %w", r.Path, ErrInvalidPrefixPath)
		}
		r.pathType = PATH_TYPE_PRIFIX
	} else {
		r.pathType = PATH_TYPE_STATIC
	}

	if r.pathType == PATH_TYPE_PARAMS {
		return ErrNotSupportParamPath
	}

	r.realPath = trimPath(r.Path)

	// httprouter 要求/*后边必须跟一个名字
	if strings.HasSuffix(r.realPath, "/*") {
		r.realPath += "any"
	}

	if r.realPath == "" || r.realPath == "/" {
		return ErrInvalidRootPath
	}

	return nil
}

func (r *Route) String() string {
	return fmt.Sprintf("app_key: %v, service_name: %v, path: %v, method: %v", r.AppKey, r.ServerName, r.Path, r.Method)
}

func newRoutes(r *Route) *Routes {
	res := &Routes{}
	if r != nil {
		res.items = append(res.items, r)
	}
	return res
}

type Routes struct {
	items           []*Route // 路由列表,至少存在1个
	accountTypeFlag bool     // 标识是否需要获取用户的AccountType类型,避免无效的查询
}

func (routes *Routes) GetItems() []*Route {
	return routes.items
}

func (routes *Routes) HasAccountTypeFlag() bool {
	return routes.accountTypeFlag
}

func (routes *Routes) GetFirst() *Route {
	return routes.items[0]
}

func (routes *Routes) GetLast() *Route {
	return routes.items[len(routes.items)-1]
}

func (routes *Routes) isStaticPath() bool {
	return routes.items[0].IsPathType(PATH_TYPE_STATIC)
}

// canInsert 校验是否存在路由冲突
func (routes *Routes) canInsert(r *Route) error {
	if len(routes.items) == 0 {
		return nil
	}

	items := routes.items
	first := items[0]

	if r.IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
		// all-in-one必须独占
		if first.IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
			return fmt.Errorf("duplicate all-in-one handler, path: %v, app_key: %v-%v", r.Path, r.AppKey, first.AppKey)
		}

		return nil
	}

	// 忽略all-in-one
	if first.IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
		items = items[1:]
		if len(items) == 0 {
			return nil
		}
		first = items[0]
	}

	// default 独占
	if first.IsRouteType(ROUTE_TYPE_DEFAULT) {
		return fmt.Errorf("duplicate default handler, path: %v, app_key: %v-%v", r.Path, r.AppKey, first.AppKey)
	}

	// 规则1: 所有类型一样
	// 规则2: 只能有一个default handler
	// 规则3: category+account组合不能冲突
	for _, o := range items {
		if r.equal(o) {
			return fmt.Errorf("%w, path: %v, app_key: %v, service:%v", ErrDuplicateRoute, r.Path, r.AppKey, r.ServerName)
		}

		if r.Type != o.Type {
			return fmt.Errorf("different route type, path: %v, app_key: %v-%v, type: %v-%v", r.Path, r.AppKey, o.AppKey, r.Type, o.Type)
		}

		if r.IsCatetoryDefault() && o.IsCatetoryDefault() {
			return fmt.Errorf("duplicate default category route handler,  path: %v, app_key: %v-%v,", r.Path, r.AppKey, o.AppKey)
		}

		if !r.IsCatetoryDefault() && !o.IsCatetoryDefault() {
			if isSameCategory(r.Values, o.Values) && isSameAccount(r.Account, o.Account) {
				return fmt.Errorf("route value must be different, path: %v, app_key:%v-%v, category: %v-%v, account: %v-%v", r.Path, r.AppKey, o.AppKey, r.Values, o.Values, r.Account.String(), o.Account.String())
			}
		}
	}

	return nil
}

// insert 插入新路由,要求all-in-one必须在0位置,category default必须是最后
func (routes *Routes) insert(r *Route) error {
	if err := routes.canInsert(r); err != nil {
		return err
	}

	size := len(routes.items)
	if size == 0 || r.IsRouteType(ROUTE_TYPE_DEFAULT) {
		routes.items = append(routes.items, r)
	} else if r.IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
		// all in one must be at the first slot
		routes.items = append([]*Route{r}, routes.items...)
	} else {
		last := routes.items[size-1]
		if last.IsCatetoryDefault() {
			// default handler must be at the last slot
			routes.items = append(routes.items[:size-1], r, routes.items[size-1])
		} else {
			routes.items = append(routes.items, r)
		}
	}

	// 重新计算是否需要读取AccountType信息
	routes.accountTypeFlag = false
	for _, item := range routes.items {
		if item.Type != ROUTE_TYPE_CATEGORY && item.Type != ROUTE_TYPE_ACCOUNT_TYPE {
			continue
		}

		if item.IsCatetoryDefault() {
			continue
		}
		if item.Account != AccountTypeUnknown {
			// 任意一个开启account则需要调用account
			routes.accountTypeFlag = true
			break
		}
	}
	return nil
}

// category 为空则代表任意category
func isSameCategory(c1, c2 OrderedList) bool {
	if len(c1) == 0 || len(c2) == 0 || c1.ContainsAny(c2) {
		return true
	}

	return false
}

// account 未unknown则代表任意account
func isSameAccount(v1, v2 AccountType) bool {
	return v1 == AccountTypeAll || v2 == AccountTypeAll || v1.Is(v2)
}
