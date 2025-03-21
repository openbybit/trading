package getcd

import (
	"time"
)

const (
	// EtcdConfigKey for etcd config center
	EtcdConfigKey = "etcd"
	// EtcdDiscoveryKey for etcd backed service discovery
	EtcdDiscoveryKey = "etcd"
	// ConnDelay connection delay
	ConnDelay = 3
	// MaxFailTimes max failure times
	MaxFailTimes = 15
	// RegistryETCDV3Client client Name
	RegistryETCDV3Client = "etcd registry"
	// MetadataETCDV3Client client Name
	MetadataETCDV3Client = "etcd metadata"
)

// Options client configuration
type Options struct {
	// Name etcd server name
	Name string
	// Endpoints etcd endpoints
	Endpoints []string
	// Client etcd client
	Client *Client
	// Timeout timeout
	Timeout time.Duration
	// Heartbeat second
	Heartbeat int
	// Username username
	Username string
	// Password password
	Password string
}

// Option will define a function of handling Options
type Option func(*Options)

// WithEndpoints sets etcd client endpoints
func WithEndpoints(endpoints ...string) Option {
	return func(opt *Options) {
		opt.Endpoints = endpoints
	}
}

// WithName sets etcd client name
func WithName(name string) Option {
	return func(opt *Options) {
		opt.Name = name
	}
}

// WithTimeout sets etcd client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(opt *Options) {
		opt.Timeout = timeout
	}
}

// WithHeartbeat sets etcd client heartbeat
func WithHeartbeat(heartbeat int) Option {
	return func(opt *Options) {
		opt.Heartbeat = heartbeat
	}
}

// WithUsername sets etcd client username
func WithUsername(username string) Option {
	return func(opt *Options) {
		opt.Username = username
	}
}

// WithPassword sets etcd client password
func WithPassword(password string) Option {
	return func(opt *Options) {
		opt.Password = password
	}
}
