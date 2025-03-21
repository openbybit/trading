package dns

import (
	"context"

	"bgw/pkg/registry"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer/dispatcher"
)

type DNSDiscovery struct {
	ctx context.Context

	// dns resover
	resolver string

	// cache registry instances
	registryInstances []registry.ServiceInstance

	dispatcher observer.EventDispatcher
}

func NewDNSDiscovery(ctx context.Context, resolver ...string) *DNSDiscovery {
	dd := &DNSDiscovery{
		ctx:               ctx,
		registryInstances: make([]registry.ServiceInstance, 0),
		dispatcher:        dispatcher.NewDirectEventDispatcher(ctx),
	}
	if len(resolver) > 0 {
		dd.resolver = resolver[0]
	}

	return dd
}

func (dns *DNSDiscovery) Destroy() error {
	return nil
}

// Register will register an instance of ServiceInstance to registry
func (dns *DNSDiscovery) Register(instance registry.ServiceInstance) error {
	return nil
}

// Update will update the data of the instance in registry
func (dns *DNSDiscovery) Update(instance registry.ServiceInstance) error {
	return nil
}

// Unregister will unregister this instance from registry
func (dns *DNSDiscovery) Unregister(instance registry.ServiceInstance) error {
	return nil
}

// GetInstances will return all service instances with serviceName
func (dns *DNSDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	if len(dns.resolver) == 0 {
		return []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Host: serviceName,
			},
		}
	}
	ls := NewLookup(dns.resolver)
	ins, err := ls.LookupSRV(serviceName)
	if err != nil {
		return nil
	}

	svcs := make([]registry.ServiceInstance, 0)
	for _, i := range ins {
		svcs = append(svcs, &registry.DefaultServiceInstance{
			Host:   i.Target,
			Port:   int(i.Port),
			Weight: int64(i.Weight),
		})
	}
	return svcs
}

// AddListener adds a new ServiceInstancesChangedListenerImpl
func (dns *DNSDiscovery) AddListener(listener registry.ServiceListener) error {
	dns.dispatcher.AddEventListener(listener)
	return nil
}

// DispatchEventByServiceName dispatches the ServiceInstancesChangedEvent to service instance whose name is serviceName
func (dns *DNSDiscovery) DispatchEventByServiceName(service registry.ServiceMeta) error {
	return nil
}

// DispatchEventForInstances dispatches the ServiceInstancesChangedEvent to target instances
func (dns *DNSDiscovery) DispatchEventForInstances(service registry.ServiceMeta, instances []registry.ServiceInstance) error {
	return nil
}

// DispatchEvent dispatches the event
func (dns *DNSDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return dns.dispatcher.Dispatch(event)
}
