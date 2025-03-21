package gnacos

import (
	"sync"
	"sync/atomic"

	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients/naming_client"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"
)

var namingClients = namingPool{clients: map[string]*namingClient{}}

// NamingClient is the interface of nacos naming client
type NamingClient interface {
	naming_client.INamingClient
	Client() naming_client.INamingClient
	Valid() bool
	Close() error
}

// NewNamingClient create a new nacos naming client
func NewNamingClient(conf *Config) (cli NamingClient, err error) {
	if !conf.Share {
		return newNamingClient(conf)
	}

	namingClients.Lock()
	defer namingClients.Unlock()
	client, ok := namingClients.clients[conf.Name]
	if ok {
		client.activeCount++
		return client, nil
	}

	client, err = newNamingClient(conf)
	if err != nil {
		return nil, err
	}
	client.activeCount++
	namingClients.clients[conf.Name] = client

	return client, err
}

func newNamingClient(conf *Config) (*namingClient, error) {
	if conf.Client.Username == "" {
		conf.Client.Username = DEFAULT_USERNAME
	}
	if conf.Client.Password == "" {
		conf.Client.Password = DEFAULT_PASSWORD
	}

	client, err := clients.NewNamingClient(vo.NacosClientParam{ClientConfig: &conf.Client, ServerConfigs: conf.Server})
	if err != nil {
		return nil, err
	}
	res := &namingClient{INamingClient: client, name: conf.Name, share: conf.Share, valid: 1, activeCount: 0}

	return res, nil
}

type namingPool struct {
	sync.RWMutex
	clients map[string]*namingClient
}

type namingClient struct {
	naming_client.INamingClient
	name        string
	share       bool
	valid       uint32
	activeCount uint32
}

func (n *namingClient) Client() naming_client.INamingClient {
	return n.INamingClient
}

func (n *namingClient) Valid() bool {
	return atomic.LoadUint32(&n.valid) == 1
}

func (n *namingClient) Close() error {
	if atomic.CompareAndSwapUint32(&n.valid, 1, 0) {
		if n.share {
			namingClients.Lock()
			n.activeCount--
			if n.activeCount == 0 {
				n.INamingClient = nil
				delete(namingClients.clients, n.name)
			}
			namingClients.Unlock()
		} else {
			n.INamingClient = nil
		}
	}

	return nil
}
