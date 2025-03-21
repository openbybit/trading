package gnacos

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.bydev.io/frameworks/nacos-sdk-go/v2/common/constant"
)

const (
	NAMESPACE_KEY               = "namespace"
	GROUP_KEY                   = "group"
	SERVICE_KEY                 = "service"
	REGISTRY_TIMEOUT_KEY        = "registry.timeout"
	TIMEOUT_KEY                 = "timeout"
	DEFAULT_ROLETYPE            = 3
	CACHE_DIR_KEY               = "cache"
	LOG_DIR_KEY                 = "log"
	LOG_LEVEL_KEY               = "loglevel"
	BEAT_INTERVAL_KEY           = "beatInterval"
	ENDPOINT_KEY                = "endpoint"
	CATEGORY_KEY                = "category"
	PROTOCOL_KEY                = "gprotocol"
	PATH_KEY                    = "path"
	NOT_LOAD_LOCAL_CACHE_KEY    = "nacos.not.load.cache"
	UPDATE_CACHE_WHEN_EMPTY_KEY = "nacos.update.cache.when.empty"
	APP_NAME_KEY                = "appName"
	REGION_ID_KEY               = "regionId"
	ACCESS_KEY                  = "access"
	SECRET_KEY                  = "secret"
	OPEN_KMS_KEY                = "kms"
	UPDATE_THREAD_NUM_KEY       = "updateThreadNum"
	SHARE_KEY                   = "share"
	UNIQUE_NAME_KEY             = "uniqueName"
)

const (
	DEFAULT_NAMESPACE     = "public"
	DEFAULT_GROUP         = "DEFAULT_GROUP"
	DEFAULT_USERNAME      = "bybit-nacos"
	DEFAULT_PASSWORD      = "bybit-nacos"
	DEFAULT_TIMEOUT       = "10s"
	DEFUALT_BEAT_INTERVAL = 5000
)

var (
	errNacosUrlEmpty     = fmt.Errorf("nacos url empty")
	errNacosUrlHostEmpty = errors.New("nacos url host empty")
	errInvalidNacosUrl   = errors.New("invalid nacos url")
	errNacosRawUrlEmpty  = errors.New("nacos raw url empty")
)

type serverConfig = constant.ServerConfig
type clientConfig = constant.ClientConfig

// Config nacos client & server config
type Config struct {
	Name   string
	Share  bool
	Server []constant.ServerConfig
	Client constant.ClientConfig
}

// NewConfig get nacos config
func NewConfig(address string, username string, password string, namespace string, params map[string]string) (*Config, error) {
	u := url.URL{Host: address}

	if username != "" || password != "" {
		u.User = url.UserPassword(username, password)
	}

	// https://www.alexedwards.net/blog/change-url-query-params-in-go
	if len(params) > 0 || namespace != "" {
		values := u.Query()

		if namespace != "" {
			values.Set(NAMESPACE_KEY, namespace)
		}

		if len(params) > 0 {
			for k, v := range params {
				values.Set(k, v)
			}
		}

		u.RawQuery = values.Encode()
	}

	return NewConfigByURL(&u)
}

// NewConfigByURL get config from url
// nacos://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?cache=data/cache/nacos&log=data/logs/nacos&loglevel=error&namespace=bgw-aaron&timeout=30s
// nacos://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?cache=data%2Fcache%2Fnacos&log=data%2Flogs%2Fnacos&loglevel=error&namespace=bgw-aaron&timeout=30s
func NewConfigByURL(xurl interface{}) (conf *Config, err error) {
	u, err := toURL(xurl)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, errNacosUrlEmpty
	}

	if u.Host == "" {
		return nil, errNacosUrlHostEmpty
	}

	addresses := strings.Split(u.Host, ",")
	serverConfigs := make([]constant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, constant.ServerConfig{IpAddr: ip, Port: uint64(port)})
	}

	q := u.Query()

	timeoutStr := q.Get(REGISTRY_TIMEOUT_KEY)
	if timeoutStr == "" {
		timeoutStr = q.Get(TIMEOUT_KEY)
	}
	if timeoutStr == "" {
		timeoutStr = DEFAULT_TIMEOUT
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, err
	}

	namespace := getParam(q, NAMESPACE_KEY, "")
	username := u.User.Username()
	password, _ := u.User.Password()

	clientConfig := constant.ClientConfig{
		TimeoutMs:            uint64(int32(timeout / time.Millisecond)),
		BeatInterval:         getParamInt64(q, BEAT_INTERVAL_KEY, 5000),
		NamespaceId:          namespace,
		AppName:              getParam(q, APP_NAME_KEY, ""),
		Endpoint:             getParam(q, ENDPOINT_KEY, ""),
		RegionId:             getParam(q, REGION_ID_KEY, ""),
		AccessKey:            getParam(q, ACCESS_KEY, ""),
		SecretKey:            getParam(q, SECRET_KEY, ""),
		OpenKMS:              getParamBool(q, OPEN_KMS_KEY, false),
		CacheDir:             getParam(q, CACHE_DIR_KEY, ""),
		UpdateThreadNum:      getParamInt(q, UPDATE_THREAD_NUM_KEY, 20),
		NotLoadCacheAtStart:  getParamBool(q, NOT_LOAD_LOCAL_CACHE_KEY, true), // load cache at start time
		UpdateCacheWhenEmpty: getParamBool(q, UPDATE_CACHE_WHEN_EMPTY_KEY, true),
		Username:             username,
		Password:             password,
		LogDir:               getParam(q, LOG_DIR_KEY, ""),
		LogLevel:             getParam(q, LOG_LEVEL_KEY, "info"),
	}
	if clientConfig.NamespaceId == DEFAULT_NAMESPACE { // fix: overwrite nacos namespace on public
		clientConfig.NamespaceId = ""
	}

	share := getParamBool(q, SHARE_KEY, true)
	name := getParam(q, UNIQUE_NAME_KEY, "")
	if name == "" {
		name = buildName(u.Host, namespace)
	}

	return &Config{Name: name, Share: share, Server: serverConfigs, Client: clientConfig}, nil
}

func toURL(v interface{}) (*url.URL, error) {
	if v == nil {
		return nil, errInvalidNacosUrl
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return nil, errNacosRawUrlEmpty
		}
		return url.Parse(x)
	case *url.URL:
		return x, nil
	case url.URL:
		return &x, nil
	default:
		return nil, fmt.Errorf("invalid nacos url type, %v", x)
	}
}

// buildName 通过host和namespace保证nacos全局唯一
func buildName(host string, namespace string) string {
	return fmt.Sprintf("%v.%v", host, namespace)
}

func getParam(q url.Values, key string, def string) string {
	if !q.Has(key) {
		return def
	}

	return q.Get(key)
}

func getParamInt(q url.Values, key string, def int) int {
	if !q.Has(key) {
		return def
	}
	v := q.Get(key)
	res, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return res
}

func getParamInt64(q url.Values, key string, def int64) int64 {
	if !q.Has(key) {
		return def
	}
	v := q.Get(key)
	res, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return res
}

func getParamBool(q url.Values, key string, def bool) bool {
	if !q.Has(key) {
		return def
	}
	v := q.Get(key)
	res, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return res
}
