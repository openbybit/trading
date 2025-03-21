package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bgw/pkg/server/core"
	"bgw/pkg/service/tradingroute"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/wildcard"
)

type adminMgr struct {
	routeMgr core.RouteManager
}

func (m *adminMgr) init(routeMgr core.RouteManager) {
	m.routeMgr = routeMgr

	gapp.RegisterAdmin("ping", "", m.onPing)
	// curl 'http://localhost:6480/admin?cmd=routes&path=xxx&method=xxx&app=xxx'
	gapp.RegisterAdmin("routes", "get route list, params: path=xxxx app=xxx", m.onGetRoute)
	// curl 'http://localhost:6480/admin?cmd=tradingroute&uid=xxx'
	gapp.RegisterAdmin("tradingroute", "get route by uid", m.onGetTradingRoute)
	// curl 'http://localhost:6480/admin?cmd=tradingroute_clear&mode=xxx&uid=xxx&scope=xxx'
	gapp.RegisterAdmin("tradingroute_clear", "clear routing, [mode=routing,instances,all], [userId]", m.onClearTradingRoute)
	// curl 'http://localhost:6480/admin?cmd=get_account_type&uid=xxx'
	gapp.RegisterAdmin("get_account_type", "get account type by uid", m.onGetAccountType)
}

func (m *adminMgr) onPing(args gapp.AdminArgs) (interface{}, error) {
	return map[string]string{"pong": time.Now().String()}, nil
}

// 查询路由信息,可以指定path,app_module过滤
func (m *adminMgr) onGetRoute(args gapp.AdminArgs) (interface{}, error) {
	if m.routeMgr == nil {
		return nil, fmt.Errorf("empty route mgr")
	}

	appModule := args.GetStringBy("app")
	path := args.GetStringBy("path")
	method := args.GetStringBy("method")
	routes := m.routeMgr.Routes()
	if path == "" && appModule == "" {
		// 返回全部
		return routes, nil
	}

	if appModule == "" && path != "" && !strings.ContainsAny(path, "?*") {
		// 精准匹配,同时返回的routes是有序的,可以查看排序是否正确
		// 如果method为空,则查询所有method
		if method == "" {
			res := make([]*core.Route, 0)
			methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodConnect, http.MethodOptions}
			for _, mth := range methods {
				routes := m.routeMgr.FindRoutes(mth, path)
				if routes != nil {
					res = append(res, routes.GetItems()...)
				}
			}
			return res, nil
		} else {
			routes := m.routeMgr.FindRoutes(method, path)
			return routes.GetItems(), nil
		}
	}

	// 支持模糊匹配,但routes结果不保证和运行时顺序一致
	res := make(map[string][]*core.Route, len(routes))
	for _, r := range routes {
		if appModule != "" {
			if ok := wildcard.Match(appModule, r.AppKey); !ok {
				continue
			}
		}

		if path != "" {
			if ok := wildcard.Match(path, r.Path); !ok {
				continue
			}
		}
		res[r.Path] = append(res[r.Path], r)
	}

	return res, nil
}

func (m *adminMgr) onGetTradingRoute(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	if uid == 0 {
		uid = args.GetInt64By("uid")
	}
	if uid == 0 {
		return "", fmt.Errorf("invalid uid")
	}

	res, err := tradingroute.GetRouting().GetEndpoint(context.Background(), &tradingroute.GetRouteRequest{UserId: uid})
	return res, err
}

func (m *adminMgr) onClearTradingRoute(args gapp.AdminArgs) (interface{}, error) {
	mode := args.GetStringBy("mode")
	userId := args.GetInt64By("uid")
	scope := args.GetStringBy("scope")
	switch mode {
	case "user_routing":
		tradingroute.GetRouting().ClearRoutingByUser(userId, scope)
		return nil, nil
	case "routings":
		tradingroute.GetRouting().ClearRoutings()
		return nil, nil
	case "instances":
		result := tradingroute.GetRouting().ClearInstances()
		return result, nil
	case "all":
		tradingroute.GetRouting().ClearRoutings()
		result := tradingroute.GetRouting().ClearInstances()
		return result, nil
	default:
		return nil, fmt.Errorf("not support: %v", mode)
	}
}

func (m *adminMgr) onGetAccountType(args gapp.AdminArgs) (interface{}, error) {
	userId := args.GetInt64By("uid")
	if userId == 0 {
		userId = args.GetInt64At(0)
	}

	if userId == 0 {
		return nil, fmt.Errorf("need uid")
	}

	accountType, err := getAccountTypeByUID(context.Background(), userId)
	if err != nil {
		return nil, err
	}

	return map[string]string{"account_type": accountType.String()}, nil
}
