package ws

import (
	"container/list"
	"context"
	"errors"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

type publicStatus uint8

const (
	publicStatusStopped  = publicStatus(iota) // 停止状态
	publicStatusNotReady                      // 服务尚未就绪,需要等待第一个snapshot才能广播
	publicStatusRunning                       // 正常运行中
)

func newPublicWorkerDelta(conf PublicTopicConf) *publicWorkerDelta {
	w := &publicWorkerDelta{}
	w.conf = conf
	w.cnd = sync.NewCond(&w.mux)
	w.queue = list.New()
	return w
}

// publicWorkerDelta 增量同步消息
type publicWorkerDelta struct {
	publicWorkerBase
	newSessions  map[string]Session // 新建连接
	queue        *list.List         // 消息队列
	caches       [][]byte           // 缓存数据
	status       publicStatus       // 当前运行状态
	snapshotSize int                // 当前队列中snapshot个数
	mux          sync.RWMutex       //
	cnd          *sync.Cond         //
}

func (p *publicWorkerDelta) AddSession(sess Session) {
	p.mux.Lock()
	if p.newSessions == nil {
		p.newSessions = make(map[string]Session)
	}
	p.newSessions[sess.ID()] = sess
	p.mux.Unlock()
	p.cnd.Signal()
}

func (p *publicWorkerDelta) Start() {
	if p.status == publicStatusStopped {
		glog.Infof(context.Background(), "public_worker delta start, topic: %v", p.conf.Topic)
		p.status = publicStatusNotReady
		go p.loop()
	}
}

func (p *publicWorkerDelta) Stop() {
	glog.Infof(context.Background(), "public_worker delta stop, topic: %v", p.conf.Topic)
	p.mux.Lock()
	p.status = publicStatusStopped
	p.mux.Unlock()
	p.cnd.Signal()
}

func (p *publicWorkerDelta) Write(msg *publicMessage) {
	p.mux.Lock()
	switch p.status {
	case publicStatusNotReady:
		if msg.MessageType == messageTypeReset || msg.MessageType == messageTypeSnapshot {
			// 首次需要强制给所有用户广播一次
			msg.MessageType = messageTypeReset
			p.queue.PushBack(msg)
			p.status = publicStatusRunning
		}
	case publicStatusRunning:
		switch msg.MessageType {
		case messageTypeSnapshot:
			p.snapshotSize++
			maxSize := getAppConf().MaxSnapshotSize
			if maxSize > 0 && p.snapshotSize >= maxSize {
				msg.MessageType = messageTypeReset
				p.snapshotSize = 0
				p.queue.Init()
				WSCounterInc("public_delta_clear", p.conf.Topic)
				// glog.Debug(context.Background(), "public delta reset queue", glog.ByteString("data", msg.Data))
			}
		case messageTypeReset:
			p.queue.Init()
		}

		p.queue.PushBack(msg)
	}
	size := p.queue.Len()
	p.mux.Unlock()
	p.cnd.Signal()
	WSGauge(float64(size), "public_delta_queue_size", p.conf.Topic)
}

func (p *publicWorkerDelta) loop() {
	for {
		p.mux.Lock()
		for p.status != publicStatusStopped && p.queue.Len() == 0 && len(p.newSessions) == 0 {
			p.cnd.Wait()
		}

		running := p.status != publicStatusStopped

		// new sessions
		newSessions := p.newSessions
		p.newSessions = nil

		// new messages
		var msg *publicMessage
		if p.queue.Len() > 0 {
			msg = p.queue.Remove(p.queue.Front()).(*publicMessage)
			if msg.MessageType == messageTypeSnapshot {
				p.snapshotSize--
			}
		}
		p.mux.Unlock()

		if !running {
			break
		}

		// new sessions
		if len(newSessions) > 0 {
			for _, sess := range newSessions {
				psess, ok := p.DoAddSession(sess)
				if !ok {
					glog.Debugf(context.Background(), "public delta add session fail, sessId=%v", sess.ID())
					continue
				}

				// if enableDebug {
				// 	glog.Debugf(context.Background(), "public delta send cache, sessId=%v, cacheSize=%v", sess.ID(), len(p.caches))
				// }

				if len(p.caches) > 0 {
					psess.SetHasSendSnapshot()
					for _, data := range p.caches {
						_ = psess.Write(data)
						// glog.Debug(context.Background(), "public delta write cache", glog.String("sess_id", psess.ID()), glog.ByteString("data", data), glog.NamedError("err", err))
					}
				}
			}
		}

		// new message
		if msg != nil {
			switch msg.MessageType {
			case messageTypeSnapshot:
				p.caches = p.caches[:0]
				p.caches = append(p.caches, msg.Data)
			case messageTypeReset:
				p.Broadcast(msg, false)
				p.caches = p.caches[:0]
				p.caches = append(p.caches, msg.Data)
			case messageTypeDelta:
				p.Broadcast(msg, true)
				if len(p.caches) > 0 {
					p.caches = append(p.caches, msg.Data)
				} else {
					WSErrorInc("public_delta", "invalid_cache")
				}
			// case messageTypePassthrough:
			// 	p.Broadcast(msg.Data, false)
			default:
				WSErrorInc("public_delta_unknown_type", msg.MessageType.String())
			}
			WSGauge(float64(len(p.caches)), "public_delta_cache_size", p.conf.Topic)
		}
	}
}

func (p *publicWorkerDelta) Broadcast(msg *publicMessage, isDelta bool) {
	data := msg.Data
	start := nowUnixNano()
	count := 0
	invalidSessions := make([]string, 0)
	p.sessions.Range(func(key, value any) bool {
		psess, _ := value.(*publicSession)

		if !isDelta {
			psess.SetHasSendSnapshot()
		} else if !psess.hasSendSnapshot {
			// 第一个包非snapshot
			WSErrorInc("public_delta", "no_snapshot")
			glog.Errorf(context.Background(), "public worker write fail, no snapshot, id=%v", psess.ID())
			psess.Stop()
			invalidSessions = append(invalidSessions, psess.ID())
			return true
		}

		err := psess.Write(data)
		if err != nil {
			if errors.Is(err, errSessionWriteChannelDiscard) {
				glog.Infof(context.Background(), "public worker write discard, id=%v", psess.ID())
				psess.Stop()
			}
			invalidSessions = append(invalidSessions, psess.ID())
		} else {
			count++
		}

		// glog.Debug(context.Background(), "public delta write", glog.String("sess_id", psess.ID()), glog.ByteString("data", data), glog.NamedError("err", err))
		return true
	})

	p.DeleteSessions(invalidSessions)
	p.recordMetric(msg, start, count)
}
