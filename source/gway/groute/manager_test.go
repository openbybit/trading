package groute

import (
	"errors"
	"strings"
	"testing"
)

func TestIsSameCategory(t *testing.T) {
	r := []string{"linear,inverse"}
	if !isSameCategory(r, r) {
		t.Error("should same")
	}

	if !isSameCategory(r, nil) {
		t.Error("should same")
	}
}

func TestIsSameAccount(t *testing.T) {
	r := AccountTypeNormal
	if !isSameAccount(r, r) {
		t.Error("should same")
	}

	if !isSameAccount(r, AccountTypeAll) {
		t.Error("should same")
	}
}

func TestOrderedList(t *testing.T) {
	l := OrderedList([]string{"linear", "inverse"})
	l.Sort()
	if !l.Contains("linear") {
		t.Error("should contain")
	}
	if !l.ContainsAny(l) {
		t.Error("should contain")
	}
	if l.Contains("x") {
		t.Error("should not contain")
	}
}

// 测试插入顺序
func TestInsertOrder(t *testing.T) {
	path := "/path1"
	t.Run("category order", func(t *testing.T) {
		list := []*Route{
			{Path: path, Type: ROUTE_TYPE_CATEGORY},
			{Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")},
			{Path: path, Type: ROUTE_TYPE_ALL_IN_ONE},
		}
		rs := newRoutes(nil)
		for _, r := range list {
			if err := r.build(); err != nil {
				t.Errorf("build fail, err: %v", err)
			}
			rs.insert(r)
		}
		if !rs.GetFirst().IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
			t.Error("first should be all-in-one")
		}
		if !rs.GetLast().IsCatetoryDefault() {
			t.Error("last should be default")
		}
	})

	t.Run("category ordered1", func(t *testing.T) {
		list := []*Route{
			{Path: path, Type: ROUTE_TYPE_ALL_IN_ONE},
			{Path: path, Type: ROUTE_TYPE_CATEGORY},
		}
		rs := newRoutes(nil)
		for _, r := range list {
			if err := r.build(); err != nil {
				t.Errorf("build fail, err: %v", err)
			}
			rs.insert(r)
		}
		if !rs.GetFirst().IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
			t.Error("first should be all-in-one")
		}
		if !rs.GetLast().IsCatetoryDefault() {
			t.Error("last should be default")
		}
	})

	t.Run("account_type order", func(t *testing.T) {
		list := []*Route{
			{Path: path, Type: ROUTE_TYPE_ACCOUNT_TYPE},
			{Path: path, Type: ROUTE_TYPE_ACCOUNT_TYPE, Values: toValues("c1")},
			{Path: path, Type: ROUTE_TYPE_ALL_IN_ONE},
		}
		rs := newRoutes(nil)
		for _, r := range list {
			if err := r.build(); err != nil {
				t.Errorf("build fail, err: %v", err)
			}
			rs.insert(r)
		}
		if !rs.GetFirst().IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
			t.Error("first should be all-in-one")
		}
		if !rs.GetLast().IsCatetoryDefault() {
			t.Error("last should be default")
		}
	})

	t.Run("default handler order", func(t *testing.T) {
		list := []*Route{
			{Path: path, Type: ROUTE_TYPE_DEFAULT},
			{Path: path, Type: ROUTE_TYPE_ALL_IN_ONE},
		}
		rs := newRoutes(nil)
		for _, r := range list {
			if err := r.build(); err != nil {
				t.Errorf("build fail, err: %v", err)
			}
			rs.insert(r)
		}

		if !rs.GetFirst().IsRouteType(ROUTE_TYPE_ALL_IN_ONE) {
			t.Error("first should be all-in-one")
		}
		if !rs.GetLast().IsRouteType(ROUTE_TYPE_DEFAULT) {
			t.Error("last should be default handler")
		}
	})
}

func TestCanInsert(t *testing.T) {
	path := "/path1"
	appKey1 := "app1"
	appKey2 := "app2"
	t.Run("test all-in-one exclusive", func(t *testing.T) {
		r1 := &Route{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_ALL_IN_ONE}
		r2 := &Route{AppKey: appKey2, Path: path, Type: ROUTE_TYPE_ALL_IN_ONE}
		rs := newRoutes(r1)
		if err := rs.canInsert(r2); err == nil {
			t.Errorf("should return error")
		}
	})

	t.Run("test default handler exclusive", func(t *testing.T) {
		r1 := &Route{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_ALL_IN_ONE}
		r2 := &Route{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_DEFAULT}
		r3 := &Route{AppKey: appKey2, Path: path, Type: ROUTE_TYPE_CATEGORY}
		r4 := &Route{Path: path, Type: ROUTE_TYPE_CATEGORY}
		r5 := &Route{Path: path, Type: ROUTE_TYPE_ACCOUNT_TYPE}

		rs := newRoutes(r1)
		if err := rs.insert(r2); err != nil {
			t.Errorf("insert default fail")
		}
		if err := rs.insert(r3); err == nil {
			t.Errorf("default should be exclusive")
		}
		if err := rs.insert(r4); err == nil {
			t.Errorf("default should be exclusive")
		}
		if err := rs.insert(r5); err == nil {
			t.Errorf("default should be exclusive")
		}
	})

	t.Run("test category default exclusive", func(t *testing.T) {
		r1 := &Route{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_CATEGORY}
		r2 := &Route{AppKey: appKey2, Path: path, Type: ROUTE_TYPE_CATEGORY}
		rs := newRoutes(r1)
		if err := rs.insert(r2); err == nil {
			t.Errorf("should return err")
		}
	})

	t.Run("test type different", func(t *testing.T) {
		r1 := &Route{Path: path, Type: ROUTE_TYPE_ACCOUNT_TYPE, Account: AccountTypeNormal}
		r2 := &Route{Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")}

		rs := newRoutes(r1)
		if err := rs.insert(r2); err == nil {
			t.Errorf("should return err")
		}
	})

	t.Run("test category conflict", func(t *testing.T) {
		r1 := &Route{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")}
		r2 := &Route{AppKey: appKey2, Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")}

		rs := newRoutes(r1)
		if err := rs.insert(r2); err == nil {
		}
	})
}

func toValues(x string) []string {
	res := strings.Split(x, ",")
	for i, t := range res {
		res[i] = strings.TrimSpace(t)
	}

	return res
}

func TestInsert(t *testing.T) {
	appKey1 := "app1.module1"
	appKey2 := "app1.module2"

	routes := []*Route{
		{AppKey: appKey1, Path: "/a/d", Type: ROUTE_TYPE_DEFAULT},
		// category
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: nil},
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1,c2")},
		{AppKey: appKey1, Path: "/a/b", Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")},
		// category and account
		{AppKey: appKey1, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: toValues("linear,inverse"), Account: AccountTypeNormal},
		{AppKey: appKey1, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnified}, // 支持category为空的情况
		{AppKey: appKey1, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnknown}, // default handler
		// prefix routes
		// 支持static和/*共存
		{AppKey: appKey1, Path: "/a/*"},
		// 这两种模糊匹配能共存,但选择时有优先级
		{AppKey: appKey1, Path: "/p/*"},
		{AppKey: appKey1, Path: "/p/c/*"},
		//
		{AppKey: appKey1, Path: "/q/c/*"},
	}

	b := newBucket()
	for _, r := range routes {
		// 首次可以正常添加
		if err := b.Insert(r); err != nil {
			t.Error("insert fail", err, r)
		}
		// 再次重复添加
		if err := b.Insert(r); err == nil {
			t.Error("should return err", err, r)
		}
	}

	// path含有/后缀,重复path,会返回错误
	if err := b.Insert(&Route{AppKey: appKey1, Path: "/a/d/", Type: ROUTE_TYPE_DEFAULT}); err == nil {
		t.Errorf("should ignore error, err=%v", err)
	}

	// 其他异常case
	invalidRoutes := []*Route{
		// 禁止注册root path
		{AppKey: appKey1, Path: "", Type: ROUTE_TYPE_DEFAULT},
		{AppKey: appKey1, Path: "/", Type: ROUTE_TYPE_DEFAULT},
		// 重复handler, appKey不一样
		{AppKey: appKey2, Path: "/a/d", Type: ROUTE_TYPE_DEFAULT},
		// 重复category
		{AppKey: appKey2, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toValues("c1")},
		// 重复的category和account
		{AppKey: appKey1, Path: "/a/b/c", Type: ROUTE_TYPE_CATEGORY, Values: toValues("linear"), Account: AccountTypeNormal},
		// 类型不一致
		{AppKey: appKey2, Path: "/a/d", Type: ROUTE_TYPE_CATEGORY, Values: nil},
		// prefix
		{AppKey: appKey1, Path: "/a/*"},
		{AppKey: appKey1, Path: "/p/c/*"},
	}

	for _, r := range invalidRoutes {
		if err := b.Insert(r); err == nil {
			t.Error("should return err", err, r)
		}
	}
}

func TestInsertCategory(t *testing.T) {
	// 验证插入category时,需要保证如果有default一定是最后一个
	// 并且AccountTypeFlag 有一个设置了account则为true
	r := Routes{}
	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY, Values: toValues("linear")})
	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY, Values: toValues("inverse")})

	if r.accountTypeFlag == true {
		t.Errorf("account type flag should be false")
	}

	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY, Account: AccountTypeUnified})
	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY})
	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY, Values: toValues("spot"), Account: AccountTypeNormal})
	r.insert(&Route{Type: ROUTE_TYPE_CATEGORY, Values: toValues("option"), Account: AccountTypeNormal})

	if !r.GetLast().IsCatetoryDefault() {
		t.Errorf("invalid default category")
	}

	if !r.accountTypeFlag {
		t.Errorf("account type flag should be true")
	}
}

func TestInsertCategoryAccount(t *testing.T) {
	// 测试category和account各种组合情况,是否会报错
	appKey1 := "app1.module1"
	// appKey2 := "app1.module2"
	routes := []*Route{
		// category and account
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toValues("linear,spot"), Account: AccountTypeNormal},
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnified}, // 支持category为空的情况
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnknown}, // default handler
	}

	b := newBucket()
	for _, r := range routes {
		if err := b.Insert(r); err != nil {
			t.Errorf("insert route fail, err=%v, route=%v", err, r)
		}
	}

	// 测试冲突的情况
	invalidRoutes := []*Route{
		{AppKey: appKey1, Path: "/a", Type: ROUTE_TYPE_CATEGORY, Values: toValues("linear"), Account: AccountTypeUnified},
	}

	for _, r := range invalidRoutes {
		if err := b.Insert(r); err == nil {
			t.Errorf("should error, %v", r)
		}
	}
}

func TestInsertAllInOne(t *testing.T) {
	// 测试category和account各种组合情况,是否会报错
	appKey1 := "option.giga"
	appKey2 := "app1.module2"
	appKey3 := "trading"
	path := "/a"

	routes := []*Route{
		{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("option"), Account: AccountTypeUnknown},
		{AppKey: appKey1, Path: path, Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnknown},
		{AppKey: appKey2, Path: path, Type: ROUTE_TYPE_CATEGORY, Values: toValues("spot"), Account: AccountTypeUnknown},
		{AppKey: appKey3, Path: path, Type: ROUTE_TYPE_ALL_IN_ONE},
	}

	b := newBucket()
	for _, r := range routes {
		if err := b.Insert(r); err != nil {
			t.Errorf("insert route fail, err=%v, route=%v", err, r)
		}
	}
}

func TestManager(t *testing.T) {
	appKey1 := "option.giga"
	path := "/a"
	method := "GET"
	routes := []*Route{
		{AppKey: appKey1, Path: path, Method: method, Type: ROUTE_TYPE_CATEGORY, Values: toValues("option"), Account: AccountTypeUnknown},
		{AppKey: appKey1, Path: path, Method: method, Type: ROUTE_TYPE_CATEGORY, Values: nil, Account: AccountTypeUnknown},
	}

	m := NewManager()
	if err := m.Replace(appKey1, routes); err != nil {
		t.Error("should nil")
	}

	if r := m.Find(method, path); r == nil {
		t.Error("cannot find route")
	}

	if len(m.Routes()) != len(routes) {
		t.Error("invalid routes")
	}
}

func TestSuffix(t *testing.T) {
	r := &Route{
		Path: "/test/mock/dynamic_url/**/middle",
	}
	err := r.build()
	if !errors.Is(err, ErrInvalidPrefixPath) {
		t.Error(err)
	}

	r = &Route{
		Path: "/test/mock/dynamic_url/***/middle",
	}
	err = r.build()
	if !errors.Is(err, ErrInvalidPrefixPath) {
		t.Error(err)
	}

	r = &Route{
		Path: "/test/mock/dynamic_url/middle/***",
	}
	err = r.build()
	if !errors.Is(err, ErrInvalidPrefixPath) {
		t.Error(err)
	}

	r = &Route{
		Path: "/test/mock/dynamic_url/middle/*",
	}
	err = r.build()
	if err != nil || r.pathType != PATH_TYPE_PRIFIX {
		t.Error("should be nil", err, r.pathType)
	}
}

func TestError(t *testing.T) {
	t.Run("bucket error", func(t *testing.T) {
		b := bucket{}
		err := b.Insert(nil)
		if err == nil {
			t.Errorf("no panic, %v", err)
		}
	})

	t.Run("manager error", func(t *testing.T) {
		m := manager{}
		m.Init()
		m.Insert([]*Route{{Method: "invalid"}})
		m.Insert([]*Route{{Method: "GET", Path: "/"}})

		routes := []*Route{
			{Method: "GET", Path: "/aa", AppKey: "key"},
		}
		m.Insert(routes)
		m.Find("GET", "/aa")
		m.Find("invalid", "/aa")
	})

	t.Run("atomic manager", func(t *testing.T) {
		m := NewManager()
		apikey := "key"
		routes := []*Route{
			{Method: "GET", Path: "/aa", AppKey: apikey},
			{Method: "GET", Path: "/bb", AppKey: "keyb"},
		}
		m.Insert(routes)
		m.Replace(apikey, routes)
		m.Replace(apikey, []*Route{{Method: "GET", Path: "/", AppKey: apikey}})
		m.Replace(apikey, []*Route{{Method: "GET", Path: "/", AppKey: "keyb"}})
	})
}
