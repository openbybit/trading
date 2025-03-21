package ws

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustinxie/lockfree"
	"github.com/opentracing/opentracing-go"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
)

const namespaceWS = "bgw_ws"

var (
	wsErrorCounter gmetric.CounterVec
	wsCounter      gmetric.CounterVec
	wsGauge        gmetric.GaugeVec

	wsDefaultLatency gmetric.HistogramVec
	wsPushLatency    gmetric.HistogramVec

	wsMessageHistogram gmetric.HistogramVec

	// ws client upgrade
	wsUpgradeCounter gmetric.CounterVec
	// 同步用户数据埋点
	wsSyncCounter gmetric.CounterVec
)

func initMetrics(cluster string) {
	var labels gmetric.Labels
	if cluster != "" {
		labels = make(gmetric.Labels)
		labels["cluster"] = cluster
	}

	if wsErrorCounter != nil {
		glog.Info(context.TODO(), "skip initMetrics", glog.String("cluster", cluster))
		return
	}

	wsErrorCounter = gmetric.NewCounterVec(&gmetric.CounterVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "",
		Name:        "error",
		Help:        "common error counter.",
		ConstLabels: labels,
		Labels: []string{
			"type",  // error type
			"error", // error message
		},
	})

	wsCounter = gmetric.NewCounterVec(&gmetric.CounterVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "counter",
		Name:        "counter",
		Help:        "default counter.",
		ConstLabels: labels,
		Labels: []string{
			"type",  // counter type
			"label", // counter label
		},
	})

	wsGauge = gmetric.NewGaugeVec(&gmetric.GaugeVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "gauge",
		Name:        "gauge",
		Help:        "default gauge.",
		ConstLabels: labels,
		Labels: []string{
			"type",
			"label",
		},
	})

	wsDefaultLatency = gmetric.NewHistogramVec(&gmetric.HistogramVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "latency",
		Name:        "default",
		Help:        "default latency histogram(ms).",
		ConstLabels: labels,
		Labels: []string{
			"type",  // section type
			"label", // label
		},
		Buckets: []float64{
			0.1, 0.2, 0.4, 0.8, // 20-800us
			1, 2, 4, 8, // 1-8ms
			10, 12, 14, 18, // 10-18ms
			20, 40, 60, 80, // 20-80ms
			100, 200, 400, 800, 1000, // 100-1000ms
		},
	})

	wsPushLatency = gmetric.NewHistogramVec(&gmetric.HistogramVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "latency",
		Name:        "push",
		Help:        "push latency histogram(ms).",
		ConstLabels: labels,
		Labels: []string{
			"type",   // section type
			"topic",  // topic
			"app_id", // appid
		},
		Buckets: []float64{
			0.1, 0.2, 0.4, 0.8, // 20-800us
			1, 2, 4, 8, // 1-8ms
			10, 12, 14, 18, // 10-18ms
			20, 40, 60, 80, // 20-80ms
			100, 200, 400, 800, 1000, // 100-1000ms
		},
	})

	// 用于统计某个节点的推送量
	// wsPushCounter = gmetric.NewCounterVec(&gmetric.CounterVecOpts{
	// 	Namespace: namespaceWS,
	// 	Subsystem: "",
	// 	Name:      "push",
	// 	Help:      "push data to user",
	// 	Labels:    []string{"app_id", "address"},
	// })

	wsMessageHistogram = gmetric.NewHistogramVec(&gmetric.HistogramVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "message",
		Name:        "histogram",
		Help:        "consumer message raw size histogram.",
		ConstLabels: labels,
		Labels: []string{
			"topic",  // topic name
			"app_id", // business tag
		},
		Buckets: []float64{
			512, 1024, 2048, 4096, 8192, // 1~8k
			10240, 20480, 40960, 81920, // 10~80k
			102400, 204800, 409600, // 100-400k
		},
	})

	// ws client upgrade
	wsUpgradeCounter = gmetric.NewCounterVec(&gmetric.CounterVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "ws",
		Name:        "upgrade",
		Help:        "http upgrade to ws client.",
		ConstLabels: labels,
		Labels: []string{
			"path",
			"platform",
			"source",
			"version",
		},
	})

	// 同步用户数据埋点
	wsSyncCounter = gmetric.NewCounterVec(&gmetric.CounterVecOpts{
		Namespace:   namespaceWS,
		Subsystem:   "",
		Name:        "sync",
		Help:        "sync user to sdk",
		ConstLabels: labels,
		Labels:      []string{"type", "app_id"},
	})
}

// WSCounterInc inc 1
func WSCounterInc(typ, label string) {
	wsCounter.Inc(typ, label)
}

func WSErrorInc(typ, err string) {
	wsErrorCounter.Inc(typ, err)
}

func WSErrorAdd(value int, typ, err string) {
	wsErrorCounter.Add(float64(value), typ, err)
}

// WSGauge gauge with type and label
func WSGauge(count float64, typ, label string) {
	wsGauge.Set(count, typ, label)
}

// WSGaugeInc gauge with type and label
func WSGaugeInc(typ, label string) {
	wsGauge.Inc(typ, label)
}

// WSGaugeDec gauge with type and label
func WSGaugeDec(typ, label string) {
	wsGauge.Dec(typ, label)
}

// WSHistogram record latency with type and label
func WSHistogram(t time.Time, typ, label string) {
	if t.IsZero() {
		return
	}

	wsDefaultLatency.Observe(toMilliseconds(time.Since(t)), typ, label)
}

func wsDefaultLatencyE6(d time.Duration, typ, label string) {
	if d != 0 {
		wsDefaultLatency.Observe(toMilliseconds(d), typ, label)
	}
}

// wsPushLatencyE6 record latency with type and label
func wsPushLatencyE6(d time.Duration, typ, topic, appID string) {
	if d != 0 {
		wsPushLatency.Observe(toMilliseconds(d), typ, topic, appID)
	}
}

// wsMessageSize records raw value size for consumer message
func wsMessageSize(size int, topic string, appId string) {
	wsMessageHistogram.Observe(float64(size), topic, appId)
}

func wsUpgradeInc(path, platform, source, version string) {
	wsUpgradeCounter.Inc(path, platform, source, version)
}

func wsSyncInc(typ, appId string) {
	wsSyncCounter.Inc(typ, appId)
}

func wsSyncAdd(v int, typ, appId string) {
	wsSyncCounter.Add(float64(v), typ, appId)
}

// toMilliseconds returns the duration as a floating point number of milliseconds.
func toMilliseconds(d time.Duration) float64 {
	sec := d / time.Millisecond
	nsec := d % time.Millisecond
	return float64(sec) + float64(nsec)/1e6
}

var gMetricsMgr = newMetricsManager()

type metricsType uint8

const (
	metricsTypePush metricsType = iota + 1 // 推送消息耗时
)

var gMetricsMsgPool = sync.Pool{
	New: func() interface{} {
		return &metricsMessage{}
	},
}

func newMetricsMessage() *metricsMessage {
	msg := gMetricsMsgPool.Get().(*metricsMessage)
	msg.Refs = 1
	msg.ErrCount = 0
	msg.EndTimeE9 = msg.EndTimeE9[:0]
	msg.Sessions = msg.Sessions[:0]
	return msg
}

type metricsMessage struct {
	Refs          int32                   // 引用计数,当为零时被回收
	Type          metricsType             // 类型
	Push          *envelopev1.PushMessage // 推送消息
	AppID         string                  // app id
	RemoteAddr    string                  // app address
	WsStartTimeE9 int64                   // bgws收到消息时间
	WsEndTimeE9   int64                   // bgws处理完消息时间
	ErrCount      int                     // 发生失败数
	EndTimeE9     []int64                 // 多个session会记录多次
	Sessions      []string                // 写入成功的sessionId列表
}

func (m *metricsMessage) Retain() {
	atomic.AddInt32(&m.Refs, 1)
}

func (m *metricsMessage) Release() {
	if atomic.AddInt32(&m.Refs, -1) <= 0 {
		m.Push = nil
		gMetricsMsgPool.Put(m)
	}
}

type metricsManager struct {
	queue lockfree.Queue
	quit  atomic.Value
	log   glog.Logger
}

func newMetricsManager() *metricsManager {
	mgr := &metricsManager{
		queue: lockfree.NewQueue(),
		log:   glog.Default(),
	}

	return mgr
}

func (m *metricsManager) initLogger() {
	conf := getStaticConf()

	pushConf := conf.Log.convert(pushLogName)
	pushConf.Level = glog.InfoLevel
	pushConf.DisableCaller = true
	pushConf.DisableLevel = true

	m.log = glog.New(&pushConf)
	glog.Infof(context.Background(), "push log conf:%v", toJsonString(pushConf))
}

func (m *metricsManager) Get() *metricsMessage {
	return newMetricsMessage()
}

func (m *metricsManager) Send(msg *metricsMessage) {
	if msg == nil {
		return
	}

	msg.WsEndTimeE9 = nowUnixNano()

	if m.queue.Len() > getDynamicConf().MaxMetricsSize {
		m.writeMetrics(msg)
		msg.Release()
		WSErrorInc("metrics_mgr", "drop")
		return
	}

	m.queue.Enque(msg)
}

func (m *metricsManager) Start() {
	m.initLogger()

	go m.Loop()
}

func (m *metricsManager) Stop() {
	m.quit.Store(1)
	if m.log != nil {
		_ = m.log.Close()
	}
}

func (m *metricsManager) Loop() {
	for {
		if m.quit.Load() == 1 {
			break
		}

		if m.queue.Len() == 0 {
			time.Sleep(time.Millisecond * 5)
			continue
		}

		for {
			item := m.queue.Deque()
			if item == nil {
				break
			}
			msg := item.(*metricsMessage)
			m.sendMessage(msg)
		}
	}
}

func (m *metricsManager) sendMessage(msg *metricsMessage) {
	switch msg.Type {
	case metricsTypePush:
		m.writeMetrics(msg)
		m.writeTrace(msg)
		m.writeLog(msg)
		// 执行完后释放
		msg.Release()
	}
}

func (m *metricsManager) writeMetrics(msg *metricsMessage) {
	pmsg := msg.Push
	if pmsg.SdkTimeE9 == 0 {
		pmsg.SdkTimeE9 = pmsg.RequestTimeE9
	}

	for _, endTimeE9 := range msg.EndTimeE9 {
		if pmsg.InitTimeE9 > 0 {
			wsPushLatencyE6(time.Duration(endTimeE9-pmsg.InitTimeE9), "since_init_time", pmsg.Topic, msg.AppID)
		}
		wsPushLatencyE6(time.Duration(endTimeE9-pmsg.RequestTimeE9), "since_request_time", pmsg.Topic, msg.AppID)
		wsPushLatencyE6(time.Duration(endTimeE9-pmsg.SdkTimeE9), "since_sdk_recv_time", pmsg.Topic, msg.AppID)
	}

	wsPushLatencyE6(time.Duration(msg.WsEndTimeE9-msg.WsStartTimeE9), "ws_process_cost", pmsg.Topic, msg.AppID)
	wsMessageSize(len(pmsg.Data), pmsg.Topic, msg.AppID)
}

func (m *metricsManager) writeTrace(msg *metricsMessage) {
	pmsg := msg.Push
	if pmsg.TraceId == "" {
		return
	}
	startTime := time.Unix(0, pmsg.SdkTimeE9)
	tags := opentracing.Tags{
		"uid":          pmsg.UserId,
		"topic":        pmsg.Topic,
		"app_id":       msg.AppID,
		"msg_id":       pmsg.MessageId,
		"init_time_e9": pmsg.InitTimeE9,
		"req_time_e9":  pmsg.RequestTimeE9,
		"sdk_time_e9":  pmsg.SdkTimeE9,
		"ws_cost":      time.Duration(msg.WsEndTimeE9 - msg.WsStartTimeE9).String(),
	}
	span, _ := gtrace.Begin(context.Background(), "push_cost", gtrace.WithStartTime(startTime), tags, gtrace.WithUberTraceID(pmsg.TraceId))
	gtrace.Finish(span)
	traceID := gtrace.TraceIDFromSpan(span)
	pmsg.TraceId = traceID
}

func (m *metricsManager) writeLog(msg *metricsMessage) {
	pmsg := msg.Push
	if pmsg.PushType == envelopev1.PushType_PUSH_TYPE_PUBLIC {
		return
	}
	if !getDynamicConf().Log.CanWritePushLog(pmsg.UserId, msg.AppID, pmsg.Topic) {
		return
	}

	const (
		kSeparator = '\t' // 默认分隔符
		kEmpty     = '*'  // 空内容
	)

	b := strings.Builder{}
	b.Grow(128)
	// uid
	b.WriteString(strconv.FormatInt(pmsg.UserId, 10))
	b.WriteByte(kSeparator)
	// app info
	b.WriteString(msg.AppID)
	b.WriteByte(kSeparator)
	b.WriteString(msg.RemoteAddr)
	b.WriteByte(kSeparator)

	// data info
	b.WriteString(pmsg.Topic)
	b.WriteByte(kSeparator)
	if pmsg.TraceId != "" {
		b.WriteString(pmsg.TraceId)
	} else {
		b.WriteByte(kEmpty)
	}
	b.WriteByte(kSeparator)
	if pmsg.MessageId != "" {
		b.WriteString(pmsg.MessageId)
	} else {
		b.WriteByte(kEmpty)
	}
	b.WriteByte(kSeparator)
	b.WriteString(strconv.Itoa(len(pmsg.Data)))
	b.WriteByte(kSeparator)

	// session info
	sessionSize := len(msg.Sessions)
	b.WriteString(strconv.Itoa(sessionSize)) // 成功写入数量
	b.WriteByte(kSeparator)
	b.WriteString(strconv.Itoa(msg.ErrCount)) // 失败写入数量
	b.WriteByte(kSeparator)
	if sessionSize > 0 {
		b.WriteString(msg.Sessions[0])
	} else {
		b.WriteByte(kEmpty)
	}
	b.WriteByte(kSeparator)

	// timestamp
	b.WriteString(strconv.FormatInt(msg.WsEndTimeE9, 10))
	b.WriteByte(kSeparator)
	if pmsg.InitTimeE9 != 0 {
		b.WriteString(time.Duration(msg.WsEndTimeE9 - pmsg.InitTimeE9).String())
	} else {
		b.WriteByte(kEmpty)
	}

	b.WriteByte(kSeparator)
	b.WriteString(time.Duration(msg.WsEndTimeE9 - pmsg.RequestTimeE9).String())
	b.WriteByte(kSeparator)
	b.WriteString(time.Duration(msg.WsEndTimeE9 - pmsg.SdkTimeE9).String())
	b.WriteByte(kSeparator)
	b.WriteString(time.Duration(msg.WsEndTimeE9 - msg.WsStartTimeE9).String())

	b.WriteByte(kSeparator)
	b.WriteString(strconv.FormatUint(pmsg.LabelFlags, 10))

	m.log.Info(context.Background(), b.String())
}
