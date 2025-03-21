package ws

import (
	"context"
	"errors"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func newPublicWorkerFull(conf PublicTopicConf) *publicWorkerFull {
	w := &publicWorkerFull{}
	w.conf = conf
	w.cnd = sync.NewCond(&w.mux)
	return w
}

type publicWorkerFull struct {
	publicWorkerBase
	newSessions map[string]Session // 新建连接
	last        *publicMessage     // 最新数据
	cache       []byte             // 缓存数据
	running     bool               //
	mux         sync.Mutex         //
	cnd         *sync.Cond         //
}

func (w *publicWorkerFull) Start() {
	if !w.running {
		glog.Infof(context.Background(), "public_worker full start, topic: %v", w.conf.Topic)
		w.running = true
		go w.loop()
	}
}

func (w *publicWorkerFull) Stop() {
	glog.Infof(context.Background(), "public_worker full stop, topic: %v", w.conf.Topic)
	w.mux.Lock()
	w.running = false
	w.mux.Unlock()
	w.cnd.Signal()
}

func (w *publicWorkerFull) AddSession(sess Session) {
	w.mux.Lock()
	if w.newSessions == nil {
		w.newSessions = make(map[string]Session)
	}
	w.newSessions[sess.ID()] = sess
	w.mux.Unlock()
	w.cnd.Signal()
}

func (w *publicWorkerFull) Write(msg *publicMessage) {
	w.mux.Lock()
	if w.last != nil {
		WSCounterInc("public_full_discard", w.conf.Topic)
		// glog.Debug(context.Background(), "public_full_discard", glog.ByteString("last", w.last.Data))
	}
	w.last = msg
	w.mux.Unlock()
	w.cnd.Signal()
}

func (w *publicWorkerFull) loop() {
	for {
		w.mux.Lock()
		for w.running && w.last == nil && len(w.newSessions) == 0 {
			w.cnd.Wait()
		}

		running := w.running

		newSessions := w.newSessions
		w.newSessions = nil

		msg := w.last
		w.last = nil
		w.mux.Unlock()

		if !running {
			break
		}

		// new sessions
		if len(newSessions) > 0 {
			for _, sess := range newSessions {
				psess, ok := w.DoAddSession(sess)
				if !ok {
					// glog.Debug(context.Background(), "public full add session fail", glog.String("sess_id", sess.ID()))
					continue
				}

				if w.cache != nil {
					_ = psess.Write(w.cache)
				}
			}
		}

		if msg != nil {
			w.Broadcast(msg)
			w.cache = msg.Data
		}
	}
}

// Broadcast 广播数据
func (w *publicWorkerFull) Broadcast(msg *publicMessage) {
	data := msg.Data
	start := nowUnixNano()
	count := 0
	invalidSessions := make([]string, 0)
	w.sessions.Range(func(key, value any) bool {
		psess, _ := value.(*publicSession)
		err := psess.Write(data)
		if err == nil {
			count++
		} else if errors.Is(err, errSessionClosed) {
			invalidSessions = append(invalidSessions, psess.ID())
		}
		// glog.Debug(context.Background(), "public full write", glog.String("sess_id", psess.ID()), glog.ByteString("data", data), glog.NamedError("err", err))
		return true
	})

	w.DeleteSessions(invalidSessions)
	w.recordMetric(msg, start, count)
}
