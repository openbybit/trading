package ws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var errNotFoundPublicTopic = errors.New("not found public topic group")

var gPublicMgr = newPublicMgr()

func newPublicMgr() *publicMgr {
	return &publicMgr{prefixMap: make(map[string]PublicTopicConf)}
}

// PublicTopicConf 公有推送配置
// nolint
type PublicTopicConf struct {
	Topic    string `json:"topic"`                          // 用户订阅topic
	PushMode string `json:"push_mode,options=[full,delta]"` // 推送模式
}

type publicMgr struct {
	workers   sync.Map                   // topic->worker
	prefixMap map[string]PublicTopicConf // 前缀匹配
	mux       sync.RWMutex               //
}

// Info 用于admin打印信息
func (mgr *publicMgr) Info() interface{} {
	type Info struct {
		Mode        string `json:"mode"`
		SessionSize int    `json:"session_size"`
	}

	type Result struct {
		Workers  map[string]Info   `json:"workers,omitempty"`
		Prefixes map[string]string `json:"prefixes,omitempty"`
	}

	result := &Result{
		Workers:  make(map[string]Info),
		Prefixes: make(map[string]string),
	}

	mgr.workers.Range(func(key, value any) bool {
		w, _ := value.(publicWorker)
		if w != nil {
			conf := w.Config()
			info := Info{
				Mode:        conf.PushMode,
				SessionSize: w.Size(),
			}

			result.Workers[conf.Topic] = info
		}

		return true
	})

	mgr.mux.RLock()
	for _, item := range mgr.prefixMap {
		result.Prefixes[item.Topic] = item.PushMode
	}
	mgr.mux.RUnlock()

	return result
}

// Run 根据配置启动工作线程,每个topic对应一个worker,幂等,重复的会忽略,只增不减
func (mgr *publicMgr) Run(list []PublicTopicConf) {
	if len(list) == 0 {
		return
	}

	for _, c := range list {
		_ = mgr.Add(c)
	}
}

func (mgr *publicMgr) Stop() {
	keys := []string{}
	mgr.workers.Range(func(key, value any) bool {
		keys = append(keys, key.(string))
		w, _ := value.(publicWorker)
		w.Stop()
		return true
	})

	for _, key := range keys {
		mgr.workers.Delete(key)
	}

}

func (mgr *publicMgr) Get(topic string) publicWorker {
	if v, ok := mgr.workers.Load(topic); ok {
		return v.(publicWorker)
	}

	return nil
}

// Add 添加配置
func (mgr *publicMgr) Add(conf PublicTopicConf) error {
	// 判断是否是前缀模糊匹配topic
	conf.Topic = strings.TrimSpace(conf.Topic)
	isDynamic, prefix, err := mgr.checkDynamicTopic(conf.Topic)
	if err != nil {
		glog.Errorf(context.Background(), "checkDynamicTopic err: %v %v", conf, err)
		return err
	}
	if isDynamic {
		if conf.PushMode == "" {
			glog.Errorf(context.Background(), "invalid PushMode: %v", conf)
			return fmt.Errorf("invalid push mode")
		}

		mgr.mux.Lock()
		defer mgr.mux.Unlock()
		old, ok := mgr.prefixMap[prefix]
		if ok {
			if conf.PushMode != old.PushMode {
				glog.Errorf(context.Background(), "topic config conflict: %v, %v", conf, old)
				return fmt.Errorf("topic config conflict: %v, %v", conf, old)
			}
			return nil
		}
		mgr.prefixMap[prefix] = conf
		return nil
	}

	// 普通topic,直接创建队列,幂等
	if w := mgr.Get(conf.Topic); w != nil {
		return nil
	}

	if w := mgr.create(conf); w == nil {
		glog.Errorf(context.Background(), "create err: %v", conf)
		return fmt.Errorf("create public worker fail, %v", conf)
	}

	return nil
}

// checkDynamicTopic 判断是否是动态topic
func (mgr *publicMgr) checkDynamicTopic(topic string) (bool, string, error) {
	index := strings.IndexByte(topic, '@')
	if index == -1 {
		return false, "", nil
	}

	if index == 0 || index == len(topic)-1 {
		glog.Errorf(context.Background(), "invalid topic: %v", topic)
		return false, "", fmt.Errorf("invalid public prefix topic, %v", topic)
	}

	// 动态topic必须含有{}
	if !strings.ContainsAny(topic[index+1:], "{}") {
		return false, "", nil
	}

	return true, topic[:index], nil
}

// create 创建worker
func (mgr *publicMgr) create(conf PublicTopicConf) publicWorker {
	v := newPublicWorker(conf)
	if v == nil {
		glog.Errorf(context.Background(), "create public worker failed, %v", conf)
		WSErrorInc("public_mgr", "create_worker_fail")
		return nil
	}

	actual, loaded := mgr.workers.LoadOrStore(conf.Topic, v)
	if !loaded {
		v.Start()
		gConfigMgr.AddTopic(topicTypePublic, []string{conf.Topic})
	}

	return actual.(publicWorker)
}

func (mgr *publicMgr) getPrefixConf(prefix string) (PublicTopicConf, bool) {
	mgr.mux.RLock()
	conf, ok := mgr.prefixMap[prefix]
	mgr.mux.RUnlock()
	return conf, ok
}

// getOrCreate 动态创建,前缀匹配
func (mgr *publicMgr) getOrCreate(topic string) publicWorker {
	if w := mgr.Get(topic); w != nil {
		return w
	}

	// 必须以@分割,且前后都需要有字符
	index := strings.IndexAny(topic, "@")
	if index < 1 || index >= len(topic)-1 {
		WSErrorInc("public_invalid_dyn_topic", topic)
		return nil
	}

	prefix := topic[:index]
	conf, ok := mgr.getPrefixConf(prefix)
	if !ok {
		WSErrorInc("public_no_prefix_config", prefix)
		return nil
	}

	conf.Topic = topic
	return mgr.create(conf)
}

func (mgr *publicMgr) Write(msg *publicMessage) error {
	w := mgr.getOrCreate(msg.Topic)
	if w == nil {
		WSErrorInc("public_mgr_write_fail", msg.Topic)
		return errNotFoundPublicTopic
	}

	w.Write(msg)
	return nil
}

func (mgr *publicMgr) OnSubscribe(sess Session, topics []string) {
	if len(topics) == 0 {
		return
	}

	for _, t := range topics {
		w := mgr.Get(t)
		if w != nil {
			w.AddSession(sess)
		}
	}

	return
}

func (mgr *publicMgr) OnUnsubscribe(sess Session, topics []string) {
	for _, t := range topics {
		w := mgr.Get(t)
		if w != nil {
			w.DelSession(sess.ID())
		}
	}
}

func (mgr *publicMgr) OnSessionStop(sess Session) {
	topics := sess.GetClient().GetTopics()
	for t := range topics.items {
		w := mgr.Get(t)
		if w != nil {
			w.DelSession(sess.ID())
		}
	}
}
