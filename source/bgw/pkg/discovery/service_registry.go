package discovery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/registry/dns"
	retcd "bgw/pkg/registry/etcd"
	"bgw/pkg/registry/nacos"
)

var (
	_ registry.ServiceListener = &serviceRegistry{}
)

type ServiceRegistryModule = *serviceRegistry

var (
	globalServiceRegistry *serviceRegistry
	registryOnce          sync.Once
)

type insListener func(service string, ins []string) error

// serviceRegistry for instance registry
type serviceRegistry struct {
	lock         sync.RWMutex
	ctx          context.Context
	serviceNames *container.HashSet // ServiceMeta: service names, namespace, group
	allInstances sync.Map           // ServiceMeta -> instances   map[ServiceMeta][]ServiceInstance
	insListeners []insListener
}

func NewServiceRegistry(ctx context.Context) ServiceRegistryModule {
	registryOnce.Do(func() {
		s := &serviceRegistry{
			ctx:          ctx,
			serviceNames: container.NewSet(),
		}
		globalServiceRegistry = s
	})
	return globalServiceRegistry
}

// Watch on the url (including protocol, service key)
func (s *serviceRegistry) Watch(ctx context.Context, url *common.URL) error {
	if url == nil {
		return fmt.Errorf("serviceRegistry Watch nil URL")
	}
	// !NOTE: get service name in addr, a trick for extension
	// TODO: parse full url registry
	r := s.getRegistry(url)
	if r == nil {
		return errors.New("getRegistry fail, " + url.String())
	}

	serviceMeta := registry.ServiceMeta{
		Name:      url.Addr,
		Namespace: url.GetParam(constant.NAMESPACE_KEY, constant.DEFAULT_NAMESPACE),
		Group:     url.GetParam(constant.GROUP_KEY, constant.DEFAULT_GROUP),
	}
	s.lock.Lock()
	// ignore invalid service
	if s.serviceNames.Contains(serviceMeta) {
		s.lock.Unlock()
		return nil
	}

	s.serviceNames.Add(serviceMeta)
	s.lock.Unlock()

	if err := r.AddListener(s); err != nil {
		galert.Error(ctx, fmt.Sprintf("service registry:%s, AddListener error: %s", serviceMeta.String(), err))
		return err
	}

	return nil
}

// getRegistry nacos only right now
// extend other protocol on url
func (s *serviceRegistry) getRegistry(url *common.URL) registry.ServiceDiscovery {
	var (
		r   registry.ServiceDiscovery
		err error
	)

	switch url.Protocol {
	case constant.NacosProtocol:
		r, err = nacos.NewNacosServiceDiscovery(
			s.ctx,
			url.GetParam(constant.NAMESPACE_KEY, constant.NACOS_DEFAULT_NAMESPACE),
			url.GetParam(constant.GROUP_KEY, constant.NACOS_DEFAULT_GROUP))
	case constant.DNSProtocol:
		r = dns.NewDNSDiscovery(s.ctx)
	case constant.EtcdProtocol:
		r, err = retcd.NewETCDServiceDiscovery(s.ctx, url.GetPath())
	default:
	}
	if err != nil {
		glog.Error(s.ctx, "getRegistry error", glog.String("protocol", url.Protocol), glog.String("error", err.Error()))
		return nil
	}

	return r
}

func (s *serviceRegistry) Services() []interface{} {
	return s.serviceNames.Values()
}

// GetInstances get instance from cache, otherwise from registry
func (s *serviceRegistry) GetInstances(url *common.URL) (ins []registry.ServiceInstance) {
	if url == nil {
		return nil
	}
	service := registry.ServiceMeta{
		Name:      url.Addr,
		Namespace: url.GetParam(constant.NAMESPACE_KEY, constant.DEFAULT_NAMESPACE),
		Group:     url.GetParam(constant.GROUP_KEY, constant.DEFAULT_GROUP),
	}

	if ins = s.getInstances(service); ins != nil {
		return ins
	}

	r := s.getRegistry(url)
	if r == nil {
		return make([]registry.ServiceInstance, 0)
	}
	ins = r.GetInstances(url.Addr)

	if len(ins) > 0 {
		glog.Info(s.ctx, "GetInstances hit", glog.String("service", service.String()), glog.String("namespace", service.Namespace),
			glog.String("group", service.Group), glog.Any("instances", ins))
		s.allInstances.Store(service, ins)
	}

	return ins
}

// GetInstancesNoCache get instance from registry no cache
func (s *serviceRegistry) GetInstancesNoCache(url *common.URL) (ins []registry.ServiceInstance) {
	if url == nil {
		return nil
	}
	r := s.getRegistry(url)
	if r == nil {
		return make([]registry.ServiceInstance, 0)
	}
	ins = r.GetInstances(url.Addr)
	return ins
}

// getInstances from cache
func (s *serviceRegistry) getInstances(service registry.ServiceMeta) []registry.ServiceInstance {
	v, ok := s.allInstances.Load(service)
	if ok {
		ins, ok := v.([]registry.ServiceInstance)
		if ok && len(ins) > 0 {
			return ins
		}
	}
	return nil
}

func (s *serviceRegistry) GetAllInstances() map[string][]registry.ServiceInstance {
	m := map[string][]registry.ServiceInstance{}
	s.allInstances.Range(func(key, value interface{}) bool {
		service, ok := key.(registry.ServiceMeta)
		if !ok {
			return true
		}
		ins, ok := value.([]registry.ServiceInstance)
		if !ok {
			return true
		}
		m[service.String()] = ins
		return true
	})
	return m
}

// GetServiceNames registered services
func (s *serviceRegistry) GetServiceNames() *container.HashSet {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceNames
}

// RemoveListener remove listener and related instances
func (s *serviceRegistry) RemoveListener(service registry.ServiceMeta) {
	s.lock.Lock()
	s.serviceNames.Remove(service)
	s.allInstances.Delete(service)
	s.lock.Unlock()
}

func (s *serviceRegistry) AddInsListener(f insListener) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.insListeners = append(s.insListeners, f)
}

// OnEvent fired on service discovery
// put cached instances
func (s *serviceRegistry) OnEvent(e observer.Event) error {
	ce, ok := e.(*registry.ServiceInstancesChangedEvent)
	if !ok {
		return nil
	}

	gmetric.SetDefaultGauge(float64(len(ce.Instances)), "service", ce.Service.GetName())
	glog.Info(s.ctx, "SubscribeCallback OnEvent", glog.String("service", ce.Service.String()), glog.String("namespace", ce.Service.Namespace),
		glog.String("group", ce.Service.Group), glog.Int64("len", int64(len(ce.Instances))), glog.Any("instances", ce.Instances))

	oldIns := s.getInstances(ce.Service)
	if len(ce.Instances) > 0 {
		s.allInstances.Store(ce.Service, ce.Instances)
	}

	// compare with old instances, close old conn
	removes := s.compareAndClose(oldIns, ce.Instances)

	s.lock.RLock()
	listeners := s.insListeners
	s.lock.RUnlock()
	for _, l := range listeners {
		_ = l(ce.Service.Name, removes)
	}

	return nil
}

func (s *serviceRegistry) compareAndClose(oldIns, newIns []registry.ServiceInstance) (removes []string) {
	defer func() {
		pool.Remove(removes...)
	}()

	if len(newIns) == 0 {
		for _, in := range oldIns {
			removes = append(removes, in.GetAddress(constant.GrpcProtocol))
		}
		return
	}

	// event do conn clean
	hit := make(map[string]struct{})
	for _, in := range oldIns {
		hit[in.GetAddress(constant.GrpcProtocol)] = struct{}{}
	}

	for _, in := range newIns {
		delete(hit, in.GetAddress(constant.GrpcProtocol))
	}

	if len(hit) == 0 {
		return
	}
	for addr := range hit {
		removes = append(removes, addr)
	}
	return
}

// GetEventType fired on ServiceInstancesChangedEvent
func (s *serviceRegistry) GetEventType() reflect.Type {
	return reflect.TypeOf(registry.ServiceInstancesChangedEvent{})
}

func (s *serviceRegistry) GetPriority() int {
	return 0
}

// Accept just registered services and the specified ServiceInstancesChangedEvent
// otherwise ignore and drop listener (see: observer.dispatcher)
func (s *serviceRegistry) Accept(e observer.Event) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if ce, ok := e.(*registry.ServiceInstancesChangedEvent); ok {
		return s.serviceNames.Contains(ce.Service)
	}
	return false
}
