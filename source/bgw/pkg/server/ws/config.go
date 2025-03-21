package ws

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/nacos"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
	"gopkg.in/yaml.v3"
)

const (
	defaultMaxSessions          = 100000
	defaultMaxSessionsPerIp     = 500
	defaultMaxSessionsPerUser   = 100
	defaultMaxSyncMsgSize       = 0 // 10000
	defaultSessionCmdRateLimit  = 250
	defaultSessionCmdRatePeriod = time.Second
	defaultSessionBufferSize    = 1024 * 15
	defaultAcceptorBufferSize   = 40960
	defaultMaxMetricsSize       = 40960 * 4
	defaultInputDataSize        = 4096
	// 这些topic会定时推送,默认自动屏蔽
	defaultPushTopicBlacklist = `
	private.position,private.wallet,
	umWallet,umPosition,umUsdcPosition,
	user.unify.uta.wallet,
	user.unify.perp.position,
	user.v3.unfiy.merge.option.position,user.v3.unfiy.merge.perp.position,
	user.unify.usdc.borrowInfo,user.unify.usdc.wallet,
	user.portfolio.wallet,
	user.asset.total-balance`
)

const (
	bgwsLogName = "bgws.log"
	pushLogName = "push.log"
)

// Config bgws config
// byone: https://uponly.larksuite.com/wiki/Afv5wSBhoiE0cskhYZsutImUsGe
type Config struct {
	App   AppConf
	WS    WSServerConf
	RPC   RPCServerConf
	Nacos nacos.NacosConf
	Log   LogConf
	Alert AlertConf

	Kafka   kafka.UniversalClientConfig
	MasqRpc zrpc.RpcClientConf
	UserRpc zrpc.RpcClientConf
	BanRpc  zrpc.RpcClientConf
}

// AppConf app config
// nolint
type AppConf struct {
	DevPort                  int           `json:"dev_port,default=6480"`                      // 调试端口
	Mode                     string        `json:"mode,default=debug,opitons=[debug,release]"` // 是否是调试模式
	Cluster                  string        `json:"cluster,optional"`                           // 集群信息,主网必须配置
	EnableMockLogin          bool          `json:"enable_mock_login,optional"`                 // 是否开启登录mock
	DisableSubscribeCheck    bool          `json:"disable_subscribe_check,optional"`           // 是否禁用订阅验证
	EnableTopicConflictCheck bool          `json:"enable_topic_conflict_check,default=false"`  // 是否开启topic注册冲突检测,仅uws集群需要
	EnableAsyncMetrics       bool          `json:"enable_async_metrics,default=true"`          // 是否开启异步metrics
	DisableDeadlockCheck     bool          `json:"disable_deadlock_check,default=true"`        // 是否禁止死锁检测
	AsyncUserEnable          bool          `json:"async_user_enable,default=true"`             // 是否允许异步同步用户
	AsyncUserChannelSize     int           `json:"async_user_channel_size,default=50000"`      // chan大小,散户集群总共有2ktps订阅量,单机几百tps
	AsyncUserBatchSize       int           `json:"async_user_batch_size,default=100"`          // 同步用户批次合并大小
	ReplaceStreamPlatform    bool          `json:"replace_stream_platform,optional"`           // 是否需要强制设置_platform,期权通过此参数判断是否是stream集群
	EnableController         bool          `json:"enable_controller,default=false"`            // 是否支持http trade
	UserSvcRateLimit         int64         `json:"user_svc_rate_limit,default=0"`              // 用户服务限频
	DisableUserTick          bool          `json:"disable_user_tick,default=false"`            // 禁止user定时器,
	DisableSessionTick       bool          `json:"disable_session_tick,default=false"`         // 禁止session定时器
	MaxSessions              int           `json:"max_sessions,default=100000"`                // 最大连接数
	MaxSessionsPerIp         int           `json:"max_sessions_per_ip,default=500"`            // 单个ip最大连接数
	MaxSessionsPerUser       int           `json:"max_sessions_per_user,default=100"`          // 单个用户最大连接数
	UserPrivateKey           string        `json:"user_private_key,optional"`                  // user服务私钥
	UserCacheSize            int           `json:"user_cache_size,optional"`                   // user服务cache大小
	BanCacheSize             int           `json:"ban_cache_size,optional"`                    // ban 服务cache大小
	PrivateTopicList         []string      `json:"private_topic_list,optional"`                // 私有topic列表
	AuthTickEnable           bool          `json:"auth_tick_enable,default=true"`              // 是否禁用定时auth校验
	MaxSnapshotSize          int           `json:"max_snapshot_size,default=20"`               // 队列中最大snapshot数量,过大会积压,过小会频繁全量同步
	StopWaitTime             time.Duration `json:"stop_wait_time,default=20s"`                 // 服务停止等待时间,用于优雅下线
}

func (a *AppConf) SetDefaultTesting() {
	a.EnableMockLogin = true
	a.DisableSubscribeCheck = true
	a.EnableTopicConflictCheck = false
	a.DisableDeadlockCheck = true
}

// LogConf
// nolint
type LogConf struct {
	Type       string `json:"type,optional"`  // 默认本地console,其他file
	Level      string `json:"level,optional"` // 默认主网info,测试环境debug
	Path       string `json:"path,optional"`  // 根目录,不需要文件名
	MaxSize    int    `json:"max_size,default=300"`
	MaxAge     int    `json:"max_age,default=30"`
	MaxBackups int    `json:"max_backups,default=200"`
}

func (c *LogConf) convert(fileName string) glog.Config {
	if c.Level == "" {
		if env.IsMainnet() {
			c.Level = "info"
		} else {
			c.Level = "debug"
		}
	}

	if c.Type == "" {
		if isLocalDev() {
			c.Type = glog.TypeConsole
		} else {
			c.Type = glog.TypeLumberjack
		}
	}

	if c.Path == "" {
		c.Path = "/data/logs/bgw/bgws"
	}

	level, _ := glog.ParseLevel(c.Level)
	file := filepath.Join(c.Path, fileName)

	return glog.Config{
		File:       file,
		Type:       c.Type,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Level:      level,
	}
}

type AlertConf struct {
	Path string `json:"path,optional"`
}

// RPCServerConf
// nolint
type RPCServerConf struct {
	ListenType     string `json:"listen_type,default=all,options=[tcp,all]"` // 监听类型
	ListenTcpPort  int    `json:"listen_tcp_port,default=8060"`              // tcp监听端口
	ListenUnixAddr string `json:"listen_unix_addr,optional"`                 // unix监听地址
}

// WSServerConf
// nolint
type WSServerConf struct {
	ListenPort         int           `json:"listen_port,default=8081"`          // 监听端口
	Compression        bool          `json:"compression,default=true"`          //
	ReadTimeout        time.Duration `json:"read_timeout,default=60s"`          //
	WriteTimeout       time.Duration `json:"write_timeout,default=6s"`          //
	IdleTimeout        time.Duration `json:"idle_timeout,default=10s"`          //
	ReadBufferSize     int           `json:"read_buffer_size,default=81920"`    //
	WriteBufferSize    int           `json:"write_buffer_size,default=1024000"` //
	MaxRequestBodySize int           `json:"max_request_body_size,default=4"`   // 单位M
	Routes             []string      `json:"routes"`                            //
	EnableRegistry     bool          `json:"enable_registry,default=false"`     // 是否开启服务注册
	ServiceName        string        `json:"service_name,optional"`             // 服务名后缀
}

func newDynamicConf() *dynamicConf {
	appConf := getAppConf()
	d := &dynamicConf{
		MaxSessions:          appConf.MaxSessions,
		MaxSessionsPerIp:     appConf.MaxSessionsPerIp,
		MaxSessionsPerUser:   appConf.MaxSessionsPerUser,
		MaxSyncMsgSize:       defaultMaxSyncMsgSize,
		SessionBufferSize:    defaultSessionBufferSize,
		AcceptorBufferSize:   defaultAcceptorBufferSize,
		MaxMetricsSize:       defaultMaxMetricsSize,
		InputDataSize:        defaultInputDataSize,
		SessionCmdRateLimit:  defaultSessionCmdRateLimit,
		SessionCmdRatePeriod: defaultSessionCmdRatePeriod,
		UidBlackList:         make(Int64Set),
		IpBlackList:          make(StringSet),
		IpWhiteList:          make(StringSet),
		GrayList:             make(Int64Set),
		EnableAcceptorRetry:  false,
		EnableGracefulClose:  false,
		EnableInputLog:       true,
		DisableBan:           false,
	}
	d.Verify()
	return d
}

// dynamicConf 服务相关动态配置
type dynamicConf struct {
	Log                  dynamicLogConf `yaml:"log" json:"log"`                                         // 日志配置
	MaxSessions          int            `yaml:"max_sessions" json:"max_sessions"`                       // 最大连接数
	MaxSessionsPerIp     int            `yaml:"max_sessions_per_ip" json:"max_sessions_per_ip"`         // 单个ip最大连接数
	MaxSessionsPerUser   int            `yaml:"max_sessions_per_user" json:"max_sessions_per_user"`     // 单个用户最大连接数
	MaxSyncMsgSize       int            `yaml:"max_sync_msg_size" json:"max_sync_msg_size"`             // 分割大小,如何是零的话则不分割
	SessionBufferSize    int            `yaml:"session_buffer_size" json:"session_buffer_size"`         // session缓冲区大小
	AcceptorBufferSize   int            `yaml:"acceptor_buffer_size" json:"acceptor_buffer_size"`       // acceptor缓冲区大小
	InputDataSize        int            `yaml:"input_data_size" json:"input_data_size"`                 // im数据包大小
	SessionCmdRateLimit  int            `yaml:"session_cmd_rate_limit" json:"session_cmd_rate_limit"`   // 用户限频
	SessionCmdRatePeriod time.Duration  `yaml:"session_cmd_rate_period" json:"session_cmd_rate_period"` // 用户限频周期
	GrayList             Int64Set       `yaml:"gray_list" json:"gray_list"`                             // 灰度账户列表
	UidBlackList         Int64Set       `yaml:"uid_black_list" json:"uid_black_list"`                   // 用户黑名单
	IpBlackList          StringSet      `yaml:"ip_black_list" json:"ip_black_list"`                     // ip很名单
	IpWhiteList          StringSet      `yaml:"ip_white_list" json:"ip_white_list"`                     // ip白名单

	EnableGracefulClose          bool          `yaml:"enable_graceful_close" json:"enable_graceful_close"`                     // 是否允许优雅关闭
	EnableAcceptorRetry          bool          `yaml:"enable_acceptor_retry" json:"enable_acceptor_retry"`                     // 是否允许发送重试
	EnableInputLog               bool          `yaml:"enable_input_log" json:"enable_input_log"`                               // 是否允许打印im聊天简要日志
	DisableBan                   bool          `yaml:"disable_ban" json:"disable_ban"`                                         // 禁用ban服务,仅用于临时手动降级ban服务,默认false
	SyncUserRegularEnable        bool          `yaml:"sync_user_regular_enable" json:"sync_user_regular_enable"`               // 是否开启定时常规同步用户
	SyncUserRegularInterval      time.Duration `yaml:"sync_user_regular_interval" json:"sync_user_regular_interval"`           // 异步同步用户时间间隔
	DisableAuthorizedOnConnected bool          `yaml:"disable_authorized_on_connected" json:"disable_authorized_on_connected"` // 连接时鉴权,默认开启
	MaxMetricsSize               int           `yaml:"max_metrics_size" json:"max_metrics_size"`                               // 最大数量
}

func (d *dynamicConf) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, d); err != nil {
		glog.Errorf(context.Background(), "parseSdkConfig unmarshal fail, err: %v", err)
		return err
	}

	d.Log.build()
	d.Verify()

	return nil
}

func (d *dynamicConf) Verify() {
	verifyIntFn := func(v *int, def int) {
		if *v <= 0 {
			*v = def
		}
	}

	verifyIntFn(&d.MaxSessions, defaultMaxSessions)
	verifyIntFn(&d.MaxSessionsPerIp, defaultMaxSessionsPerIp)
	verifyIntFn(&d.MaxSessionsPerUser, defaultMaxSessionsPerUser)
	verifyIntFn(&d.MaxSyncMsgSize, defaultMaxSyncMsgSize)
	verifyIntFn(&d.SessionBufferSize, defaultSessionBufferSize)
	verifyIntFn(&d.AcceptorBufferSize, defaultAcceptorBufferSize)
	verifyIntFn(&d.InputDataSize, defaultInputDataSize)
	verifyIntFn(&d.MaxMetricsSize, defaultMaxMetricsSize)
}

func (d *dynamicConf) IsGray(uid int64) bool {
	_, ok := d.GrayList[uid]
	return ok
}

type dynamicLogConf struct {
	Level              string        `yaml:"level" json:"level"`                               // glog日志等级
	Enable             bool          `yaml:"enable" json:"enable"`                             // 是否启用日志功能
	EnableAllUser      bool          `yaml:"enable_all_user" json:"enable_all_user"`           // 是否所用用户都打印日志
	EnableWriteData    bool          `yaml:"enable_write_data" json:"enable_write_data"`       // 是否打印message中的数据, 默认false
	EnableSessionLimit bool          `yaml:"enable_session_limit" json:"enable_session_limit"` // 是否允许打印session rate limit日志
	Duration           time.Duration `yaml:"duration" json:"duration"`                         // 记录时长，避免忘记关闭
	Uids               Int64Set      `yaml:"uids" json:"uids"`                                 // 需要跟踪的uid
	SyncAppWhitelist   StringSet     `yaml:"sync_app_whitelist" json:"sync_app_whitelist"`     // 同步日志输出白名单
	PushAppWhitelist   StringSet     `yaml:"push_app_whitelist" json:"push_app_whitelist"`     // 推送日志输出白名单,根据AppId
	PushTopicWhitelist StringSet     `yaml:"push_topic_whitelist" json:"push_topic_whitelist"` // 推送日志输出白名单,根据Topic
	PushTopicBlacklist StringSet     `yaml:"push_topic_blacklist" json:"push_topic_blacklist"` // 推送日志输出黑名单,根据Topic,debug场景下,position/wallet这种定时推送都不需要推送
	endTime            int64
}

// CanWritePushLog 能否打印推送日志
func (l *dynamicLogConf) CanWritePushLog(uid int64, appId string, topic string) bool {
	if enableDebug {
		return !l.PushTopicBlacklist.Has(topic)
	}

	if !l.Enable {
		return false
	}

	// 白名单
	if l.EnableAllUser || l.PushAppWhitelist.Has(appId) || l.PushTopicWhitelist.Has(topic) {
		return true
	}

	if _, ok := l.Uids[uid]; !ok {
		return false
	}

	return time.Now().UnixNano() <= l.endTime
}

// CanWriteSyncLog 是否打印同步日志
func (l *dynamicLogConf) CanWriteSyncLog(appId string) bool {
	if enableDebug {
		return true
	}

	if _, ok := l.SyncAppWhitelist[appId]; ok {
		return true
	}

	return false
}

func (l *dynamicLogConf) build() {
	if enableDebug && len(l.PushTopicBlacklist) == 0 {
		if l.PushTopicBlacklist == nil {
			l.PushTopicBlacklist = make(StringSet)
		}

		list := strings.Split(defaultPushTopicBlacklist, ",")
		for _, v := range list {
			l.PushTopicBlacklist.Add(strings.TrimSpace(v))
		}
	}

	if len(l.Uids) == 0 {
		return
	}

	if l.Duration <= 0 {
		l.Duration = time.Minute * 30
	}

	l.endTime = time.Now().Add(l.Duration).UnixNano()
}

func newSdkConf() *sdkConf {
	return &sdkConf{}
}

type options map[string]string

// sdkConf 同步给sdk的配置
// 当前支持的配置
// gray_mode: new old hybrid
type sdkConf struct {
	Disable    bool               `yaml:"disable" json:"disable"`                   // 是否禁用动态配置
	Options    map[string]string  `yaml:"options" json:"options,omitempty"`         // 全局的options,会同步给所有sdk
	AppOptions map[string]options `yaml:"app_options" json:"app_options,omitempty"` // app基本的config
}

func (c *sdkConf) GetOptionsByAppID(appId string) options {
	if appId != "" {
		res, ok := c.AppOptions[appId]
		if ok {
			return res
		}
	}

	return c.Options
}

func (c *sdkConf) Parse(data string) error {
	if err := yaml.Unmarshal([]byte(data), c); err != nil {
		glog.Errorf(context.Background(), "parseSdkConfig unmarshal fail, err: %v", err)
		return err
	}

	// 合并config,将全局的options写到每个独立的app中,但不能覆盖已经存在的配置
	if len(c.Options) > 0 && len(c.AppOptions) > 0 {
		for k, v := range c.Options {
			for _, appOpts := range c.AppOptions {
				if _, ok := appOpts[k]; !ok {
					appOpts[k] = v
				}
			}
		}
	}

	return nil
}
