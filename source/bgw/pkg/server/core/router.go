package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/groute"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/service/tradingroute"
)

const (
	keyCategory    = "category"
	keyAccountType = "accountType"
)

var errMultiCategory = errors.New("Parameter error: request have multiple categories")

// RouteDataProvider
type RouteDataProvider interface {
	GetMethod() string
	GetPath() string
	GetValue(key string) string
	GetValues(key string) [][]byte
	GetUserID() (int64, bool, error)
	GetAccountType(uid int64) (constant.AccountType, error)
}

type Route = groute.Route

// RouteManager route manager
// 底层实现:
// 1:每个method对应一个routeBucket
// 2:每个routeBucket由一个RadixTree管理path
// 3:每个path对应一组Routes(RouteGroup), 只有static类型path才能有多个route,其他模糊匹配只能有1个handler
// 4:RouteGroup基于一定规则插入route和查询route
//
// httprouter/gin实现的radix-tree不支持static优先执行
// https://github.com/gin-gonic/gin/issues/2416
// https://github.com/julienschmidt/httprouter/issues/73
// echo虽然支持static优先,但是代码耦合比较高,很难分离代码,而且echo的实现是基于path管理多个method
// account_type: https://uponly.larksuite.com/wiki/wikusBm6vPmCmQqQSmSXJ7xvYFc
//
//go:generate mockgen -source=router.go -destination=./router_mock.go -package=mock
type RouteManager interface {
	// Find 查找route handler
	FindRoute(ctx context.Context, provider RouteDataProvider) (*Route, error)
	// FindRoutes 通过method,path精确查找route group
	FindRoutes(method string, path string) *groute.Routes
	// Load 加载路由信息
	Load(appConf *AppConfig) error
	// Routes 获取所有路由信息
	Routes() []*Route
}

func parseRouteTypeBySelectorMeta(s *SelectorMeta) groute.RouteType {
	res := groute.ParseRouteType(s.RouteType)
	if res == groute.ROUTE_TYPE_UNKNOWN && s.RouteType == "" {
		if s.Groups != nil {
			res = groute.ROUTE_TYPE_CATEGORY
		} else {
			res = groute.ROUTE_TYPE_DEFAULT
		}
	}

	return res
}

type handlerCreateFunc func(mc *MethodConfig) (types.Handler, error)

func newRouteManager(fn handlerCreateFunc) RouteManager {
	return &routeManager{
		creator: fn,
		mgr:     groute.NewManager(),
	}
}

type userinfo struct {
	uid         int64
	isDemo      bool
	accountType constant.AccountType
}

func (u *userinfo) Fill(p RouteDataProvider) error {
	var err error
	if u.uid == 0 {
		u.uid, u.isDemo, err = p.GetUserID()
		if err != nil {
			return err
		}
	}

	u.accountType, err = p.GetAccountType(u.uid)
	if err != nil {
		return err
	}

	return nil
}

type routeManager struct {
	creator handlerCreateFunc // 创建handler回调函数
	mgr     groute.Manager    //
	mux     sync.Mutex        //
}

// Routes 根据path聚合routes并返回
func (rm *routeManager) Routes() []*Route {
	return rm.mgr.Routes()
}

func (rm *routeManager) FindRoutes(method string, path string) *groute.Routes {
	return rm.mgr.Find(method, path)
}

// Find find route
func (rm *routeManager) FindRoute(ctx context.Context, p RouteDataProvider) (*Route, error) {
	routes := rm.mgr.Find(p.GetMethod(), p.GetPath())
	if routes == nil {
		return nil, nil
	}

	items := routes.GetItems()
	if len(items) == 0 {
		return nil, nil
	}

	first := items[0]
	group := items
	if !first.IsPathType(groute.PATH_TYPE_STATIC) {
		// 模糊匹配只可能有一个handler
		return first, nil
	}

	user := &userinfo{}

	if first.IsRouteType(groute.ROUTE_TYPE_ALL_IN_ONE) {
		if err := user.Fill(p); err != nil {
			gmetric.IncDefaultError("route", "user_fill_error")
			glog.Errorf(ctx, "get userinfo fail, err: %v", err)
			// return nil, err
		}
		if user.isDemo {
			return first, nil
		}
		hit, err := rm.isHitAllInOne(ctx, first, p, user)
		if err != nil {
			gmetric.IncDefaultError("route", "check_aio_error")
			glog.Errorf(ctx, "check is aio fail, err: %v", err)
			if errors.Is(err, errMultiCategory) {
				return nil, err
			}
		}
		if hit {
			return first, nil
		}

		// 去除all in one路由后,保留之前逻辑
		group = items[1:]
		if len(group) == 0 {
			return nil, nil
		}
		first = group[0]
	}

	switch first.Type {
	case groute.ROUTE_TYPE_DEFAULT:
		return first, nil
	case groute.ROUTE_TYPE_CATEGORY, groute.ROUTE_TYPE_ACCOUNT_TYPE:
		key := keyCategory
		if first.Type == groute.ROUTE_TYPE_ACCOUNT_TYPE {
			key = keyAccountType
		}
		value := strings.ToLower(p.GetValue(key))
		accountType := constant.AccountTypeUnknown
		if routes.HasAccountTypeFlag() {
			if user.accountType == constant.AccountTypeUnknown {
				if err := user.Fill(p); err != nil {
					return nil, err
				}
			}
			accountType = user.accountType
		}

		glog.Debug(ctx, "[route] findRoute",
			glog.String("item0_type", items[0].Type.String()),
			glog.String("path", p.GetPath()),
			glog.String("key", key),
			glog.String("value", value),
			glog.String("account", accountType.String()),
			glog.Int64("size", int64(len(group))),
			glog.Int64("user_id", user.uid),
			glog.String("user_accounttype", user.accountType.String()),
		)

		route := rm.findRouteByValueAndAccount(ctx, value, groute.AccountType(accountType), group)
		if route != nil {
			return route, nil
		}

		// 校验是否能直接执行default,一定是最后一个
		last := group[len(group)-1]
		if last != nil && last.IsCatetoryDefault() {
			return last, nil
		}
		return nil, nil
	default:
		return nil, nil
	}
}

func (rm *routeManager) isHitAllInOne(ctx context.Context, route *Route, p RouteDataProvider, user *userinfo) (bool, error) {
	if len(route.Values) > 0 {
		vals := p.GetValues(keyCategory)
		if len(vals) > 1 {
			return false, errMultiCategory
		}
		if len(vals) == 0 || !route.Values.Contains(string(vals[0])) {
			var v string
			if len(vals) > 0 {
				v = string(vals[0])
			}
			glog.Debugf(ctx, "category not match: %v, %v", route.Values, v)
			return false, nil
		}
	}

	// 只有统保账户才会走all_in_one
	isUta := user.accountType != constant.AccountTypeNormal
	if !isUta {
		glog.Debugf(ctx, "uta not match, %v, %v", user.uid, user.accountType)
		return false, nil
	}

	isAioUser, err := tradingroute.GetRouting().IsAioUser(ctx, user.uid)
	if err != nil {
		glog.Debugf(ctx, "get routing fail err: %v", err)
		return false, err
	}

	if !isAioUser {
		glog.Debugf(ctx, "routing not match, uid: %v, %v", user.uid, isAioUser)
		return false, nil
	}

	return true, nil
}

func (r *routeManager) findRouteByValueAndAccount(ctx context.Context, value string, accountType groute.AccountType, group []*Route) *Route {
	if accountType != groute.AccountTypeUnknown {
		for _, item := range group {
			if (len(item.Values) == 0 || item.Values.Contains(value)) &&
				(item.Account == groute.AccountTypeAll || item.Account.Is(accountType)) {
				return item
			}
		}
	} else if value != "" {
		for _, item := range group {
			if item.Values.Contains(value) {
				return item
			}
		}
	}

	return nil
}

// Load 加载配置
func (rm *routeManager) Load(appConf *AppConfig) error {
	if appConf.App == "" || appConf.Module == "" {
		return fmt.Errorf("invalid name, app_name=%s, module_name=%s", appConf.App, appConf.Module)
	}

	rm.mux.Lock()
	defer rm.mux.Unlock()

	routes := make([]*Route, 0)

	appKey := appConf.Key()
	now := time.Now()
	for _, sc := range appConf.Services {
		for _, mc := range sc.Methods {
			// test unit情况下未设置
			mc.SetService(sc)
			if mc.Disable {
				continue
			}

			for _, method := range mc.GetMethod() {
				if method == "" {
					continue
				}

				method = strings.TrimPrefix(method, "HTTP_METHOD_")
				for _, path := range mc.GetPath() {
					if path == "" {
						continue
					}

					handler, err := rm.creator(mc)
					if err != nil {
						return fmt.Errorf("create handler fail,appkey: %v, registry: %v, path: %v, err: %v", appKey, sc.Registry, path, err)
					}

					metaStr := mc.GetSelectorMeta()
					meta, err := ParseSelectorMeta(metaStr)
					if err != nil {
						return fmt.Errorf("parse selector meta fail, appkey: %v, registry: %v, path: %v, meta: %v, err: %v", appKey, sc.Registry, path, metaStr, err)
					}

					route := &groute.Route{
						Path:       path,
						Method:     method,
						AppKey:     appKey,
						ServerName: sc.Registry,
						Type:       parseRouteTypeBySelectorMeta(meta),
						Handler:    handler,
						UpdateTime: now,
					}

					if meta != nil && meta.Groups != nil {
						if x, err := rm.buildGroups(route, meta); err != nil {
							return err
						} else {
							routes = append(routes, x...)
						}
					} else {
						routes = append(routes, route)
					}
				}
			}
		}
	}

	return rm.mgr.Replace(appKey, routes)
}

func (rm *routeManager) buildGroups(route *Route, meta *SelectorMeta) ([]*groute.Route, error) {
	result := make([]*groute.Route, 0, 1)
	routes := meta.Groups.Routes
	if len(routes) == 0 && !meta.Groups.Default {
		return nil, fmt.Errorf("empty group config, %s:%s", route.AppKey, route.Path)
	}

	for _, item := range routes {
		if item.Category == "" && item.Account == "" {
			// default category handler
			continue
		}

		account := groute.ToAccountType(item.Account)
		if account == groute.AccountTypeUnknown && item.Account != "" {
			return nil, fmt.Errorf("unknown account type, %v", item.Account)
		}

		r := *route
		r.Values = item.Categories()
		r.Account = account
		result = append(result, &r)
	}

	if meta.Groups.Default {
		// 如果指定default,则把value设置为空,只能有一个default,因为多个value为空会冲突
		r := *route
		r.Values = nil
		r.Account = groute.AccountTypeAll
		result = append(result, &r)
	}

	return result, nil
}
