package core

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/groute"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/service/tradingroute"
	"bgw/pkg/test"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
)

func testHandler(rctx *fasthttp.RequestCtx) error {
	return nil
}

func TestParseRouteTypeBySelectorMeta(t *testing.T) {
	res := parseRouteTypeBySelectorMeta(&SelectorMeta{})
	assert.Equal(t, groute.ROUTE_TYPE_DEFAULT, res)

	res = parseRouteTypeBySelectorMeta(&SelectorMeta{
		Groups: &GroupMeta{},
	})
	assert.Equal(t, groute.ROUTE_TYPE_CATEGORY, res)

	res = parseRouteTypeBySelectorMeta(&SelectorMeta{
		RouteType: "all_in_one",
	})
	assert.Equal(t, groute.ROUTE_TYPE_ALL_IN_ONE, res)
}

func TestIsHitAllInOne(t *testing.T) {
	rm := &routeManager{
		mgr: &mockMgr{},
	}
	b, err := rm.isHitAllInOne(context.Background(), &Route{}, &mockProvider{}, &userinfo{})
	assert.Equal(t, false, b)
	assert.NoError(t, err)

	b, err = rm.isHitAllInOne(context.Background(), &Route{Values: groute.OrderedList{"1", "2"}}, &mockProvider{vals: make([][]byte, 2)}, &userinfo{})
	assert.Equal(t, false, b)
	assert.EqualError(t, err, "Parameter error: request have multiple categories")

	rm = &routeManager{
		mgr: &mockMgr{},
	}
	b, err = rm.isHitAllInOne(context.Background(), &Route{
		Values: groute.OrderedList{"1", "2"},
	}, &mockProvider{}, &userinfo{})
	assert.Equal(t, false, b)
	assert.NoError(t, err)

	rm = &routeManager{
		mgr: &mockMgr{},
	}
	b, err = rm.isHitAllInOne(context.Background(), &Route{}, &mockProvider{}, &userinfo{
		accountType: constant.AccountTypeNormal,
	})
	assert.Equal(t, false, b)
	assert.NoError(t, err)

	rm = &routeManager{
		mgr: &mockMgr{},
	}
	b, err = rm.isHitAllInOne(context.Background(), &Route{}, &mockProvider{}, &userinfo{
		accountType: constant.AccountTypeUnified,
	})
	assert.Equal(t, false, b)
	assert.NoError(t, err)

	rm = &routeManager{
		mgr: &mockMgr{},
	}
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(tradingroute.GetRouting()), "IsAioUser", func(ctx context.Context, userId int64) (bool, error) {
		return true, nil
	})
	b, err = rm.isHitAllInOne(context.Background(), &Route{}, &mockProvider{}, &userinfo{
		accountType: constant.AccountTypeUnified,
	})
	assert.Equal(t, true, b)
	assert.NoError(t, err)
	p.Reset()

	rm = &routeManager{
		mgr: &mockMgr{},
	}
	p = gomonkey.ApplyPrivateMethod(reflect.TypeOf(tradingroute.GetRouting()), "IsAioUser", func(ctx context.Context, userId int64) (bool, error) {
		return false, errors.New("sss")
	})
	b, err = rm.isHitAllInOne(context.Background(), &Route{}, &mockProvider{}, &userinfo{
		accountType: constant.AccountTypeUnified,
	})
	assert.Equal(t, false, b)
	assert.EqualError(t, err, "sss")
	p.Reset()

}

func TestFindRout(t *testing.T) {
	rm := &routeManager{
		mgr: &mockMgr{},
	}
	r, err := rm.FindRoute(context.Background(), &mockProvider{})
	assert.Nil(t, r)
	assert.Nil(t, err)

	rm = &routeManager{
		mgr: &mockMgr{
			route: &groute.Routes{},
		},
	}
	r, err = rm.FindRoute(context.Background(), &mockProvider{})
	assert.Nil(t, r)
	assert.Nil(t, err)

	rm = &routeManager{
		mgr: &mockMgr{
			route: &groute.Routes{},
		},
	}
	patch := gomonkey.ApplyFunc((*groute.Route).IsRouteType, func(*groute.Route, groute.RouteType) bool { return true })
	defer patch.Reset()
	patch2 := gomonkey.ApplyFunc((*groute.Route).IsPathType, func(*groute.Route, groute.PathType) bool { return true })
	defer patch2.Reset()
	patch3 := gomonkey.ApplyFunc((*groute.Routes).GetItems, func(*groute.Routes) []*groute.Route { return []*groute.Route{{Values: groute.OrderedList{"1", "2"}}} })
	defer patch3.Reset()
	patch4 := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(a, b string) {})
	defer patch4.Reset()

	r, err = rm.FindRoute(context.Background(), &mockProvider{vals: make([][]byte, 2)})
	assert.Nil(t, r)
	assert.NotNil(t, err)
}

func TestGetAccountType(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	p := NewCtxRouteDataProvider(rctx, nil, nil)

	a, c := p.GetAccountType(100)
	assert.Equal(t, constant.AccountTypeUnknown, a)
	assert.Equal(t, "not support", c.Error())

	p = NewCtxRouteDataProvider(rctx, nil, func(ctx context.Context, uid int64) (constant.AccountType, error) {
		return constant.AccountTypeUnifiedTrading, nil
	})
	a, c = p.GetAccountType(100)
	assert.Equal(t, constant.AccountTypeUnifiedTrading, a)
	assert.Equal(t, nil, c)
}

func TestGetUserID(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	p := NewCtxRouteDataProvider(rctx, nil, nil)

	a, b, c := p.GetUserID()
	assert.Equal(t, int64(0), a)
	assert.Equal(t, false, b)
	assert.Equal(t, "not support", c.Error())

	p = NewCtxRouteDataProvider(rctx, func(ctx *types.Ctx) (int64, bool, error) {
		return 100, true, nil
	}, nil)
	a, b, c = p.GetUserID()
	assert.Equal(t, int64(100), a)
	assert.Equal(t, true, b)
	assert.Equal(t, nil, c)
}

func TestBuildGroups(t *testing.T) {
	rm := &routeManager{}
	rs, err := rm.buildGroups(&Route{}, &SelectorMeta{
		Groups: &GroupMeta{},
	})
	assert.Equal(t, "empty group config, :", err.Error())
	assert.Nil(t, rs)

	rs, err = rm.buildGroups(&Route{}, &SelectorMeta{
		Groups: &GroupMeta{
			Routes: []RouteTuple{{}, {
				Category: "111",
				Account:  "1112",
				Default:  "",
			}},
		},
	})
	assert.Equal(t, "unknown account type, 1112", err.Error())
	assert.Nil(t, rs)

	rs, err = rm.buildGroups(&Route{}, &SelectorMeta{
		Groups: &GroupMeta{
			Routes: []RouteTuple{{}, {
				Category: "111",
				Account:  "normal",
				Default:  "",
			}},
			Default: true,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rs))
	assert.Equal(t, groute.AccountTypeNormal, rs[0].Account)
	assert.Equal(t, "111", rs[0].Values[0])
	assert.Equal(t, groute.AccountTypeAll, rs[1].Account)

}

func TestFind(t *testing.T) {
	ROUTE_TYPE_DEFAULT := groute.ROUTE_TYPE_DEFAULT
	ROUTE_TYPE_CATEGORY := groute.ROUTE_TYPE_CATEGORY

	mth := "GET"
	appKey1 := "app1.module1"
	// appKey2 := "app1.module2"
	// 测试正常插入
	routes := []*Route{
		{AppKey: appKey1, Method: mth, Path: "/a/d", Type: ROUTE_TYPE_DEFAULT},
		// category
		{AppKey: appKey1, Method: mth, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: nil},
		{AppKey: appKey1, Method: mth, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toCategories("c2")},
		{AppKey: appKey1, Method: mth, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toCategories("c1")},
		{AppKey: appKey1, Method: mth, Path: "/a/b", Type: ROUTE_TYPE_CATEGORY, Values: toCategories("c1")},
		// category and account
		{AppKey: appKey1, Method: mth, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: toCategories("linear"), Account: groute.AccountTypeNormal},
		{AppKey: appKey1, Method: mth, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: toCategories("inverse"), Account: groute.AccountTypeNormal},
		{AppKey: appKey1, Method: mth, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: groute.AccountTypeUnified}, // 支持category为空的情况
		{AppKey: appKey1, Method: mth, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: groute.AccountTypeUnknown}, // default handler
		// prefix routes
		// 支持static和/*共存
		{AppKey: appKey1, Method: mth, Path: "/a/*"},
		// 这两种模糊匹配能共存,但选择时有优先级
		{AppKey: appKey1, Method: mth, Path: "/p/*"},
		{AppKey: appKey1, Method: mth, Path: "/p/c/*"},
		//
		{AppKey: appKey1, Method: mth, Path: "/q/c/*"},
	}

	mgr := &routeManager{
		mgr: groute.NewManager(),
		creator: func(mc *MethodConfig) (types.Handler, error) {
			return testHandler, nil
		},
	}

	if err := mgr.mgr.Insert(routes); err != nil {
		t.Errorf("insert fail, %v", err)
	}

	// test find
	type findCase struct {
		Path     string
		Key      string
		Category string
		Account  groute.AccountType
	}

	// 可以找到的case
	findCases := []findCase{
		// 精准匹配
		{Path: "/a/d", Category: ""},                     // 精确匹配default
		{Path: "/a", Key: keyCategory, Category: "c1"},   // 精准匹配category
		{Path: "/a", Key: keyCategory, Category: "c2"},   // 精准匹配category
		{Path: "/a", Key: keyCategory, Category: ""},     // 匹配category空
		{Path: "/a", Key: keyCategory, Category: "c3"},   // category都没有, 会匹配空
		{Path: "/a/b", Key: keyCategory, Category: "c1"}, // 精准匹配category
		// category and account
		{Path: "/a/b/c", Key: keyCategory, Category: "linear", Account: groute.AccountTypeNormal},         //
		{Path: "/a/b/c", Key: keyCategory, Category: "inverse", Account: groute.AccountTypeNormal},        //
		{Path: "/a/b/c", Key: keyCategory, Category: "linear", Account: groute.AccountTypeUnifiedTrading}, // 命中category为空的account为unified情况
		{Path: "/a/b/c", Key: keyCategory, Category: "spot", Account: groute.AccountTypeNormal},           // 命中default handler
		// 前缀匹配
		{Path: "/p/any"},
		{Path: "/p/c/any"},
		{Path: "/q/c/any"},
	}

	for _, item := range findCases {
		route, _ := mgr.FindRoute(context.Background(), newMockProvider("GET", item.Path, item.Category, constant.AccountType(item.Account)))

		if route == nil {
			t.Errorf("not find route, %v", item)
		} else {
			switch {
			case route.IsPathType(groute.PATH_TYPE_STATIC):
				// 静态匹配,匹配结果要一致
				if route.Path != item.Path {
					t.Errorf("can't find route, route=%v, item=%v", route, item)
				}

				// 判断是否是category
				if route.Type == ROUTE_TYPE_CATEGORY {
					isHitCategory := len(route.Values) == 0 || route.Values.Contains(item.Category)
					isHitAccount := route.Account == groute.AccountTypeAll || route.Account.Is(item.Account)
					if !(isHitCategory && isHitAccount) {
						t.Errorf("can't find route, route=%v, item=%v", route, item)
					}
				}

			case route.IsPathType(groute.PATH_TYPE_PRIFIX):
				// 必须包含前缀
				if !strings.HasPrefix(item.Path, trimPrefixPath(route.Path)) {
					t.Errorf("can't find route, route=%v, item=%v", route, item)
				} else {
					// prefix 会优先选择,需要肉眼去识别
					t.Logf("%v --> %v", item.Path, route.Path)
				}
			}
		}
	}
}

// trimPrefixPath 去除/*后的path
func trimPrefixPath(p string) string {
	idx := strings.LastIndex(p, "/*")
	if idx != -1 {
		return p[:idx]
	}

	return p
}

func newMockProvider(method string, path string, category string, account constant.AccountType) RouteDataProvider {
	if method == "" {
		method = "GET"
	}
	return &mockProvider{
		method:  method,
		path:    path,
		value:   category,
		account: account,
	}
}

type mockProvider struct {
	method  string
	path    string
	value   string
	vals    [][]byte
	account constant.AccountType
}

func (m *mockProvider) GetMethod() string {
	return m.method
}

func (m *mockProvider) GetPath() string {
	return m.path
}

func (m *mockProvider) GetValue(key string) string {
	return m.value
}

func (m *mockProvider) GetValues(key string) [][]byte {
	return m.vals
}

func (m *mockProvider) GetUserID() (int64, bool, error) {
	return 0, false, nil
}

func (m *mockProvider) GetAccountType(uid int64) (constant.AccountType, error) {
	return m.account, nil
}

type mockMgr struct {
	replaceErr error
	insertErr  error
	route      *groute.Routes
	routes     []*groute.Route
}

func (m *mockMgr) Replace(appKey string, routes []*groute.Route) error {
	return m.replaceErr
}

func (m *mockMgr) Insert(routes []*groute.Route) error {
	return m.insertErr
}

func (m *mockMgr) Find(method string, path string) *groute.Routes {
	return m.route
}

func (m *mockMgr) Routes() []*groute.Route {
	return m.routes
}
