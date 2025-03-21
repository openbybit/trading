package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bgw/pkg/service/ban"
	"bgw/pkg/service/masque"
	"bgw/pkg/service/openapi"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/nacos"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/xid"
)

var (
	banOnce  sync.Once
	masqOnce sync.Once
	userOnce sync.Once
)

func getBanService() (ban.BanServiceIface, error) {
	banOnce.Do(func() {
		conf := getStaticConf()
		appConf := getAppConf()
		_ = ban.Init(ban.Config{
			RpcConf:   conf.BanRpc,
			KafkaConf: conf.Kafka,
			CacheSize: appConf.BanCacheSize,
		})
	})

	return ban.GetBanService()
}

func getMasqueService() (masque.MasqueIface, error) {
	masqOnce.Do(func() {
		conf := getStaticConf()
		_ = masque.Init(masque.Config{
			RpcConf: conf.MasqRpc,
		})
	})

	return masque.GetMasqueService()
}

func getUserService() (openapi.OpenAPIServiceIface, error) {
	userOnce.Do(func() {
		conf := getStaticConf()
		appConf := getAppConf()
		_ = openapi.Init(openapi.Config{
			RpcConf:    conf.UserRpc,
			KafkaConf:  conf.Kafka,
			CacheSize:  appConf.UserCacheSize,
			PrivateKey: appConf.UserPrivateKey,
		})
	})

	return openapi.GetOpenapiService()
}

func isNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}

func dumpPanic(module string, err interface{}) {
	glog.Error(context.Background(), "panic", glog.String("module", module), glog.Any("error", err), glog.String("stack", string(debug.Stack())))
	WSErrorInc(module, "panic")
}

func jsonDecode(r io.Reader, v interface{}) error {
	return jsoniter.NewDecoder(r).Decode(v)
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return jsoniter.Marshal(v)
}

func toJsonString(v interface{}) string {
	x, _ := json.Marshal(v)
	return string(x)
}

func newUUID() string {
	// return uuid.New().String()
	return xid.New().String()
}

func toInt64(v interface{}) int64 {
	return cast.ToInt64(v)
}

func toString(v interface{}) string {
	return cast.ToString(v)
}

func toStringList(args []interface{}) []string {
	res := make([]string, 0, len(args))
	for _, x := range args {
		res = append(res, toString(x))
	}

	return res
}

func nowUnixNano() int64 {
	return time.Now().UnixNano()
}

func decodeUserIDFromToken(tokenStr string) (int64, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if len(tokenStr) == 0 {
		return 0, errors.New("empty token")
	}
	uid, err := strconv.ParseInt(tokenStr, 10, 64)
	if err == nil && uid != 0 {
		return uid, nil
	}

	tokens := strings.SplitN(tokenStr, ".", 3)
	if len(tokens) != 3 {
		return 0, fmt.Errorf("invalid jwt token")
	}
	payload := tokens[1]
	data, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(payload)
	if err != nil {
		return 0, err
	}
	info := make(map[string]interface{})
	if err := json.Unmarshal(data, &info); err != nil {
		return 0, err
	}
	uid = cast.ToInt64(info["user_id"])
	if uid == 0 {
		return 0, fmt.Errorf("invalid userid, %s", data)
	}
	return uid, nil
}

// trimSuffixIndex 去除后缀索引, 例如：ins.linear_1
func trimSuffixIndex(str string) string {
	idx := strings.LastIndexAny(str, "_:.")
	if idx > 0 {
		suf := str[idx+1:]
		_, err := strconv.Atoi(suf)
		if err == nil {
			return str[:idx]
		}
	}

	return str
}

func distinctString(list []string) []string {
	res := make([]string, 0, len(list))
	dic := make(map[string]struct{})
	for _, x := range list {
		if _, ok := dic[x]; !ok {
			dic[x] = struct{}{}
			res = append(res, x)
		}
	}

	return res
}

func verifyPlatform(platform string) string {
	switch platform {
	case "0":
		return "UNKNOWN"
	case "1":
		return "STREAM"
	case "2":
		return "WEB"
	case "3":
		return "APP"
	default:
		if platform == "" {
			return "UNKNOWN"
		}
		return platform
	}
}

// verifyVersion format: xxx.xxx.xxx
func verifyVersion(version string) string {
	if version == "" {
		return "UNKNOWN"
	}

	tokens := strings.Split(version, ".")
	for _, t := range tokens {
		_, err := strconv.Atoi(t)
		if err != nil {
			return "UNKNOWN"
		}
	}
	return version
}

func verifySource(source string) string {
	if source == "" {
		return "UNKNOWN"
	}

	// 存在这种情况 2&timestamp=1670707841325
	index := strings.IndexByte(source, '&')
	if index > 0 {
		return source[:index]
	}
	return source
}

// containsInOrderedList 从有序数组中查找字符串是否存在
func containsInOrderedList(orderedList []string, x string) bool {
	idx := sort.SearchStrings(orderedList, x)
	if idx < len(orderedList) && orderedList[idx] == x {
		return true
	}

	return false
}

// containsAnyInOrderedList 从有序数组中查找另外一个数组,任意一个存在则认为存在,要求第一个参数是有序数组
func containsAnyInOrderedList(orderedList []string, others []string) bool {
	if len(orderedList) == 0 {
		return false
	}

	for _, x := range others {
		if containsInOrderedList(orderedList, x) {
			return true
		}
	}

	return false
}

// isLocalDev 判断是否是本机开发环境
func isLocalDev() bool {
	return env.EnvName() == ""
}

// func isTesting() bool {
// 	if flag.Lookup("test.v") != nil {
// 		return true
// 	}

// 	for _, arg := range os.Args {
// 		if strings.HasPrefix(arg, "-test.") {
// 			return true
// 		}
// 	}
// 	return false
// }

func toListenAddress(port int) string {
	return fmt.Sprintf(":%d", port)
}

// toUnixNano 将秒/毫秒时间戳转换为UnixNano
func toUnixNano(x int64) int64 {
	if x > 1e18 { // unix nano
		return x
	} else if x > 1e15 { // unix micro
		return x * 1e3
	} else if x > 1e12 { // unix micro
		return x * 1e6
	} else { // unix
		return x * 1e9
	}
}

// encodeMapToString 将map序列化成k1=v1,k2=v2的格式,有序,仅用于admin
func encodeMapToString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	b := strings.Builder{}
	for idx, k := range keys {
		if idx != 0 {
			b.WriteByte(',')
		}
		v := m[k]
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}

	return b.String()
}

// setDefaultNacosConfig 通过环境变量设置默认nacos配置
func setDefaultNacosConfig(nc *nacos.NacosConf) {
	const (
		defaultAddress   = "nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848" // 测试环境地址
		defaultNamespace = "public"
		defaultUsername  = "bybit-nacos"
		defaultPassword  = "bybit-nacos"
	)

	if len(nc.ServerConfigs) == 0 {
		addr := os.Getenv("NACOS_REGISTRY_ADDRESS")
		if addr == "" && !env.IsProduction() {
			addr = defaultAddress
		}
		nc.ServerConfigs = append(nc.ServerConfigs, nacos.ServerConfig{Address: addr})
	}

	if nc.NamespaceId == "" {
		if env.IsProduction() {
			nc.NamespaceId = defaultNamespace
		} else {
			nc.NamespaceId = env.ProjectEnvName()
		}
	}

	if nc.AppName == "" {
		nc.AppName = env.ProjectName()
	}

	if nc.Username == "" {
		nc.Username = defaultUsername
	}

	if nc.Password == "" {
		nc.Password = defaultPassword
	}

	if nc.CacheDir == "" {
		nc.CacheDir = "data/nacos/cache"
	}

	if nc.LogDir == "" {
		nc.LogDir = "data/nacos/log"
	}

	if nc.LogLevel == "" {
		nc.LogLevel = "info"
	}
}

func newEnvStore() envStore {
	es := envStore{}
	es.Store()
	return es
}

type envStore struct {
	EnvName        string
	ProjectEnvName string
}

func (s *envStore) SetMainnet() {
	env.SetEnvName("mainnet")
}

func (s *envStore) Store() {
	s.EnvName = env.EnvName()
	s.ProjectEnvName = env.ProjectEnvName()
}

func (s *envStore) Recovery() {
	env.SetEnvName(s.EnvName)
	env.SetProjectEnvName(s.ProjectEnvName)
}
