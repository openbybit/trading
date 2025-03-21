package registry

// ServiceDiscovery is the common operations of Service Discovery
//
//go:generate mockgen -source=registry.go -destination=../mock/registry_mock.go -package=mock
type ServiceDiscovery interface {
	// Destroy will destroy the service discovery.
	Destroy() error

	// Register will register an instance of ServiceInstance to registry
	Register(instance ServiceInstance) error

	// Update will update the data of the instance in registry
	Update(instance ServiceInstance) error

	// Unregister will unregister this instance from registry
	Unregister(instance ServiceInstance) error

	// GetInstances will return all service instances with serviceName
	GetInstances(serviceName string) []ServiceInstance

	// AddListener adds a new ServiceInstancesChangedListenerImpl
	AddListener(listener ServiceListener) error

	// DispatchEventByServiceName dispatches the ServiceInstancesChangedEvent to service instance whose name is serviceName
	DispatchEventByServiceName(service ServiceMeta) error

	// DispatchEventForInstances dispatches the ServiceInstancesChangedEvent to target instances
	DispatchEventForInstances(service ServiceMeta, instances []ServiceInstance) error

	// DispatchEvent dispatches the event
	DispatchEvent(event *ServiceInstancesChangedEvent) error
}
