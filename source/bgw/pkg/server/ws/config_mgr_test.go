package ws

import (
	"strings"
	"testing"
	"time"

	"bgw/pkg/common/constant"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
)

func TestParseSdkConfig(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	data := `
topics:
  private.position: 1
  private.wallet: 2
options:
  test: aaa
  same: aaa
app_options:
  trading-express-dev:
    same: bbb
    diff: ccc
`

	t.Run("parse config", func(t *testing.T) {
		gConfigMgr.onLoadSdkConfig(&gconfig.Event{Value: strings.TrimSpace(data)})

		cfg := gConfigMgr.GetSdkConf()
		assert.EqualValues(t, map[string]string{"test": "aaa", "same": "aaa"}, cfg.Options)
		assert.EqualValues(t, map[string]string{"diff": "ccc", "same": "bbb", "test": "aaa"}, cfg.GetOptionsByAppID("trading-express-dev"))
	})

	t.Run("onload", func(t *testing.T) {
		gConfigMgr.onLoadSdkConfig(nil)
		gConfigMgr.onLoadSdkConfig(&gconfig.Event{})
		gConfigMgr.onLoadSdkConfig(&gconfig.Event{Value: "invalid data"})
	})
}

func TestParseDynamicConfig(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	data := `
log:
    enable: true
    enable_session_limit: true
    enable_write_data: false
    duration: 1s
    uids: [71834710,63228888]
max_sessions: 1
max_sessions_per_ip: 2
max_sessions_per_user: 3
max_sync_msg_size: 4
session_buffer_size: 5
acceptor_buffer_size: 6
input_data_size: 7
session_cmd_rate_limit: 8
session_cmd_rate_period: 9s
async_sync_user_interval: 10s
gray_list: [1301180,1198860]
uid_black_list: [11]
ip_black_list: [12]
ip_white_list: [13]
enable_graceful_close: false
enable_acceptor_retry: false
enable_input_log: false
disable_ban: false
	`

	t.Run("parse_dynamic_config", func(t *testing.T) {
		gConfigMgr.onLoadDynamicConfig(&gconfig.Event{Value: strings.TrimSpace(data)})

		cfg := gConfigMgr.GetDynamicConf()
		lconf := cfg.Log
		lconf.PushTopicBlacklist = nil
		lconf.endTime = 0
		assert.Equal(t, &dynamicLogConf{Enable: true, EnableSessionLimit: true, EnableWriteData: false, Duration: time.Second, Uids: Int64Set{71834710: {}, 63228888: {}}}, &lconf)
		assert.Equal(t, 1, cfg.MaxSessions)
		assert.Equal(t, 2, cfg.MaxSessionsPerIp)
		assert.Equal(t, 3, cfg.MaxSessionsPerUser)
		assert.Equal(t, 4, cfg.MaxSyncMsgSize)
		assert.Equal(t, 5, cfg.SessionBufferSize)
		assert.Equal(t, 6, cfg.AcceptorBufferSize)
		assert.Equal(t, 7, cfg.InputDataSize)
		assert.Equal(t, 8, cfg.SessionCmdRateLimit)
		assert.Equal(t, time.Second*9, cfg.SessionCmdRatePeriod)
		assert.Equal(t, Int64Set{1301180: {}, 1198860: {}}, cfg.GrayList)
		assert.Equal(t, Int64Set{11: {}}, cfg.UidBlackList)
		assert.Equal(t, StringSet{"12": {}}, cfg.IpBlackList)
		assert.Equal(t, StringSet{"13": {}}, cfg.IpWhiteList)
	})

	t.Run("onload", func(t *testing.T) {
		gConfigMgr.onLoadDynamicConfig(nil)
		gConfigMgr.onLoadDynamicConfig(&gconfig.Event{})
		gConfigMgr.onLoadDynamicConfig(&gconfig.Event{Value: "invalid data"})
	})
}

func TestLogEnable(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	uid := int64(123)
	appId := "app1"
	topic := "t1"
	l := dynamicLogConf{
		Enable:             true,
		Duration:           time.Second,
		Uids:               map[int64]struct{}{uid: {}},
		PushTopicWhitelist: map[string]struct{}{},
		SyncAppWhitelist:   map[string]struct{}{appId: {}},
	}
	l.build()

	t.Run("push log", func(t *testing.T) {
		enableDebug = false
		assert.Falsef(t, l.CanWritePushLog(1, appId, topic), "uid not match")
		assert.True(t, l.CanWritePushLog(uid, appId, topic), "uid match")

		l.Enable = false
		assert.Falsef(t, l.CanWritePushLog(uid, appId, topic), "disable")
		l.Enable = true
		l.PushAppWhitelist = map[string]struct{}{appId: {}}
		assert.Truef(t, l.CanWritePushLog(uid, appId, topic), "app whitelist")

		enableDebug = true
		assert.True(t, l.CanWritePushLog(1, "", ""))
	})

	t.Run("sync log", func(t *testing.T) {
		enableDebug = false
		assert.True(t, l.CanWriteSyncLog(appId))
		assert.False(t, l.CanWriteSyncLog("not_exits_id"))
		enableDebug = true
		assert.True(t, l.CanWriteSyncLog(""))
	})
}

func TestLogBuild(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	l := dynamicLogConf{
		Enable:             true,
		Duration:           0,
		Uids:               map[int64]struct{}{1: {}},
		PushTopicWhitelist: map[string]struct{}{},
	}
	l.build()
	assert.Equal(t, time.Minute*30, l.Duration)
}

func TestPushTopicBlacklist(t *testing.T) {
	list := strings.Split(defaultPushTopicBlacklist, ",")
	for _, v := range list {
		v = strings.TrimSpace(v)
		assert.NotEmpty(t, v)
	}
}

func TestLoadStaticConfig(t *testing.T) {
	getConfigMgr().LoadStaticConfig()
}

func TestLoadDynamicConfig(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	getConfigMgr().LoadDynamicConfig()
}

func TestGetNamespace(t *testing.T) {
	es := newEnvStore()
	env.SetProjectEnvName("unify-test-1")
	assert.Equal(t, "unify-test-1", gConfigMgr.getNamespace())
	env.SetProjectEnvName("")
	assert.Equal(t, constant.BGWConfigNamespace, gConfigMgr.getNamespace())
	es.SetMainnet()
	assert.Equal(t, constant.BGWConfigNamespace, gConfigMgr.getNamespace())
	es.Recovery()
}

func TestServerConfigNacosKey(t *testing.T) {
	assert.Equal(t, "bgws_config", gConfigMgr.getServerConfigNacosKey(""))
	assert.Equal(t, "bgws_config_site", gConfigMgr.getServerConfigNacosKey("site"))
}

func TestCheckTopics(t *testing.T) {
	gConfigMgr.AddTopic(topicTypePrivate, []string{"private_topic"})
	gConfigMgr.AddTopic(topicTypePublic, []string{"public_topic"})
	s, f := gConfigMgr.CheckTopics([]string{"private_topic", "public_topic", "invalid"})
	assert.ElementsMatch(t, s, []string{"private_topic", "public_topic"})
	assert.ElementsMatch(t, f, []string{"invalid"})
	assert.True(t, gConfigMgr.HasPrivateTopics([]string{"private_topic", "public_topic"}))
	assert.ElementsMatch(t, []string{"private_topic"}, gConfigMgr.IgnorePublicTopics([]string{"private_topic", "public_topic"}))
}
