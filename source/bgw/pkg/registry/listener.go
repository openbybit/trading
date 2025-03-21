package registry

import (
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

//go:generate mockgen -source=listener.go -destination=../mock/listener_mock.go -package=mock
type ServiceListener interface {
	// RemoveListener remove notify listener
	RemoveListener(service ServiceMeta)
	// GetServiceNames return all listener service names
	GetServiceNames() *container.HashSet
	// Accept return true if the name is the same
	observer.ConditionalEventListener
}
