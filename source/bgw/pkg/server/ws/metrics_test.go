package ws

import (
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"github.com/stretchr/testify/assert"
)

func init() {
	gmetric.Init("test")
	initMetrics("testing")
}

func TestWritePushLog(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	m := newMetricsManager()
	m.initLogger()
	// 不真正输出
	m.log.SetLevel(glog.FatalLevel)
	msg := &metricsMessage{
		AppID:         "linear",
		RemoteAddr:    "127.0.0.1",
		WsStartTimeE9: 5,
		WsEndTimeE9:   6,
		Sessions:      []string{"111"},
		Push: &envelopev1.PushMessage{
			Topic:         "topic1",
			UserId:        1,
			RequestTimeE9: 2,
			InitTimeE9:    3,
			SdkTimeE9:     4,
			TraceId:       "traceid",
			MessageId:     "msgid",
		},
	}

	enableDebug = false
	m.writeLog(msg)
	enableDebug = true
	m.writeLog(msg)
	// write empty
	m.writeLog(&metricsMessage{Push: &envelopev1.PushMessage{}})
	m.writeLog(&metricsMessage{Push: &envelopev1.PushMessage{PushType: envelopev1.PushType_PUSH_TYPE_PUBLIC}})
	m.Stop()
}

func TestMetrics(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		WSHistogram(time.Time{}, "", "")
		WSHistogram(time.Now(), "", "")
		wsSyncAdd(1, "batch_sync_by_users", "appid")
		wsUpgradeInc("/v5/private", "web", "", "0.0.1")
		WSErrorAdd(1, "a", "b")
		WSErrorInc("a", "b")
	})

	t.Run("send", func(t *testing.T) {
		gMetricsMgr.Send(nil)
		appConf := getAppConf()
		appConf.EnableAsyncMetrics = false
		gMetricsMgr.Send(&metricsMessage{Type: metricsTypePush, Push: &envelopev1.PushMessage{TraceId: "aaaa"}})
		appConf.EnableAsyncMetrics = true
		getDynamicConf().MaxMetricsSize = 0
		gMetricsMgr.Send(&metricsMessage{Type: metricsTypePush, Push: &envelopev1.PushMessage{TraceId: "aaaa"}})
		gMetricsMgr.Send(&metricsMessage{Type: metricsTypePush, Push: &envelopev1.PushMessage{TraceId: "bbbb"}})
		getDynamicConf().MaxMetricsSize = defaultMaxMetricsSize
	})

	t.Run("message", func(t *testing.T) {
		m := newMetricsMessage()
		assert.EqualValues(t, 1, m.Refs)
		m.Retain()
		assert.EqualValues(t, 2, m.Refs)
		m.Release()
		assert.EqualValues(t, 1, m.Refs)
	})

	t.Run("sendMessage", func(t *testing.T) {
		now := time.Now().UnixNano()
		msg := newMetricsMessage()
		msg.Type = metricsTypePush
		msg.Push = &envelopev1.PushMessage{
			InitTimeE9: now,
			TraceId:    "aaaa",
		}
		msg.EndTimeE9 = append(msg.EndTimeE9, now)

		m := newMetricsManager()
		m.sendMessage(msg)
	})
}
