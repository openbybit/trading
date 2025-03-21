package ws

import (
	"sync"
	"sync/atomic"

	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
)

type messageType = envelopev1.MessageType

const (
	messageTypeUnknown  messageType = envelopev1.MessageType_MESSAGE_TYPE_UNSPECIFIED // 未定义类型
	messageTypeSnapshot messageType = envelopev1.MessageType_MESSAGE_TYPE_SNAPSHOT    // 全量数据,不广播,缓存
	messageTypeDelta    messageType = envelopev1.MessageType_MESSAGE_TYPE_DELTA       // 增量数据,广播,缓存
	messageTypeReset    messageType = envelopev1.MessageType_MESSAGE_TYPE_RESET       // 全量数据,广播,缓存
	// messageTypePassthrough messageType = envelopev1.MessageType_MESSAGE_TYPE_PASSTHROUGH // 全量数据,广播,不缓存,暂不支持,没有场景需要
)

const (
	pushModeFull  = "full"
	pushModeDelta = "delta"
)

type publicMessage struct {
	*envelopev1.PushMessage
	appId      string
	remoteAddr string
}

type publicWorker interface {
	Start()
	Stop()
	Config() *PublicTopicConf
	Size() int
	AddSession(sess Session)
	DelSession(id string)
	Write(msg *publicMessage)
}

type publicSession struct {
	sess            Session // 连接
	hasSendSnapshot bool    // 标记是否发送过snapshot
}

func (ps *publicSession) SetHasSendSnapshot() {
	ps.hasSendSnapshot = true
}

func (ps *publicSession) ID() string {
	return ps.sess.ID()
}

func (ps *publicSession) Stop() {
	ps.sess.Stop()
}

func (ps *publicSession) Write(data []byte) error {
	return ps.sess.Write(&Message{Type: MsgTypePush, Data: data})
}

// publicWorkerBase
// 1:维护订阅topic到sessions的映射关系
// 2:维护新连接待广播队列
// 3:维护消息广播队列
// 4:维护cache
type publicWorkerBase struct {
	conf     PublicTopicConf // 配置信息
	sessions sync.Map        // session_id->publicSession映射关系
	size     atomic.Int32    // session大小
}

func newPublicWorker(conf PublicTopicConf) publicWorker {
	switch conf.PushMode {
	case pushModeFull:
		return newPublicWorkerFull(conf)
	case pushModeDelta:
		return newPublicWorkerDelta(conf)
	default:
		return nil
	}
}

func (w *publicWorkerBase) Config() *PublicTopicConf {
	return &w.conf
}

func (w *publicWorkerBase) Size() int {
	return int(w.size.Load())
}

func (w *publicWorkerBase) HasSession(id string) bool {
	_, ok := w.sessions.Load(id)
	return ok
}

func (w *publicWorkerBase) DoAddSession(sess Session) (*publicSession, bool) {
	if !sess.GetClient().HasTopic(w.conf.Topic) {
		return nil, false
	}

	ps, loaded := w.sessions.LoadOrStore(sess.ID(), &publicSession{sess: sess, hasSendSnapshot: false})
	if loaded {
		return ps.(*publicSession), false
	}

	w.updateSize(1)
	return ps.(*publicSession), true
}

func (w *publicWorkerBase) DelSession(id string) {
	_, loaded := w.sessions.LoadAndDelete(id)
	if loaded {
		w.updateSize(-1)
	}
}

// DeleteSessions 删除无效sessions
func (w *publicWorkerBase) DeleteSessions(sessions []string) {
	if len(sessions) == 0 {
		return
	}
	count := int32(0)
	for _, sessId := range sessions {
		_, ok := w.sessions.LoadAndDelete(sessId)
		if ok {
			count++
		}
	}
	w.updateSize(-count)
}

func (w *publicWorkerBase) updateSize(delta int32) {
	w.size.Add(delta)
	WSGauge(float64(w.size.Load()), "public_topic_size", w.conf.Topic)
}

func (w *publicWorkerBase) recordMetric(msg *publicMessage, startUnixNano int64, sendCount int) {
	WSGauge(float64(sendCount), "public_broadcast", w.conf.Topic)
	mt := gMetricsMgr.Get()
	mt.Type = metricsTypePush
	mt.AppID = msg.appId
	mt.RemoteAddr = msg.remoteAddr
	mt.Push = msg.PushMessage
	mt.WsStartTimeE9 = startUnixNano
	mt.EndTimeE9 = append(mt.EndTimeE9, nowUnixNano())
	gMetricsMgr.Send(mt)
}
