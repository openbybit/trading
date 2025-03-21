package registry

import (
	"context"
	"strconv"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

// ServiceInstance is the model class of an instance of a service, which is used for service registration and discovery.
type ServiceInstance interface {
	// GetID will return this instance's id. It should be unique.
	GetID() string

	// GetServiceName will return the serviceName
	GetServiceName() string

	// GetHost will return the hostname
	GetHost() string

	// GetPort will return the port.
	GetPort() int

	// IsEnable will return the enable status of this instance
	IsEnable() bool

	// IsHealthy will return the value represent the instance whether healthy or not
	IsHealthy() bool

	// GetWeight will return the value represent the instance weight
	GetWeight() int64

	// GetMetadata will return the metadata
	GetMetadata() Metadata

	// GetEndPoints
	GetEndPoints() []*Endpoint

	// Copy
	Copy(endpoint *Endpoint) ServiceInstance

	// GetAddress
	GetAddress(protocol string) string

	GetCluster() string
}

type ServiceMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Group     string `json:"group,omitempty"`
}

func (s ServiceMeta) String() string {
	return s.Name + "-" + s.Namespace + "-" + s.Group
}

func (s ServiceMeta) GetName() string {
	name := s.Name
	if s.Namespace != "" && s.Namespace != constant.DEFAULT_NAMESPACE {
		name += "-" + s.Namespace
	}
	if s.Group != "" && s.Group != constant.DEFAULT_GROUP {
		name += "-" + s.Group
	}
	return name
}

// nolint
type Endpoint struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// DefaultServiceInstance the default implementation of ServiceInstance
// or change the ServiceInstance to be struct???
type DefaultServiceInstance struct {
	ID          string
	ServiceName string
	Host        string
	Port        int
	Enable      bool
	Healthy     bool
	Weight      int64
	Metadata    Metadata
	Address     map[string]string // multi protocol
	Cluster     string
}

// GetID will return this instance's id. It should be unique.
func (d *DefaultServiceInstance) GetID() string {
	if d.ID != "" {
		return d.ID
	}
	if d.Port <= 0 {
		d.ID = d.Host
	} else {
		d.ID = d.Host + ":" + strconv.Itoa(d.Port)
	}
	return d.ID
}

// GetServiceName will return the serviceName
func (d *DefaultServiceInstance) GetServiceName() string {
	return d.ServiceName
}

// GetHost will return the hostname
func (d *DefaultServiceInstance) GetHost() string {
	return d.Host
}

// GetPort will return the port.
func (d *DefaultServiceInstance) GetPort() int {
	return d.Port
}

// IsEnable will return the enable status of this instance
func (d *DefaultServiceInstance) IsEnable() bool {
	return d.Enable
}

// IsHealthy will return the value represent the instance whether healthy or not
func (d *DefaultServiceInstance) IsHealthy() bool {
	return d.Healthy
}

// GetAddress will return the ip:Port
func (d *DefaultServiceInstance) GetAddress(protocol string) string {
	if protocol == constant.GrpcProtoBuffer {
		protocol = constant.GrpcProtocol
	}
	if addr, ok := d.Address[protocol]; ok {
		return addr
	}
	if d.Port <= 0 {
		return d.Host
	}
	return d.Host + ":" + strconv.Itoa(d.Port)
}

// GetEndPoints get end points from metadata
func (d *DefaultServiceInstance) GetEndPoints() []*Endpoint {
	rawEndpoints := d.Metadata[constant.SERVICE_INSTANCE_ENDPOINTS]
	if len(rawEndpoints) == 0 {
		return nil
	}
	var endpoints []*Endpoint
	err := util.JsonUnmarshalString(rawEndpoints, &endpoints)
	if err != nil {
		glog.Error(context.Background(), "json Unmarshal err", glog.String("rawEndpoints", rawEndpoints), glog.String("error", err.Error()))
		return nil
	}
	return endpoints
}

// Copy return a instance with different port
func (d *DefaultServiceInstance) Copy(endpoint *Endpoint) ServiceInstance {
	dn := &DefaultServiceInstance{
		ID:          d.GetID(),
		ServiceName: d.ServiceName,
		Host:        d.Host,
		Port:        endpoint.Port,
		Enable:      d.Enable,
		Healthy:     d.Healthy,
		Weight:      d.Weight,
		Metadata:    d.Metadata,
	}
	return dn
}

// GetMetadata will return the metadata, it will never return nil
func (d *DefaultServiceInstance) GetMetadata() Metadata {
	if d.Metadata == nil {
		d.Metadata = make(Metadata)
	}
	return d.Metadata
}

func (d *DefaultServiceInstance) GetWeight() int64 {
	if d.Weight > 0 {
		return d.Weight
	}
	return d.GetMetadata().GetWeight()
}

func (d *DefaultServiceInstance) GetCluster() string {
	return d.Cluster
}
