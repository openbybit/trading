package groute

import (
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	ErrDuplicateRoute      = errors.New("duplicate route")
	ErrInvalidRouteMgr     = errors.New("invalid route mgr")
	ErrInvalidMethod       = errors.New("invalid method")
	ErrInvalidPrefixPath   = errors.New("invalid prefix path")
	ErrInvalidRootPath     = errors.New("invalid root path")
	ErrNotSupportParamPath = errors.New("not support param path")
)

type Manager interface {
	// Replace 删除appKey下所有旧路由,并插入新路由
	// 原子操作,如果成功则全部成功,如果失败则全部失败
	Replace(appKey string, routes []*Route) error
	// Insert 插入,非并发安全,仅用于plugin
	Insert(routes []*Route) error
	// Find 通过method和path查找一组路由,由上层业务做进一步筛选
	Find(method string, path string) *Routes
	// Routes 返回所有路由
	Routes() []*Route
}

func NewManager() Manager {
	m := &atomicManager{}
	m.init()
	return m
}

func newBucket() *bucket {
	return &bucket{}
}

type bucket struct {
	tree node
}

func (rb *bucket) getRoutes(path string) *Routes {
	path = trimPath(path)
	params := Params{}
	routes, _ := rb.tree.findRoute(path, &params).(*Routes)
	return routes
}

func (rb *bucket) Insert(r *Route) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("insert panic: err: %v", e)
		}
	}()

	if err := r.build(); err != nil {
		return err
	}

	path := r.realPath
	if !r.IsPathType(PATH_TYPE_STATIC) {
		// 模糊匹配直接插入, 只能有一个handler且不能冲突
		routes := newRoutes(r)
		if err := rb.tree.addRoute(path, routes); err != nil {
			return err
		}
		return nil
	}

	// 静态path则先尝试查找,不存在则创建,存在则判断是否能直接插入
	routes := rb.getRoutes(path)
	if routes == nil || len(routes.items) == 0 || !routes.isStaticPath() {
		// 1: route不存在,或没有handler(理论上不会存在这种情况),则需要新建
		// 2: 如果是wildcard,则只能有一个handler,也需要新建
		routes = newRoutes(r)
		if err := rb.tree.addRoute(path, routes); err != nil {
			return err
		}
		return nil
	}

	return routes.insert(r)
}

type manager struct {
	buckets []*bucket // method bucket
	routes  []*Route  // all routes
}

func (m *manager) Init() {
	m.buckets = make([]*bucket, idxMethodMax)
	for i := 0; i < len(m.buckets); i++ {
		m.buckets[i] = newBucket()
	}
}

func (m *manager) Find(method string, path string) *Routes {
	methodIdx := methodIndexMap[method]
	if methodIdx == idxMethodInvaid || methodIdx >= len(m.buckets) {
		return nil
	}

	return m.buckets[methodIdx].getRoutes(path)
}

func (m *manager) Insert(routes []*Route) error {
	for _, r := range routes {
		methodIdx := methodIndexMap[r.Method]
		if methodIdx == idxMethodInvaid {
			return fmt.Errorf("invalid method, route: %v", r.String())
		}

		if err := m.buckets[methodIdx].Insert(r); err != nil {
			return fmt.Errorf("insert fail, err:%w, route: %s", err, r.String())
		}

		m.routes = append(m.routes, r)
	}

	return nil
}

type atomicManager struct {
	mgr atomic.Value
}

func (am *atomicManager) init() {
	newMgr := &manager{}
	newMgr.Init()
	am.mgr.Store(newMgr)
}

func (am *atomicManager) getMgr() *manager {
	return am.mgr.Load().(*manager)
}

func (am *atomicManager) Routes() []*Route {
	return am.getMgr().routes
}

func (am *atomicManager) Find(method string, path string) *Routes {
	return am.getMgr().Find(method, path)
}

func (am *atomicManager) Insert(routes []*Route) error {
	return am.getMgr().Insert(routes)
}

func (am *atomicManager) Replace(appKey string, routes []*Route) error {
	oldMgr := am.getMgr()
	oldRoutes := make([]*Route, 0, len(oldMgr.routes))
	for _, r := range oldMgr.routes {
		if r.AppKey == appKey {
			continue
		}
		oldRoutes = append(oldRoutes, r)
	}

	newMgr := &manager{}
	newMgr.Init()
	if err := newMgr.Insert(oldRoutes); err != nil {
		return err
	}

	if err := newMgr.Insert(routes); err != nil {
		return err
	}

	am.mgr.Store(newMgr)

	return nil
}
