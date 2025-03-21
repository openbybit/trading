package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"go.uber.org/atomic"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/discovery"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/filter/initializer"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/tradingroute"
)

var (
	defaultController Controller
	once              sync.Once
)

const logRegistryWatchErrMsg = "serviceRegistry.Watch error"

// GetController get controller
func GetController(ctx context.Context) Controller {
	once.Do(func() {
		defaultController = NewContoller(ctx)
	})

	return defaultController
}

// Controller is a controller interface
type Controller interface {
	Init() error
	GetRouteManager() RouteManager
	GetHandler(ctx context.Context, p RouteDataProvider) (types.Handler, error)

	fmt.Stringer
}

// controller is a controller component controller
type controller struct {
	ctx    context.Context
	inited *atomic.Bool

	versionController *versionController
	configManager     *configManager
	serviceRegistry   discovery.ServiceRegistryModule
	routeManager      RouteManager
	invoker           *invoker
	serviceListener   func(service string) error
}

// NewContoller create a new controller
func NewContoller(ctx context.Context) Controller {
	c := &controller{
		ctx:               ctx,
		inited:            atomic.NewBool(false),
		versionController: newVersionController(ctx),
		configManager:     newConfigureManager(ctx),
		serviceRegistry:   discovery.NewServiceRegistry(ctx),
		invoker:           newInvoker(ctx),
		serviceListener:   breakerMgr.OnConfigUpdate,
	}

	c.serviceRegistry.AddInsListener(breakerMgr.OnInstanceRemove)
	c.routeManager = newRouteManager(c.getRouteChain)
	return c
}

func (c *controller) GetRouteManager() RouteManager {
	return c.routeManager
}

// Init do init controller
// 1. init version controller
// 2. init config manager
// 3. init invoker
// 4. add listeners for (version control, configure, invoker, route, etc.)
// 5. load current versions data
// 6. start listen version on configure
// 7. register app handlers
func (c *controller) Init() (err error) {
	if !c.inited.CompareAndSwap(false, true) {
		return nil
	}

	if err = c.versionController.init(); err != nil {
		glog.Error(c.ctx, "version controller init error", glog.String("error", err.Error()))
		return
	}

	if err = c.configManager.init(); err != nil {
		glog.Error(c.ctx, "config manager init error", glog.String("error", err.Error()))
		return
	}

	if err = c.invoker.init(); err != nil {
		glog.Error(c.ctx, "invoker init error", glog.String("error", err.Error()))
		return
	}

	c.configManager.addListener(c)
	c.versionController.addListener(c.configManager, c.invoker)

	// start version controller listener
	if err = c.versionController.listen(); err != nil {
		glog.Error(c.ctx, "version controller listen error", glog.String("error", err.Error()))
		return
	}

	return
}

// GetHandler get handler for service
func (c *controller) GetHandler(ctx context.Context, p RouteDataProvider) (types.Handler, error) {
	route, err := c.routeManager.FindRoute(ctx, p)
	if route != nil {
		glog.Debug(ctx, "controller GetHandler",
			glog.String("path", route.Path),
			glog.String("app_key", route.AppKey),
			glog.String("route_type", route.Type.String()),
			glog.Any("category", route.Values),
			glog.String("account", route.Account.String()),
		)
		md := metadata.MDFromContext(ctx)
		md.StaticRoutePath = route.Path
		return route.Handler.(types.Handler), err
	}
	return nil, err
}

func (c *controller) String() string {
	type state struct {
		Routes    []*Route
		Instances map[string][]registry.ServiceInstance
		Registry  []interface{}
		Versions  []*AppVersion
		Config    map[string]interface{}
		Services  []string
	}

	st := state{
		Routes:    c.routeManager.Routes(),
		Instances: c.serviceRegistry.GetAllInstances(),
		Registry:  c.serviceRegistry.Services(),
		Versions:  c.versionController.Values(),
		Config:    c.configManager.cache.Items(),
	}
	st.Services, _ = c.invoker.grpcEngine.ListServices()

	return cast.UnsafeBytesToString(util.ToJSON(st))
}

// OnEvent event fired on configre change event
// 1. register watch on service
// 2. register router & reload
func (c *controller) OnEvent(event observer.Event) (err error) {
	ve, ok := event.(*configChangeEvent)
	if !ok {
		return nil
	}

	ac := ve.GetSource().(*AppConfig)
	glog.Info(c.ctx, "[service]fire event", glog.String("app", ac.Key()))

	if err := c.routeManager.Load(ac); err != nil {
		return err
	}

	for _, sc := range ac.Services {
		_ = c.serviceListener(sc.Registry)
		if err = c.serviceRegistry.Watch(c.ctx, sc.GetRegistry(sc.Group)); err != nil {
			glog.Error(c.ctx, logRegistryWatchErrMsg, glog.String("service", sc.Registry), glog.String("group", sc.Group), glog.NamedError("err", err))
			galert.Error(c.ctx, logRegistryWatchErrMsg, galert.WithField("service", sc.Registry), galert.WithField("group", sc.Group), galert.WithField("err", err))
		}

		for _, method := range sc.Methods {
			if method.GroupRouteMode != defaultOnly && method.GroupRouteMode != allToDefault {
				if err = c.serviceRegistry.Watch(c.ctx, sc.GetRegistry(demoAccountGroup)); err != nil {
					glog.Error(c.ctx, logRegistryWatchErrMsg, glog.String("service", sc.Registry), glog.String("group", demoAccountGroup), glog.NamedError("err", err))
					galert.Error(c.ctx, logRegistryWatchErrMsg, galert.WithField("service", sc.Registry), galert.WithField("group", demoAccountGroup), galert.WithField("err", err))
				}
				break
			}
		}
	}

	return
}

// GetEventType fired on configChangeEvent
func (c *controller) GetEventType() reflect.Type {
	return reflect.TypeOf(configChangeEvent{})
}

// GetPriority fired priority on configChangeEvent
func (c *controller) GetPriority() int {
	return -1
}

// getRouteChain get handler accordin by route config
func (c *controller) getRouteChain(mc *MethodConfig) (types.Handler, error) {
	chain := filter.NewChain()
	// initialize handler
	initFilter := initializer.New(mc.RouteKey(), mc.Service().Registry)
	chain.Append(initFilter)

	// add response filter
	chain, err := chain.AppendNames(filter.ResponseFilterKey)
	if err != nil {
		return nil, err
	}

	// construct filter chain of handler
	filters := mc.GetFilters()
	// glog.Info(c.ctx, "filter list", glog.String("route", mc.RouteKey().String()), glog.String("path", mc.Path), glog.Any("filters", filters))
	for _, ff := range filters {
		// add route key as first arg
		args := append([]string{mc.RouteKey().String()}, ff.GetArgs()...)
		f, err := filter.GetFilter(c.ctx, ff.Name, args...)
		if f == nil {
			return nil, fmt.Errorf("GetFilter error: %s -> %s -> %w", ff.Name, ff.Args, err)
		}
		chain.Append(f)
	}

	// construct upstream invoker
	invoke, err := c.getInvoker(mc)
	if err != nil {
		return nil, fmt.Errorf("getInvoker error: %w", err)
	}

	return chain.Finally(invoke), nil
}

// getInvoker construct upstream finally invoker for the route
// select an instance using service discovery (nacos)
// construct metadata from context
// call grpc invoke with request, and send back result for front side
func (c *controller) getInvoker(mc *MethodConfig) (types.Handler, error) {
	slectorMeta, err := ParseSelectorMeta(mc.GetSelectorMeta())
	if err != nil {
		return nil, fmt.Errorf("parse selector meta fail, err=%v", err)
	}

	selectorConf := cluster.ExtractConf{
		Registry:        mc.Service().Registry,
		Namespace:       mc.Service().Namespace,
		Group:           mc.Service().Group,
		ServiceName:     mc.Service().Name,
		MethodName:      mc.Name,
		LoadBalanceMeta: mc.GetLBMeta(),
		SelectKeys:      slectorMeta.SelectKeys,
	}

	var metas interface{}
	selector := cluster.GetSelector(c.ctx, mc.GetSelector())
	if se, ok := selector.(cluster.Extract); ok {
		me, err := se.Extract(&selectorConf)
		if err != nil {
			return nil, fmt.Errorf("getInvoker selector Extract error: %w", err)
		}
		metas = me
	}

	if st, ok := selector.(cluster.Setter); ok {
		st.SetDiscovery(c.serviceRegistry.GetInstances)
	}

	isAllInOneTrading := tradingroute.IsRoutingService(mc.Service().Registry)
	return func(ctx *types.Ctx) error {
		md := metadata.MDFromContext(ctx)

		if isAllInOneTrading {
			if md.IsDemoUID {
				return c.tradingInvoke(ctx, tradingroute.GetDemoRouting(), mc, md)
			}
			return c.tradingInvoke(ctx, tradingroute.GetRouting(), mc, md)
		}

		// inject select meta
		if se, ok := selector.(cluster.Inject); ok {
			_, err := se.Inject(ctx, metas)
			if err != nil {
				return err
			}
		}

		// select instance from registry
		group := c.getGroup(mc, md.IsDemoUID)
		md.InvokeNamespace = mc.Service().Namespace
		md.InvokeGroup = group
		server := mc.Service().GetRegistry(group)
		instances := c.serviceRegistry.GetInstances(server)

		instance, err := selector.Select(ctx, cluster.SwimLaneSelector(ctx, instances))
		if instance == nil || err != nil {
			err = fmt.Errorf("instance not found: %w", err)
			glog.Error(ctx, "instance not found", glog.Any("registry", server), glog.String("group", group),
				glog.String("path", md.Path), glog.String("error", err.Error()))
			if md.IsDemoUID {
				return berror.NewUpStreamErr(berror.UpstreamErrDemoInstanceNotFound, md.InvokeService, err.Error())
			}
			return berror.NewUpStreamErr(berror.UpstreamErrInstanceNotFound, md.InvokeService, err.Error())
		}

		md.InvokeAddr = instance.GetAddress(mc.Service().Protocol)
		md.WithContext(ctx)

		// invoke upstream
		return c.invoker.invoke(ctx, mc, md)
	}, nil
}

func (c *controller) getGroup(mc *MethodConfig, isDemoUID bool) string {
	group := mc.Service().Group
	switch mc.GroupRouteMode {
	case allToDefault, defaultOnly:
		return group
	case demoOnly:
		group = demoAccountGroup
	default: // strict
		if isDemoUID {
			group = demoAccountGroup
		}
	}
	return group
}

func (c *controller) tradingInvoke(ctx *types.Ctx, router tradingroute.Routing, mc *MethodConfig, md *metadata.Metadata) error {
	req := &tradingroute.GetRouteRequest{
		UserId: md.UID,
		Scope:  md.Route.GetAppName(ctx),
	}
	md.InvokeNamespace = router.Namespace()
	route, err := router.GetEndpoint(ctx, req)
	if err != nil {
		return err
	}

	if route.Address == "" {
		return berror.NewInterErr("empty routing address")
	}

	md.InvokeAddr = route.Address
	if err := c.invoker.invoke(ctx, mc, md); err != nil {
		glog.Error(ctx, "aio invoke fail", glog.NamedError("error", err), glog.Any("route", route), glog.Any("req", req))
		return err
	}
	return nil
}
