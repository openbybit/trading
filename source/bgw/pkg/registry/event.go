package registry

import (
	"fmt"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

var _ observer.Event = &ServiceInstancesChangedEvent{}

// ServiceInstancesChangedEvent represents service instances make some changing
type ServiceInstancesChangedEvent struct {
	observer.BaseEvent
	Service   ServiceMeta
	Instances []ServiceInstance
}

// String return the description of the event
func (s *ServiceInstancesChangedEvent) String() string {
	return fmt.Sprintf("ServiceInstancesChangedEvent[source=%s]", s.Service.String())
}

// NewServiceInstancesChangedEvent will create the ServiceInstanceChangedEvent instance
func NewServiceInstancesChangedEvent(service ServiceMeta, instances []ServiceInstance) *ServiceInstancesChangedEvent {
	return &ServiceInstancesChangedEvent{
		Service:   service,
		Instances: instances,
		BaseEvent: observer.BaseEvent{
			Source:    service.Name,
			Timestamp: time.Now(),
		},
	}
}
