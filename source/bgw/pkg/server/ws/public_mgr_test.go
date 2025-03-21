package ws

import (
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"github.com/stretchr/testify/assert"
)

func newPublicMessage(topic string, typ messageType, data []byte) *publicMessage {
	return &publicMessage{PushMessage: &envelopev1.PushMessage{Topic: topic, Data: data, PushType: envelopev1.PushType_PUSH_TYPE_PUBLIC, MessageType: typ}}
}

func TestPublicMgr(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	t.Run("new_public_worker", func(t *testing.T) {
		assert.NotNil(t, newPublicWorker(PublicTopicConf{Topic: "topic", PushMode: "full"}))
		assert.NotNil(t, newPublicWorker(PublicTopicConf{Topic: "delta", PushMode: "delta"}))
		assert.Nil(t, newPublicWorker(PublicTopicConf{Topic: "invalid_topic"}))
	})

	t.Run("test_run", func(t *testing.T) {
		mgr := newPublicMgr()
		// 忽略空的
		mgr.Run(nil)
		// add one
		mgr.Run([]PublicTopicConf{{Topic: "topic1", PushMode: "full"}})
	})

	t.Run("test_add", func(t *testing.T) {
		var err error
		mgr := newPublicMgr()

		// normal
		err = mgr.Add(PublicTopicConf{Topic: "topic1", PushMode: "full"})
		assert.Nil(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "topic1", PushMode: "full"})
		assert.Nil(t, err)
		// add normal error
		err = mgr.Add(PublicTopicConf{Topic: "topic_fail", PushMode: ""})
		assert.Error(t, err)

		// prefix
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.BTCUSDT", PushMode: "delta"}) // static
		assert.Nil(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.{symbol}", PushMode: "delta"})
		assert.Nil(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.{symbol}", PushMode: "delta"})
		assert.Nil(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.{symbol}", PushMode: "full"})
		assert.Error(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.{symbol}", PushMode: ""})
		assert.Error(t, err)

		err = mgr.Add(PublicTopicConf{Topic: "@m1.{symbol}", PushMode: "delta"})
		assert.Error(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@", PushMode: "delta"})
		assert.Error(t, err)

		assert.NotNil(t, mgr.Info())
	})

	t.Run("test_write", func(t *testing.T) {
		// normal
		mgr := newPublicMgr()
		subTopic := "public_topic"
		err := mgr.Add(PublicTopicConf{Topic: subTopic, PushMode: "full"})
		assert.Nil(t, err)
		err = mgr.Write(newPublicMessage(subTopic, messageTypeUnknown, []byte("hello")))
		assert.Nil(t, err)
		err = mgr.Write(newPublicMessage("unknown_topic", messageTypeUnknown, []byte("hello")))
		assert.Error(t, err)

		// prefix, lazy init
		err = mgr.Add(PublicTopicConf{Topic: "orderbook", PushMode: "full"})
		assert.Nil(t, err)
		err = mgr.Add(PublicTopicConf{Topic: "orderbook@m1.{symbol}", PushMode: "delta"})
		assert.Nil(t, err)

		err = mgr.Write(newPublicMessage("orderbook", messageTypeUnknown, []byte("hello")))
		assert.Nil(t, err)

		prefixTopic := "orderbook@m1.BTCUSDT"
		err = mgr.Write(newPublicMessage(prefixTopic, messageTypeReset, []byte("hello")))
		assert.Nil(t, err)
		err = mgr.Write(newPublicMessage(prefixTopic, messageTypeReset, []byte("hello")))
		assert.Nil(t, err)
		err = mgr.Write(newPublicMessage("unknown_prefix_topic@xxx", messageTypeReset, []byte("hello")))
		assert.Error(t, err)

		mgr.Stop()
	})

	t.Run("test_session", func(t *testing.T) {
		subTopic := "public_topic"
		subTopicList := []string{subTopic}

		mgr := newPublicMgr()
		_ = mgr.Add(PublicTopicConf{Topic: subTopic, PushMode: "full"})
		_ = mgr.Write(newPublicMessage(subTopic, messageTypeReset, []byte("hello")))

		sess := newMockSession(1)
		mgr.OnSubscribe(sess, nil)

		sess.GetClient().Subscribe(subTopicList)
		mgr.OnSubscribe(sess, subTopicList)

		sess.GetClient().Unsubscribe(subTopicList)
		mgr.OnUnsubscribe(sess, subTopicList)

		sess.GetClient().Subscribe(subTopicList)
		mgr.OnSessionStop(sess)

		mgr.Stop()
	})
}

func TestPublicWorker(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	t.Run("base", func(t *testing.T) {
		topic := "demo"
		w := publicWorkerBase{conf: PublicTopicConf{Topic: topic, PushMode: pushModeDelta}}

		sess1 := newMockSession(1)
		_, res := w.DoAddSession(sess1)
		assert.False(t, res)
		sess1.GetClient().Subscribe([]string{topic})
		_, res = w.DoAddSession(sess1)
		assert.True(t, res)
		_, res = w.DoAddSession(sess1)
		assert.False(t, res)
		assert.True(t, w.HasSession(sess1.ID()))

		// session closed
		sess2 := newMockSession(2)
		sess2.GetClient().Subscribe([]string{topic})
		psess2, _ := w.DoAddSession(sess2)
		psess2.Stop()

		sess3 := newMockSession(3)
		sess3.GetClient().Subscribe([]string{topic})
		w.DoAddSession(sess3)
		sess3.SetChanFull(true)

		w.DelSession(sess1.ID())
	})

	t.Run("full", func(t *testing.T) {
		topic := "topic"
		w := newPublicWorkerFull(PublicTopicConf{Topic: topic, PushMode: "full"})
		w.Start()

		w.Write(newPublicMessage(topic, 0, []byte("cache")))

		sess := newMockSession(1)
		sess.GetClient().Subscribe([]string{topic})
		w.AddSession(sess)

		w.Write(newPublicMessage(topic, 0, []byte("hello")))
		w.Write(newPublicMessage(topic, 0, []byte("world")))
		time.Sleep(time.Millisecond * 10)

		// duplicate add
		w.AddSession(sess)
		time.Sleep(time.Millisecond * 10)

		w.Stop()
		time.Sleep(time.Millisecond * 10)
	})

	t.Run("delta", func(t *testing.T) {
		topic := "delta"
		w := newPublicWorkerDelta(PublicTopicConf{Topic: "delta", PushMode: "delta"})
		w.Start()

		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("first")))
		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		time.Sleep(time.Millisecond * 10)

		sess := newMockSession(1)
		sess.GetClient().Subscribe([]string{topic})
		w.AddSession(sess)

		w.Write(newPublicMessage(topic, messageTypeReset, []byte("reset")))
		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		w.Write(newPublicMessage(topic, messageTypeDelta, []byte("delta")))
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("snapshot")))
		w.Write(newPublicMessage(topic, 0, []byte("unknown")))
		time.Sleep(time.Millisecond * 10)

		old := getAppConf().MaxSnapshotSize
		getAppConf().MaxSnapshotSize = 1
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("snapshot")))
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("snapshot")))
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("snapshot")))
		w.Write(newPublicMessage(topic, messageTypeSnapshot, []byte("snapshot")))
		time.Sleep(time.Millisecond * 10)
		getAppConf().MaxSnapshotSize = old

		// duplicate add
		w.AddSession(sess)
		time.Sleep(time.Millisecond * 10)
		w.Stop()
	})

	t.Run("full broadcast", func(t *testing.T) {
		topic := "full"
		w := newPublicWorkerFull(PublicTopicConf{Topic: topic, PushMode: pushModeFull})

		s1 := newMockSession(1)
		s1.GetClient().Subscribe([]string{topic})
		_, _ = w.DoAddSession(s1)

		s2 := newMockSession(2)
		s2.GetClient().Subscribe([]string{topic})
		_, _ = w.DoAddSession(s2)
		s2.Stop()

		s3 := newMockSession(3)
		s3.GetClient().Subscribe([]string{topic})
		s3.SetChanFull(true)
		_, _ = w.DoAddSession(s3)

		w.Broadcast(newPublicMessage(topic, messageTypeUnknown, []byte("data")))
	})

	t.Run("delta broadcast", func(t *testing.T) {
		topic := "delta"
		w := newPublicWorkerDelta(PublicTopicConf{Topic: topic, PushMode: "delta"})

		s1 := newMockSession(1)
		s1.GetClient().Subscribe([]string{topic})
		ps1, _ := w.DoAddSession(s1)
		ps1.SetHasSendSnapshot()

		s2 := newMockSession(2)
		s2.GetClient().Subscribe([]string{topic})
		ps2, _ := w.DoAddSession(s2)
		ps2.SetHasSendSnapshot()
		ps2.Stop()

		s3 := newMockSession(3)
		s3.GetClient().Subscribe([]string{topic})
		_, _ = w.DoAddSession(s3)

		s4 := newMockSession(3)
		s4.GetClient().Subscribe([]string{topic})
		s4.SetChanFull(true)
		ps4, _ := w.DoAddSession(s4)
		ps4.SetHasSendSnapshot()

		w.Broadcast(newPublicMessage(topic, messageTypeDelta, []byte("delta")), true)
		w.Broadcast(newPublicMessage(topic, messageTypeDelta, []byte("snapshot")), false)
	})
}
