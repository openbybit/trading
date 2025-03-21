package nacos

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/clients/config_client"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/common/constant"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"
)

const (
	schemeKey = "nacos"

	defaultGroup    = "DEFAULT_GROUP"
	defaultUsername = "bybit-nacos"
	defaultPassword = "bybit-nacos"

	nacosDevAddr = "nacos-test.efficiency.ww5sawfyut0k.bitsvc.io:8848"
	nacosSitAddr = "nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848"
)

const (
	envKeyNacosAddress  = "NACOS_CONFIG_ADDRESS"
	envKeyNacosUsername = "NACOS_USERNAME"
	envKeyNacosPassword = "NACOS_PASSWORD"

	envKeyMyEnvName        = "MY_ENV_NAME"
	envKeyMyProjectName    = "MY_PROJECT_NAME"
	envKeyMyProjectEnvName = "MY_PROJECT_ENV_NAME"
)

func init() {
	gconfig.Register(schemeKey, New)
}

var (
	clientMap = make(map[string]config_client.IConfigClient)
	clientMux sync.Mutex
)

// NewWithClient 通过原生client创建
func NewWithClient(client config_client.IConfigClient, group string) gconfig.Configure {
	return &nacosConfigure{
		client: client,
		group:  group,
	}
}

// New 通过url创建configure,使用时通过使用默认的url就可以了
// 如果url是非法的,即中不包含nacos://,则认为是namespace,测试环境通常只需要指定namespace
// 内部会通过host+namespace+username作为唯一key复用nacos client
// 生成环境namespace为public,测试环境通过namespace区分不同集群,并使用MY_PROJECT_ENV_NAME初始化namespace
// 测试环境有两个address,dev环境的addr已经废弃,但可以使用use_dev_nacos标识强制指定使用此地址
func New(address string) (gconfig.Configure, error) {
	addrURL, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	group := addrURL.Query().Get("group")
	client, err := newNacosClient(addrURL)
	if err != nil {
		return nil, fmt.Errorf("create nacos client fail,addr=%v ,err=%v", address, err)
	}

	return &nacosConfigure{
		client: client,
		group:  group,
	}, nil
}

func parseAddress(address string) (*url.URL, error) {
	address = strings.TrimSpace(address)
	if address != "" {
		if !strings.Contains(address, "://") {
			// namespace
			if !strings.ContainsAny(address, ":&=/") {
				address = fmt.Sprintf("%s://?namespace=%s", schemeKey, address)
			} else {
				address = fmt.Sprintf("%s://%s", schemeKey, address)
			}
		}

		u, err := url.Parse(address)
		if err != nil {
			return nil, fmt.Errorf("parse nacos address fail, address=%v, err=%v", address, err)
		}
		return u, nil
	} else {
		return &url.URL{Scheme: schemeKey}, nil
	}
}

func isProd() bool {
	envName := os.Getenv(envKeyMyEnvName)
	return envName == "testnet" || envName == "mainnet"
}

// https://github.com/nacos-group/nacos-sdk-go
// url格式: nacos://username:password@host:port?namespace=public&log_dir=/tmp/nacos/log&cache_dir=/tmp/nacos/cache&log_level=debug
// http://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848
// 相关配置文档: https://uponly.larksuite.com/wiki/wikusePZSq11OC1duSBU5m6krxd
func newNacosClient(u *url.URL) (config_client.IConfigClient, error) {
	q := u.Query()

	if u.Host == "" {
		u.Host = os.Getenv(envKeyNacosAddress)

		if !isProd() && q.Has("use_dev_nacos") {
			u.Host = nacosDevAddr
		}

		// 本地使用sit环境配置
		if u.Host == "" {
			u.Host = nacosSitAddr
		}
	}

	namespace := q.Get("namespace")
	if namespace == "" && !isProd() {
		namespace = os.Getenv(envKeyMyProjectEnvName)
	}

	cacheDir := q.Get("cache_dir")
	if cacheDir == "" {
		cacheDir = "/tmp/nacos/cache"
	}
	logDir := q.Get("log_dir")
	if logDir == "" {
		logDir = "/tmp/nacos/log"
	}
	logLevel := q.Get("log_level")
	if logLevel == "" {
		logLevel = "info"
	}

	username := u.User.Username()
	password, _ := u.User.Password()
	if username == "" {
		username = os.Getenv(envKeyNacosUsername)
		if username == "" {
			username = defaultUsername
		}
	}
	if password == "" {
		password = os.Getenv(envKeyNacosPassword)
		if password == "" {
			password = defaultPassword
		}
	}

	key := fmt.Sprintf("%s:%s:%s", u.Host, namespace, username)
	clientMux.Lock()
	defer clientMux.Unlock()
	if cli, ok := clientMap[key]; ok {
		return cli, nil
	}

	serverConfigs := make([]constant.ServerConfig, 0)
	for _, addr := range strings.Split(u.Host, ",") {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, constant.ServerConfig{IpAddr: ip, Port: uint64(port)})
	}

	params := vo.NacosClientParam{
		ServerConfigs: serverConfigs,
		ClientConfig: &constant.ClientConfig{
			NamespaceId:         namespace,
			AppName:             os.Getenv(envKeyMyProjectName),
			Username:            username,
			Password:            password,
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
			CacheDir:            cacheDir,
			LogDir:              logDir,
			LogLevel:            logLevel,
		},
	}
	if params.ClientConfig.NamespaceId == "public" { // fix: overwrite nacos namespace on public
		params.ClientConfig.NamespaceId = ""
	}

	cli, err := clients.NewConfigClient(params)
	if err != nil {
		return nil, fmt.Errorf("create nacos config client fail, url: %v, err:%v", u.String(), err)
	}

	clientMap[key] = cli

	return cli, nil
}

// nacos 配置中心
// 1: Put,Delete,Listen为异步接口,也就是说Put完数据后Get是查询不到的
// 2: 数据不存在时, Get会返回空数据,但不会报错
// 3: 删除和新增key时, Listen可以监听到数据,删除时收到数据为空
// 4: 相同key只能被监听一次,第二次监听会被忽略
type nacosConfigure struct {
	client    config_client.IConfigClient
	group     string
	listeners sync.Map
}

func (c *nacosConfigure) Get(ctx context.Context, key string, opts ...gconfig.Option) (string, error) {
	o := gconfig.Options{Group: c.group}
	o.Init(opts...)

	data, err := c.client.GetConfig(vo.ConfigParam{
		DataId: key,
		Group:  o.Group,
	})

	if err != nil {
		return "", fmt.Errorf("nacos get fail, group: %v, err: %v", o.Group, err)
	}

	return data, nil
}

func (c *nacosConfigure) Put(ctx context.Context, key string, value string, opts ...gconfig.Option) error {
	o := gconfig.Options{Group: c.group}
	o.Init(opts...)

	ok, err := c.client.PublishConfig(vo.ConfigParam{
		DataId:  key,
		Content: value,
		Group:   o.Group,
	})
	if err != nil {
		return fmt.Errorf("nacos put fail, group: %v, err: %v", o.Group, err)
	}
	if !ok {
		return gconfig.ErrPutFailure
	}

	return nil
}

func (c *nacosConfigure) Delete(ctx context.Context, key string, opts ...gconfig.Option) error {
	o := gconfig.Options{Group: c.group}
	o.Init(opts...)

	param := vo.ConfigParam{
		DataId: key,
		Group:  o.Group,
	}

	ok, err := c.client.DeleteConfig(param)
	if err != nil {
		return fmt.Errorf("nacos delete fail, group: %v, err: %v", o.Group, err)
	}
	if !ok {
		return gconfig.ErrDelFailure
	}

	return nil
}

func (c *nacosConfigure) Listen(ctx context.Context, key string, listener gconfig.Listener, opts ...gconfig.Option) error {
	o := gconfig.Options{Group: c.group}
	o.Init(opts...)
	if o.Group == "" {
		o.Group = defaultGroup
	}

	var err error
	if o.ForceGet {
		// 强制同步Get一次, 保证能够立马获取到数据
		var firstData string
		firstData, err = c.client.GetConfig(vo.ConfigParam{
			DataId: key,
			Group:  o.Group,
		})

		if err != nil {
			return fmt.Errorf("nacos get fail, group: %v, dataId: %v, err: %w", o.Group, key, err)
		}

		if firstData != "" {
			listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: key, Value: firstData})
		}

		checkFirst := true
		err = c.doListen(vo.ConfigParam{
			DataId: key,
			Group:  o.Group,
			OnChange: func(namespace, group, dataId, data string) {
				// 因为首次加载已经同步阻塞调用过了Get,Listen会异步重新触发一次调用,这里忽略第一次重复触发
				if checkFirst {
					shouldIgnore := firstData == data
					checkFirst = false
					firstData = ""
					if shouldIgnore {
						return
					}
				}

				if data != "" {
					listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: dataId, Value: data})
				} else {
					listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeDelete, Key: dataId, Value: data})
				}
			},
		})
	} else {
		err = c.doListen(vo.ConfigParam{
			DataId: key,
			Group:  o.Group,
			OnChange: func(namespace, group, dataId, data string) {
				if data != "" {
					listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: dataId, Value: data})
				} else {
					listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeDelete, Key: dataId, Value: data})
				}
			},
		})
	}

	if err != nil {
		return fmt.Errorf("nacos listen fail, group: %v, dataId: %v, err: %w", o.Group, key, err)
	}

	return nil
}

// doListen 相同namespace+group+dataId只会触发一次回调
func (c *nacosConfigure) doListen(params vo.ConfigParam) error {
	key := fmt.Sprintf("%s.%s", params.Group, params.DataId)
	_, ok := c.listeners.Load(key)
	if ok {
		return gconfig.ErrDuplicateListen
	}
	if err := c.client.ListenConfig(params); err != nil {
		return err
	}
	c.listeners.Store(key, true)
	return nil
}
