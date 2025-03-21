package nacos

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer/dispatcher"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	bn "code.bydev.io/frameworks/byone/core/discov/nacos"
	n1 "code.bydev.io/frameworks/byone/core/nacos"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/model"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"bgw/pkg/registry"
	"bgw/pkg/remoting/nacos"
)

const (
	allClusterNameKey = "ALL"
)

var (
	// 16 would be enough. We won't use concurrentMap because in most cases, there are no race condition
	instanceMap = make(map[string]registry.ServiceDiscovery, 16)
	initLock    sync.RWMutex
)

// nolint
type nacosServiceDiscovery struct {
	ctx       context.Context
	namespace string
	group     string

	// namingClient is the NacosCfg' namingClient
	namingClient gnacos.NamingClient
	// cache registry instances
	registryInstances []registry.ServiceInstance
	// cache watch service name
	watchServices sync.Map // serviceName -> struct{}{}
	once          sync.Once

	dispatcher observer.EventDispatcher
}

// NewNacosServiceDiscovery will create new service discovery instance
func NewNacosServiceDiscovery(ctx context.Context, namespace, group string) (registry.ServiceDiscovery, error) {
	initLock.RLock()
	instance, ok := instanceMap[namespace+group]
	if ok {
		initLock.RUnlock()
		return instance, nil
	}
	initLock.RUnlock()

	initLock.Lock()
	defer initLock.Unlock()
	instance, ok = instanceMap[namespace+group]
	if ok {
		return instance, nil
	}

	cfg, err := nacos.GetNacosConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("nacos.NewConfig error, namespace:%s : %w", namespace, err)
	}
	client, err := gnacos.NewNamingClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("NewNamingClient error, namespace:%s : %w", namespace, err)
	}

	newInstance := &nacosServiceDiscovery{
		ctx:               ctx,
		namespace:         namespace,
		group:             group,
		namingClient:      client,
		registryInstances: []registry.ServiceInstance{},
		dispatcher:        dispatcher.NewDirectEventDispatcher(ctx),
	}
	instanceMap[namespace+group] = newInstance
	return newInstance, nil
}

// Destroy will close the service discovery.
// Actually, it only marks the naming namingClient as null and then return
func (n *nacosServiceDiscovery) Destroy() error {
	for _, inst := range n.registryInstances {
		err := n.Unregister(inst)
		glog.Info(n.ctx, "Unregister nacos instance", glog.Any("instance", inst))
		if err != nil {
			glog.Error(n.ctx, "Unregister nacos instance error", glog.Any("instance", inst), glog.String("error", err.Error()))
		}
	}
	n.namingClient.Close()
	n.dispatcher.RemoveAllEventListeners()
	return nil
}

// Register will register the service to nacos
func (n *nacosServiceDiscovery) Register(instance registry.ServiceInstance) error {
	ins := n.toRegisterInstance(instance)
	ok, err := n.namingClient.RegisterInstance(ins)
	if err != nil || !ok {
		return fmt.Errorf("could not register the instance: %s, err: %w", instance.GetServiceName(), err)
	}
	n.registryInstances = append(n.registryInstances, instance)
	glog.Info(n.ctx, "Register service success", glog.Any("instance", instance))
	return nil
}

// Update will update the information
// However, because nacos client doesn't support the update API,
// so we should unregister the instance and then register it again.
// the error handling is hard to implement
func (n *nacosServiceDiscovery) Update(instance registry.ServiceInstance) error {
	err := n.Unregister(instance)
	if err != nil {
		return fmt.Errorf("unregister err: %w", err)
	}
	return n.Register(instance)
}

// Unregister will unregister the instance
func (n *nacosServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	ok, err := n.namingClient.DeregisterInstance(n.toDeregisterInstance(instance))
	if err != nil || !ok {
		return fmt.Errorf("could not unregister the instance: %s, err: %w", instance.GetServiceName(), err)
	}
	glog.Info(n.ctx, "Unregister service success", glog.Any("instance", instance))
	return nil
}

// GetInstances will return the instances of serviceName and the group
func (n *nacosServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	instances, err := n.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   n.group,
	})
	if err != nil {
		glog.Error(n.ctx, "SelectAllInstances error",
			glog.String("name", serviceName),
			glog.String("namespace", n.namespace),
			glog.String("group", n.group),
			glog.String("error", err.Error()))
		return make([]registry.ServiceInstance, 0)
	}
	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		// ignore Ephemeral instance unhealthy
		if ins.Weight <= 0 || ins.Ephemeral && !ins.Healthy {
			continue
		}

		res = append(res, n.buildServiceInstance(ins))
	}
	return res
}

// AddListener will add a listener
func (n *nacosServiceDiscovery) AddListener(listener registry.ServiceListener) error {
	for _, t := range listener.GetServiceNames().Values() {
		if t == nil {
			continue
		}

		service := t.(registry.ServiceMeta)
		if service.Name == "" || service.Namespace != n.namespace || service.Group != n.group {
			continue
		}
		_, loaded := n.watchServices.LoadOrStore(service.Name, struct{}{})
		if loaded {
			continue
		}
		n.once.Do(func() {
			n.dispatcher.AddEventListener(listener)
		})

		err := n.namingClient.Subscribe(&vo.SubscribeParam{
			ServiceName: service.Name,
			GroupName:   n.group,
			Clusters:    []string{allClusterNameKey},
			SubscribeCallback: func(services []model.Instance, err error) {
				if err != nil {
					glog.Info(n.ctx, "Could not handle the subscribe notification because the err is not nil", glog.String("service", service.String()),
						glog.String("namespace", n.namespace), glog.String("group", n.group), glog.String("error", err.Error()))
				}
				glog.Info(n.ctx, "SubscribeCallback hit", glog.String("service", service.String()), glog.String("namespace", n.namespace),
					glog.String("group", n.group), glog.Int64("len", int64(len(services))), glog.Any("services", services))
				instances := make([]registry.ServiceInstance, 0, len(services))
				for _, ins := range services {
					// ignore Ephemeral instance unhealthy
					if ins.Weight <= 0 || ins.Ephemeral && !ins.Healthy {
						continue
					}
					instances = append(instances, n.buildServiceInstance(ins))
				}

				e := n.DispatchEventForInstances(service, instances)
				if e != nil {
					glog.Error(n.ctx, "Dispatching event got exception", glog.String("service", service.String()),
						glog.String("namespace", n.namespace), glog.String("group", n.group), glog.String("error", err.Error()))
				}
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DispatchEventByServiceName will dispatch the event for the service with the service name
func (n *nacosServiceDiscovery) DispatchEventByServiceName(service registry.ServiceMeta) error {
	return n.DispatchEventForInstances(service, n.GetInstances(service.Name))
}

// DispatchEventForInstances will dispatch the event to those instances
func (n *nacosServiceDiscovery) DispatchEventForInstances(service registry.ServiceMeta, instances []registry.ServiceInstance) error {
	return n.DispatchEvent(registry.NewServiceInstancesChangedEvent(service, instances))
}

// DispatchEvent will dispatch the event
func (n *nacosServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return n.dispatcher.Dispatch(event)
}

// toRegisterInstance convert the ServiceInstance to RegisterInstanceParam
// the Ephemeral will be true
func (n *nacosServiceDiscovery) toRegisterInstance(instance registry.ServiceInstance) vo.RegisterInstanceParam {
	metadata := instance.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string, 1)
	}
	metadata["create_time"] = cast.ToString(time.Now().UnixNano() / 1e6)
	metadata["language"] = "golang"
	az := env.AvailableZoneID()
	if az != "" {
		metadata["az"] = az
	}
	w := instance.GetWeight()
	if w <= 0 {
		w = 100
	}
	return vo.RegisterInstanceParam{
		ServiceName: instance.GetServiceName(),
		Ip:          instance.GetHost(),
		Port:        uint64(instance.GetPort()),
		Metadata:    metadata,
		// We must specify the weight since Java nacos namingClient will ignore the instance whose weight is 0
		Weight:      float64(w),
		Enable:      true,
		Healthy:     true,
		GroupName:   n.group,
		Ephemeral:   true,
		ClusterName: instance.GetCluster(),
	}
}

// toDeregisterInstance will convert the ServiceInstance to DeregisterInstanceParam
func (n *nacosServiceDiscovery) toDeregisterInstance(instance registry.ServiceInstance) vo.DeregisterInstanceParam {
	return vo.DeregisterInstanceParam{
		ServiceName: instance.GetServiceName(),
		Ip:          instance.GetHost(),
		Port:        uint64(instance.GetPort()),
		GroupName:   n.group,
		Ephemeral:   true,
	}
}

func (n *nacosServiceDiscovery) buildServiceInstance(ins model.Instance) *registry.DefaultServiceInstance {
	instance := &registry.DefaultServiceInstance{
		ID:          ins.Ip + ":" + strconv.Itoa(int(ins.Port)),
		ServiceName: ins.ServiceName,
		Host:        ins.Ip,
		Port:        int(ins.Port),
		Enable:      ins.Enable,
		Healthy:     ins.Healthy,
		Weight:      int64(ins.Weight),
		Metadata:    ins.Metadata,
		Cluster:     ins.ClusterName,
		Address: map[string]string{
			constant.HttpProtocol: buildAddress(constant.HttpProtocol, ins.Ip, int(ins.Port), ins.Metadata),
			constant.GrpcProtocol: buildAddress(constant.GrpcProtocol, ins.Ip, int(ins.Port), ins.Metadata),
		},
	}
	return instance
}

// buildAddress build discovery addr
func buildAddress(protocol, host string, port int, metadatas map[string]string) (addr string) {
	switch protocol {
	case constant.HttpProtocol:
		if pp := metadatas["http_port"]; pp != "" {
			p := cast.ToInt(pp)
			if p > 0 {
				return host + ":" + pp
			}
		}
	case constant.GrpcProtocol:
		if pp := metadatas["gRPC_port"]; pp != "" {
			p := cast.ToInt(pp)
			if p > 0 {
				return host + ":" + pp
			}
		}
	}
	if port > 0 {
		return host + ":" + strconv.Itoa(port)
	}
	return host
}

const (
	httpPort       = "http_port"
	createTime     = "create_time"
	lastModifyTime = "last_modify_time"
	language       = "language"
	az             = "az"
	cloudName      = "cloud_name"
	tgwGroup       = "TRAFFIC_GATEWAY"
	cluster        = "cluster"
)

// BuildRegister build register
func BuildRegister(serverPrefix string, port int, conf config.ServiceRegistry) (*bn.Register, error) {
	if !conf.Enable {
		glog.Info(context.TODO(), "http Server skip register to nacos")
		return nil, nil
	}

	glog.Info(context.TODO(), "Server buildRegister nacos")
	namespace := config.GetNamespace()
	if env.IsProduction() {
		namespace = constant.DEFAULT_NAMESPACE
	}
	gcfg, err := nacos.GetNacosConfig(namespace)
	if err != nil {
		glog.Error(context.TODO(), "rnacos.GetNacosConfig error", glog.String("namespace", namespace), glog.NamedError("err", err))
		return nil, err
	}
	glog.Info(context.TODO(), "Server GetNacosConfig ok", glog.String("namespace", namespace))

	cfg := n1.NacosConf{
		NamespaceId:          namespace,
		AppName:              env.AppName(),
		CacheDir:             gcfg.Client.CacheDir,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		Username:             gcfg.Client.Username,
		Password:             gcfg.Client.Password,
		LogDir:               gcfg.Client.LogDir,
		LogLevel:             gcfg.Client.LogLevel,
	}
	for _, serverConfig := range gcfg.Server {
		sc := n1.ServerConfig{Address: serverConfig.IpAddr + ":" + cast.ToString(serverConfig.Port)}
		cfg.ServerConfigs = append(cfg.ServerConfigs, sc)
	}
	cc, err := cfg.BuildNamingClient()
	if err != nil {
		glog.Error(context.TODO(), "BuildNamingClient error", glog.String("namespace", namespace), glog.NamedError("err", err))
		return nil, err
	}
	glog.Info(context.TODO(), "Server BuildNamingClient ok", glog.String("namespace", namespace))

	// bgw-fbu-siteapi
	// bgws-ins
	clusterName := conf.ServiceName
	if clusterName == "" {
		if env.IsProduction() {
			glog.Error(context.TODO(), "get service_name error", glog.String("namespace", namespace))
			return nil, fmt.Errorf("empty nacos server name")
		}
		glog.Info(context.TODO(), "service_name is empty", glog.String("namespace", namespace))
		clusterName = env.ProjectEnvName() // use env name in test
	}
	fullServer := serverPrefix + "-" + clusterName
	glog.Info(context.TODO(), "service_name info", glog.String("namespace", namespace), glog.String("name", fullServer))

	now := cast.ToString(time.Now().UnixMilli())

	md := map[string]string{
		httpPort:       cast.ToString(port),
		createTime:     now,
		lastModifyTime: now,
		language:       "golang",
		cluster:        clusterName,
	}
	azz := env.AvailableZoneID()
	if azz != "" {
		md[az] = azz
	}
	cn := env.CloudProvider()
	if cn != "" {
		md[cloudName] = cn
	}
	n, err := bn.NewRegister(cc, fullServer, fmt.Sprintf("%s:%d", nets.GetLocalIP(), port), tgwGroup, bn.WithMetadata(md))
	if err != nil {
		glog.Error(context.TODO(), "NewRegister error", glog.String("namespace", namespace), glog.NamedError("err", err))
		return nil, err
	}

	return n, nil
}
