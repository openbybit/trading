package ws

import (
	"math"
	"math/rand"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"github.com/stretchr/testify/assert"

	"bgw/pkg/server/ws/mock"
)

func TestSplitMessage(t *testing.T) {
	sconf := getDynamicConf()
	t.Run("random", func(t *testing.T) {
		for i := 1; i < 10; i++ {
			sconf.MaxSyncMsgSize = i

			acc := newAcceptor(mock.NewGrpcServerStream(), "", "", nil, nil)
			ex := newExchange()
			msg := &envelopev1.SubscribeResponse{
				Cmd:    envelopev1.Command_COMMAND_SYNC,
				Users:  []*envelopev1.User{},
				Events: []*envelopev1.Event{},
			}

			userSize := rand.Intn(100) + 1
			eventSize := rand.Intn(100) + 1
			expUserSize := int(math.Ceil(float64(userSize) / float64(sconf.MaxSyncMsgSize)))
			expEventSize := int(math.Ceil(float64(eventSize) / float64(sconf.MaxSyncMsgSize)))
			expectSize := expUserSize + expEventSize
			if userSize+eventSize <= sconf.MaxSyncMsgSize {
				expectSize = 1
			}
			for i := 0; i < userSize; i++ {
				msg.Users = append(msg.Users, &envelopev1.User{MemberId: int64(rand.Intn(100000))})
			}
			for i := 0; i < eventSize; i++ {
				msg.Events = append(msg.Events, &envelopev1.Event{Type: envelopev1.EventType_EVENT_TYPE_SESSION_ONLINE, UserId: int64(rand.Intn(100000))})
			}

			if res := ex.sendAndSplitMsg(acc, msg); res != expectSize {
				t.Errorf("should be %d-> %d, %d, %d, userSize=%v, eventSize=%v, msgSize=%v", expectSize, res, expUserSize, expEventSize, userSize, eventSize, sconf.MaxSyncMsgSize)
			}
		}
	})

	t.Run("sendAndSplitMsg", func(t *testing.T) {
		dconf := getDynamicConf()
		old := dconf.MaxSyncMsgSize
		dconf.MaxSyncMsgSize = 0
		acc := newAcceptor(mock.NewGrpcServerStream(), "", "", nil, nil)
		msg := &envelopev1.SubscribeResponse{
			Users: []*envelopev1.User{
				{MemberId: 1},
			},
		}
		ex := newExchange()
		ex.sendAndSplitMsg(acc, msg)
		dconf.MaxSyncMsgSize = old
	})
}

func TestCheckUserShard(t *testing.T) {
	accGray := newAcceptor(nil, "id", "appid", []string{"t1"}, &acceptorOptions{ShardIndex: 2, ShardTotal: 2})
	assert.False(t, checkUserShard(accGray, 1))
	acc1 := newAcceptor(nil, "id", "appid", []string{"t1"}, &acceptorOptions{ShardIndex: 0, ShardTotal: 2})
	assert.False(t, checkUserShard(acc1, 1))
	assert.True(t, checkUserShard(acc1, 2))
}

func TestIsFocusEvents(t *testing.T) {
	if isFocusEvents(0, uint64(ActionSessionOnline)) {
		t.Error("should false")
	}

	if isFocusEvents(uint64(ActionSessionOnline|ActionSessionOffline), uint64(ActionSessionSub)) {
		t.Error("should false")
	}

	if !isFocusEvents(uint64(ActionSessionOnline|ActionSessionOffline), uint64(ActionSessionOnline)) {
		t.Error("should true")
	}

	if !isFocusEvents(uint64(ActionSessionOnline|ActionSessionOffline), uint64(ActionSessionOffline)) {
		t.Error("should true")
	}

	if !isFocusEvents(uint64(ActionSessionOnline|ActionSessionOffline), uint64(ActionSessionOnline|ActionSessionOffline)) {
		t.Error("should true")
	}
}

func TestDispatchMessage(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	initExchange()
	uid := int64(1)
	topic := "t1"
	topics := []string{topic}
	acc := newAcceptor(nil, "id", "appid", topics, nil)
	t.Run("single invalid message", func(t *testing.T) {
		// invalid uid
		DispatchMessage(acc, &envelopev1.SubscribeRequest{})
		msg := &envelopev1.SubscribeRequest{MemberId: uid, Topics: topics, Header: &envelopev1.Header{}}
		// invalid data
		DispatchMessage(acc, msg)
	})

	t.Run("no_sessions", func(t *testing.T) {
		// no session
		sess := newMockSession(uid)
		sess.client.Subscribe([]string{"other_topic"})

		_, _ = GetUserMgr().Bind(uid, sess)

		msg := &envelopev1.SubscribeRequest{MemberId: uid, Topics: topics, Header: &envelopev1.Header{}}
		DispatchMessage(acc, msg)
		GetUserMgr().Unbind(uid, sess.ID())
	})

	t.Run("single", func(t *testing.T) {
		sess := newMockSession(uid)
		sess.client.Subscribe(topics)
		_, _ = GetUserMgr().Bind(uid, sess)

		msg := &envelopev1.SubscribeRequest{MemberId: uid, Topics: []string{topic}, Header: &envelopev1.Header{}, Data: []byte("aa")}
		DispatchMessage(acc, msg)
		GetUserMgr().Unbind(uid, sess.ID())
	})

	t.Run("batch", func(t *testing.T) {
		sess := newMockSession(uid)
		sess.client.Subscribe(topics)
		_, _ = GetUserMgr().Bind(uid, sess)

		msg := &envelopev1.SubscribeRequest{PushMessages: []*envelopev1.PushMessage{
			{UserId: uid, Topic: topic, Data: []byte("aa")},
		}}
		DispatchMessage(acc, msg)
		GetUserMgr().Unbind(uid, sess.ID())
	})

	t.Run("public", func(t *testing.T) {
		msg := &envelopev1.SubscribeRequest{PushMessages: []*envelopev1.PushMessage{
			{Topic: "public_topic", Data: []byte("aa"), PushType: envelopev1.PushType_PUSH_TYPE_PUBLIC, MessageType: envelopev1.MessageType_MESSAGE_TYPE_PASSTHROUGH},
		}}
		DispatchMessage(acc, msg)
	})
}

func TestDispatchEvent(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	initExchange()
	appConf := getAppConf()
	appConf.AsyncUserEnable = false

	topic := "t1"
	topics := []string{topic}
	acc := newAcceptor(nil, "id", "appid", topics, &acceptorOptions{ShardIndex: 1, ShardTotal: 2})
	_ = gAcceptorMgr.Add(acc)

	uid := int64(1)
	sess := newMockSession(uid)
	sess.client.Subscribe(topics)
	user, _ := GetUserMgr().Bind(uid, sess)

	sess1 := newMockSession(2)
	sess.client.Subscribe(topics)
	user1, _ := GetUserMgr().Bind(2, sess1)

	t.Run("input", func(t *testing.T) {
		_ = DispatchEvent(&SyncInputEvent{UserID: uid, Topic: topic, Data: "test", Acceptors: []Acceptor{acc}})
	})

	t.Run("force", func(t *testing.T) {
		_ = DispatchEvent(NewForceSyncUserEvent(uid, acc.ID()))
	})

	t.Run("sync_one", func(t *testing.T) {
		_ = DispatchEvent(NewSyncOneUserEvent(nil, nil))
		_ = DispatchEvent(NewSyncOneUserEvent(user, nil))
		_ = DispatchEvent(NewSyncOneUserEvent(user, newAction(ActionSessionOnline, uid, sess.ID(), nil)))
		// 分片不一致,不同步
		_ = DispatchEvent(NewSyncOneUserEvent(user1, nil))
	})

	t.Run("sync_all", func(t *testing.T) {
		_ = DispatchEvent(NewSyncAllUserEvent(""))
		_ = DispatchEvent(NewSyncAllUserEventWithUsers("", GetUserMgr().GetAllUsers()))
	})

	t.Run("sync_config", func(t *testing.T) {
		_ = DispatchEvent(NewSyncConfigEvent(acc.ID()))
	})

	gAcceptorMgr.Remove(acc.ID())
	GetUserMgr().Unbind(uid, sess.ID())

	t.Run("ingore", func(t *testing.T) {
		// 没有acceptor
		_ = DispatchEvent(NewSyncOneUserEvent(user, nil))
		// ignore nil user
		globalExchange.buildUser(nil, nil, nil)
	})

	t.Run("stopped error", func(t *testing.T) {
		m := newExchange()
		assert.NotNil(t, m.DispatchEvent(NewSyncOneUserEvent(user, nil)))
	})

	t.Run("test channel overflow", func(t *testing.T) {
		appConf := getAppConf()
		appConf.AsyncUserEnable = true
		appConf.AsyncUserChannelSize = 1
		appConf.AsyncUserBatchSize = 1
		defer func() {
			appConf.AsyncUserEnable = false
			appConf.AsyncUserChannelSize = 1000
		}()
		m := newExchange()
		m.Start()
		for i := 0; i < 5; i++ {
			_ = m.DispatchEvent(NewSyncOneUserEvent(user, nil))
		}
	})
}

func TestOnBatchSyncUser(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	m := newExchange()
	m.onBatchSyncUser(nil)
	m.onBatchSyncUser([]*SyncOneUserEvent{NewSyncOneUserEvent(newUser(1), nil)})
}

func TestOnForceSyncUser(t *testing.T) {
	ex := newExchange()
	ex.onForceSyncUser(NewForceSyncUserEvent(1, "NoAcceptorID"))
}

func TestExchangeLoopEvents(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	uid := int64(1)
	sess := newMockSession(0)
	user, _ := GetUserMgr().Bind(uid, sess)
	defer func() {
		GetUserMgr().Unbind(uid, sess.ID())
	}()

	t.Run("loop one", func(t *testing.T) {
		conf := getAppConf()
		conf.AsyncUserEnable = true
		conf.AsyncUserBatchSize = 1
		defer func() {
			conf.AsyncUserEnable = false
		}()

		m := newExchange()
		m.Start()
		_ = m.DispatchEvent(NewSyncOneUserEvent(user, nil))
		_ = m.DispatchEvent(NewSyncOneUserEvent(user, nil))
		m.Close()
	})

	t.Run("Loop batch", func(t *testing.T) {
		conf := getAppConf()
		conf.AsyncUserEnable = true
		conf.AsyncUserBatchSize = 100
		defer func() {
			conf.AsyncUserEnable = false
		}()

		m := newExchange()
		m.Start()
		for i := 0; i < 10; i++ {
			_ = m.DispatchEvent(NewSyncOneUserEvent(user, nil))
		}
		m.Close()
	})
}

func TestExchange(t *testing.T) {
	m := newExchange()

	t.Run("basic", func(t *testing.T) {
		assert.EqualValues(t, 0, m.LastWriteAcceptorFailTime())
		assert.EqualValues(t, 0, m.LastWriteChannelFailTime())

		assert.EqualValues(t, 0, m.EventChannelRate())
		m.eventCh = make(chan Event, 1)
		assert.EqualValues(t, 0, m.EventChannelRate())
	})

	t.Run("buildActions", func(t *testing.T) {
		sessId := "sess-id"
		acc := newAcceptor(nil, "id", "appid", []string{"t1"}, &acceptorOptions{FocusEvents: 0xff})

		msg := &envelopev1.SubscribeResponse{}
		m.buildActions(msg, acc, nil)
		assert.Equal(t, 0, len(msg.Events))

		actions := []*Action{
			newAction(ActionSessionOnline, 1, sessId, []string{}),
			newAction(ActionSessionSub, 1, sessId, []string{"t1"}),
			newAction(ActionSessionSub, 1, sessId, []string{"t2"}),
		}
		m.buildActions(msg, acc, actions)
		assert.Equal(t, 2, len(msg.Events))
	})

	m.Close()
}

func TestSyncConfig(t *testing.T) {
	conf := &sdkConf{
		Disable: false,
		Options: map[string]string{"mode": "new"},
	}
	acc := newAcceptor(mock.NewGrpcServerStream(), "test_sync_config", "test_sync_config", []string{"t1", "t2"}, nil)
	gConfigMgr.sdkConf.Store(conf)
	ex := newExchange()
	ex.onSyncConfig(NewSyncConfigEvent("not_exits_id"))
	ex.onSyncConfig(NewSyncConfigEvent(acc.AppID()))
	ex.onSyncConfig(NewSyncConfigEvent(""))

	gConfigMgr.sdkConf.Store(&sdkConf{Disable: true})
	ex.doSyncConfig(acc)
	gConfigMgr.sdkConf.Store(&sdkConf{Disable: false, Options: map[string]string{"disable": "true"}})
	ex.doSyncConfig(acc)
	gConfigMgr.sdkConf.Store(&sdkConf{})
	ex.doSyncConfig(acc)
}

func TestDoSyncUsers(t *testing.T) {
	ex := newExchange()
	ex.doSyncUsers(nil, nil)
}

func TestBatchSyncUser(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	appId := "test_batch_sync_user"
	sess := newMockSession(1)
	user, _ := GetUserMgr().Bind(1, sess)
	events := []*SyncOneUserEvent{
		{User: user, Action: newAction(ActionSessionOnline, 1, "aa", []string{"t1"})},
		{User: newUser(2)},
	}

	acc := newAcceptor(mock.NewGrpcServerStream(), "aa", appId, []string{"t1"}, nil)
	_ = gAcceptorMgr.Add(acc)
	ex := newExchange()
	ex.onBatchSyncUser(events)

	gUserMgr.Unbind(1, sess.ID())
	gAcceptorMgr.Remove(appId)
}
