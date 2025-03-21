package gnacos

import (
	"sync"
	"sync/atomic"

	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients/config_client"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"
)

var configureClients = configPool{clients: map[string]*configClient{}}

// ConfigClient is the interface of nacos config client
type ConfigClient interface {
	config_client.IConfigClient
	Client() config_client.IConfigClient
	Valid() bool
	Close()
}

// NewConfigClient create a new nacos config client
func NewConfigClient(conf *Config) (cli ConfigClient, err error) {
	if !conf.Share {
		return newConfigClient(conf)
	}
	configureClients.Lock()
	defer configureClients.Unlock()
	client := configureClients.clients[conf.Name]
	if client == nil {
		client, err = newConfigClient(conf)
	}

	if client != nil {
		client.activeCount++
		configureClients.clients[conf.Name] = client
	}

	return client, err
}

func newConfigClient(conf *Config) (*configClient, error) {
	client, err := clients.NewConfigClient(vo.NacosClientParam{ClientConfig: &conf.Client, ServerConfigs: conf.Server})
	if err != nil {
		return nil, err
	}

	res := &configClient{IConfigClient: client, name: conf.Name, share: conf.Share, valid: 1, activeCount: 0}
	if conf.Share {
		configureClients.clients[conf.Name] = res
	}

	return res, nil
}

type configPool struct {
	sync.RWMutex
	clients map[string]*configClient
}

type configClient struct {
	config_client.IConfigClient
	name        string
	share       bool
	valid       uint32
	activeCount uint32
}

func (n *configClient) Client() config_client.IConfigClient {
	return n.IConfigClient
}

func (n *configClient) Valid() bool {
	return atomic.LoadUint32(&n.valid) == 1
}

func (n *configClient) Close() {
	if atomic.CompareAndSwapUint32(&n.valid, 1, 0) {
		if n.share {
			namingClients.Lock()
			n.activeCount--
			if n.activeCount == 0 {
				n.IConfigClient = nil
				delete(namingClients.clients, n.name)
			}
			namingClients.Unlock()
		} else {
			n.IConfigClient = nil
		}
	}
}
