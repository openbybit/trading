package etcd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/getcd"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/registry"
	retcd "bgw/pkg/remoting/etcd"
)

var (
	// 16 would be enough. We won't use concurrentMap because in most cases, there are not race condition
	instanceMap = make(map[string]registry.ServiceDiscovery, 16)
	initLock    sync.Mutex
)

// nolint
type etcdServiceDiscovery struct {
	ctx context.Context

	// root etcd discovery root path, prefix
	root string

	// etcd raw client
	client getcd.Client

	// cache registry instances
	registryInstances []registry.ServiceInstance
}

// NewETCDServiceDiscovery will create new service discovery instance
func NewETCDServiceDiscovery(ctx context.Context, root string) (registry.ServiceDiscovery, error) {
	initLock.Lock()
	defer initLock.Unlock()

	instance, ok := instanceMap[root]
	if ok {
		return instance, nil
	}

	client, err := retcd.NewConfigClient(ctx)
	if err != nil {
		return nil, err
	}

	newInstance := &etcdServiceDiscovery{
		root:   root,
		client: client,
		ctx:    ctx,
	}

	instanceMap[root] = newInstance
	return newInstance, nil
}

func (esd *etcdServiceDiscovery) getPath(key string) string {
	return filepath.Join(esd.root, key)
}

// Destroy will close the service discovery.
// Actually, it only marks the naming namingClient as null and then return
func (esd *etcdServiceDiscovery) Destroy() error {
	for _, inst := range esd.registryInstances {
		err := esd.Unregister(inst)
		glog.Info(esd.ctx, "Unregister nacos instance", glog.Any("instance", inst))
		if err != nil {
			glog.Error(esd.ctx, "Unregister nacos instance error", glog.Any("instance", inst), glog.String("error", err.Error()))
		}
	}
	esd.client.Close()
	return nil
}

// Register will register the service to nacos
func (esd *etcdServiceDiscovery) Register(instance registry.ServiceInstance) error {
	err := esd.client.Put(esd.getPath(instance.GetServiceName()), instance.GetAddress(""))
	if err != nil {
		return fmt.Errorf("could not register the instance: %s, err: %w", instance.GetServiceName(), err)
	}
	esd.registryInstances = append(esd.registryInstances, instance)
	return nil
}

// Update will update the information
// However, because nacos client doesn't support the update API,
// so we should unregister the instance and then register it again.
// the error handling is hard to implement
func (esd *etcdServiceDiscovery) Update(instance registry.ServiceInstance) error {
	err := esd.Unregister(instance)
	if err != nil {
		return fmt.Errorf("unregister err: %w", err)
	}
	return esd.Register(instance)
}

// Unregister will unregister the instance
func (esd *etcdServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	return esd.client.Delete(esd.getPath(instance.GetServiceName()))
}

// GetInstances will return the instances of serviceName and the group
func (esd *etcdServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	kl, vl, err := esd.client.GetChildren(esd.getPath(serviceName), false)
	if err != nil {
		return nil
	}

	res := make([]registry.ServiceInstance, 0, len(kl))
	for index, key := range kl {
		addr := strings.SplitN(vl[index], ":", 2)
		if len(addr) != 2 {
			continue
		}

		res = append(res, &registry.DefaultServiceInstance{
			ID:          key,
			ServiceName: serviceName,
			Host:        addr[0],
			Port:        cast.ToInt(addr[1]),
			Enable:      true,
			Healthy:     true,
			Weight:      100,
			Metadata:    make(registry.Metadata),
		})
	}
	return res
}

// AddListener will add a listener
func (esd *etcdServiceDiscovery) AddListener(listener registry.ServiceListener) error {
	el := getcd.NewEventListener(esd.ctx, esd.client)
	el.ListenWithChildren(esd.root, listener)
	return nil
}

// DispatchEventByServiceName will dispatch the event for the service with the service name
func (esd *etcdServiceDiscovery) DispatchEventByServiceName(service registry.ServiceMeta) error {
	return nil
}

// DispatchEventForInstances will dispatch the event to those instances
func (esd *etcdServiceDiscovery) DispatchEventForInstances(service registry.ServiceMeta, instances []registry.ServiceInstance) error {
	return nil
}

// DispatchEvent will dispatch the event
func (esd *etcdServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return nil
}
