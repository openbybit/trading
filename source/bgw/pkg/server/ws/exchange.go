package ws

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"github.com/rs/xid"
)

type pushFlag int32

func (f pushFlag) IsPassthrough() bool {
	return f&flagTopicPassthrough != 0
}

const (
	// FlagReserved topic passthrough flag
	// FlagReserved = iota
	// flagTopicPassthrough topic passthrough flag
	flagTopicPassthrough = 0x01
)

var globalOnce sync.Once
var globalExchange = newExchange()

func initExchange() {
	globalOnce.Do(func() {
		globalExchange.Start()
	})
}

func DispatchMessage(acceptor Acceptor, msg *envelopev1.SubscribeRequest) {
	globalExchange.OnMessage(context.Background(), acceptor, msg)
}

func DispatchEvent(ev Event) error {
	return globalExchange.DispatchEvent(ev)
}

// Exchange 用于将用户状态数据同步至SDK,同时接收SDK的推送数据并发送给用户
type Exchange interface {
	OnMessage(ctx context.Context, acceptor Acceptor, msg *envelopev1.SubscribeRequest)
	DispatchEvent(ev Event) error
	Start()
	Close()
}

func newExchange() *exchange {
	ex := &exchange{}
	return ex
}

// exchange 单线程异步同步用户状态数据
type exchange struct {
	running                   atomic.Bool
	eventCh                   chan Event   // 异步同步channel
	lastWriteChannelFailTime  atomic.Int64 // 最后一次写channel失败时间
	lastWriteAcceptorFailTime atomic.Int64 // 最后一次写Acceptor失败时间
}

func (ex *exchange) IsRunning() bool {
	return ex.running.Load()
}

// EventChannelRate channel使用率
func (ex *exchange) EventChannelRate() float64 {
	if ex.eventCh != nil {
		return 100.0 * float64(len(ex.eventCh)) / float64(cap(ex.eventCh))
	}

	return 0
}

// LastWriteChannelFailTime 最后一次写入失败时间
func (ex *exchange) LastWriteChannelFailTime() int64 {
	return ex.lastWriteChannelFailTime.Load()
}

func (ex *exchange) LastWriteAcceptorFailTime() int64 {
	return ex.lastWriteAcceptorFailTime.Load()
}

func (ex *exchange) Start() {
	appConf := getAppConf()

	if appConf.AsyncUserEnable && ex.eventCh == nil {
		ex.eventCh = make(chan Event, appConf.AsyncUserChannelSize)
		go ex.loopEvent()
	}

	ex.running.Store(true)
}

func (ex *exchange) Close() {
	ex.running.Store(false)
	if ex.eventCh != nil {
		// graceful shutdown
		const maxWaitRound = 100
		for i := 0; i < maxWaitRound; i++ {
			if len(ex.eventCh) == 0 {
				glog.Info(context.Background(), "exchange graceful shutdown")
				break
			}
			time.Sleep(time.Millisecond * 10)
		}
		close(ex.eventCh)
		ex.eventCh = nil
	}
}

func (ex *exchange) OnMessage(ctx context.Context, acceptor Acceptor, msg *envelopev1.SubscribeRequest) {
	messages := msg.PushMessages
	if len(messages) == 0 {
		// 兼容历史协议
		if len(msg.Topics) == 0 || msg.Header == nil {
			WSErrorInc("exchange", "invalid_msg")
			glog.Info(context.Background(), "invalid push message", glog.Any("msg", msg))
			return
		}
		hd := msg.Header
		pmsg := &envelopev1.PushMessage{
			UserId:        msg.MemberId,
			Topic:         msg.Topics[0],
			Data:          msg.Data,
			Flags:         msg.Flag,
			SessionId:     msg.SessionId,
			MessageId:     hd.MsgId,
			TraceId:       hd.TraceId,
			RequestTimeE9: hd.RequestTimeE9,
			InitTimeE9:    hd.InitTimeE9,
			SdkTimeE9:     hd.SdkTimeE9,
		}

		messages = append(messages, pmsg)
	}

	startTime := nowUnixNano()
	appId := acceptor.AppID()
	remoteAddr := acceptor.Address()

	for _, m := range messages {
		if len(m.Data) == 0 {
			WSCounterInc("exchange", "empty_data")
			glog.Info(context.Background(), "ws: push message is empty", glog.String("app_id", acceptor.AppID()), glog.String("topic", m.Topic), glog.Int64("uid", m.UserId))
			continue
		}

		if m.PushType == envelopev1.PushType_PUSH_TYPE_PUBLIC {
			_ = gPublicMgr.Write(&publicMessage{PushMessage: m, appId: appId, remoteAddr: remoteAddr})
			continue
		}

		if m.UserId <= 0 {
			WSCounterInc("exchange", "invalid_uid")
			continue
		}

		user := GetUserMgr().GetUser(m.UserId)
		if user == nil {
			WSCounterInc("exchange", "user_offline")
			DispatchEvent(NewForceSyncUserEvent(m.UserId, acceptor.ID()))
			glog.Info(context.Background(), "ws: user offline", glog.String("app_id", acceptor.AppID()), glog.String("topic", m.Topic), glog.Int64("uid", m.UserId))
			continue
		}

		mt := gMetricsMgr.Get()
		mt.Type = metricsTypePush
		mt.AppID = appId
		mt.RemoteAddr = remoteAddr
		mt.WsStartTimeE9 = nowUnixNano()
		mt.Push = m

		flags := pushFlag(m.Flags)
		passthrough := flags.IsPassthrough()
		sessions := user.FilterSessions(passthrough, m.Topic, m.SessionId)
		if len(sessions) == 0 {
			WSCounterInc("exchange", "not_found_sessions")
			DispatchEvent(NewForceSyncUserEvent(m.UserId, acceptor.ID()))
		}

		for _, s := range sessions {
			err := s.Write(&Message{
				Type: MsgTypePush,
				Data: m.Data,
			})

			if err == nil {
				mt.EndTimeE9 = append(mt.EndTimeE9, nowUnixNano())
				mt.Sessions = append(mt.Sessions, s.ID())
			} else {
				mt.ErrCount++
				glog.Info(context.Background(), "write fail",
					glog.Int64("uid", m.UserId),
					glog.String("sid", s.ID()),
					glog.String("topic", m.Topic),
					glog.String("app_id", appId),
					glog.String("address", remoteAddr),
					glog.String("msg_id", m.MessageId),
					glog.String("trace_id", m.TraceId),
					glog.Int("error", toCodeErr(err).Code()),
				)
			}
		}

		gMetricsMgr.Send(mt)
	}

	wsDefaultLatencyE6(time.Duration(nowUnixNano()-startTime), "exchange", "on_message")
}

func (ex *exchange) DispatchEvent(ev Event) error {
	if !ex.IsRunning() {
		return errServerStopped
	}

	if getAppConf().AsyncUserEnable && ev.Type() == EventTypeSyncOneUser {
		select {
		case ex.eventCh <- ev:
		default:
			WSErrorInc("exchange", "discard_event")
			ex.lastWriteChannelFailTime.Store(nowUnixNano())
		}
	} else {
		ex.onEvent(ev)
	}

	return nil
}

func (ex *exchange) loopEvent() {
	batchSize := getAppConf().AsyncUserBatchSize
	glog.Infof(context.Background(), "exchange start loopEvent, batch size: %d", batchSize)
	if batchSize > 1 {
		userEvents := make([]*SyncOneUserEvent, 0, batchSize) // pool for reused
		var hasData bool

		for {
			ev, ok := <-ex.eventCh
			if !ok {
				break
			}
			ex.onEvent(ev)

			// 批次发送剩余数据
			size := int(math.Ceil(float64(len(ex.eventCh)) / float64(batchSize)))
			for i := 0; i < size; i++ {
				userEvents, hasData = ex.readUserEventsFromChannel(userEvents, batchSize)
				ex.onBatchSyncUser(userEvents)
				if !hasData {
					break
				}
			}
		}
	} else {
		for {
			ev, ok := <-ex.eventCh
			if !ok {
				break
			}
			ex.onEvent(ev)
		}
	}

	glog.Info(context.Background(), "exchange loop stopped")
}

func (ex *exchange) readUserEventsFromChannel(userEvents []*SyncOneUserEvent, batchSize int) ([]*SyncOneUserEvent, bool) {
	userEvents = userEvents[:0]
	for i := 0; i < batchSize; i++ {
		select {
		case ev := <-ex.eventCh:
			if ev.Type() == EventTypeSyncOneUser {
				userEvents = append(userEvents, ev.(*SyncOneUserEvent))
			} else {
				ex.onEvent(ev)
			}
		default:
			return userEvents, false
		}
	}

	return userEvents, true
}

func (ex *exchange) onEvent(ev Event) {
	defer func() {
		if e := recover(); e != nil {
			dumpPanic("exchange", e)
		}
	}()

	switch ev.Type() {
	case EventTypeSyncInput:
		ex.onSyncInput(ev.(*SyncInputEvent))
	case EventTypeForceSyncUser:
		ex.onForceSyncUser(ev.(*ForceSyncUserEvent))
	case EventTypeSyncOneUser:
		ex.onSyncOneUser(ev.(*SyncOneUserEvent))
	case EventTypeSyncAllUser:
		ex.onSyncAllUser(ev.(*SyncAllUserEvent))
	case EventTypeSyncConfig:
		ex.onSyncConfig(ev.(*SyncConfigEvent))
	}
}

func (ex *exchange) onSyncInput(ev *SyncInputEvent) {
	if ev.ReqID == "" {
		ev.ReqID = newUUID()
	}

	nowNano := nowUnixNano()
	event := &envelopev1.Event{
		Type:      envelopev1.EventType_EVENT_TYPE_SESSION_INPUT,
		EventId:   ev.ReqID,
		Timestamp: nowNano,
		UserId:    ev.UserID,
		SessionId: ev.SessID,
		Topics:    []string{ev.Topic},
		Data:      ev.Data,
	}

	msg := &envelopev1.SubscribeResponse{
		Cmd:    envelopev1.Command_COMMAND_SYNC,
		Header: &envelopev1.Header{TraceId: newUUID()},
		Events: []*envelopev1.Event{event},
	}

	sconf := getDynamicConf()
	for _, acc := range ev.Acceptors {
		if err := acc.Send(msg); err != nil {
			WSErrorInc("discard_input", ev.Topic)
			glog.Error(context.Background(),
				"discard_input",
				glog.String("req_id", ev.ReqID),
				glog.String("sess_id", ev.SessID),
				glog.Int64("uid", ev.UserID),
				glog.String("topic", ev.Topic),
				glog.Int("data_size", len(ev.Data)),
			)
		} else {
			wsSyncInc("sync_input", acc.AppID())
			if sconf.EnableInputLog {
				glog.Info(context.Background(),
					"sync_input",
					glog.String("req_id", ev.ReqID),
					glog.String("sess_id", ev.SessID),
					glog.Int64("uid", ev.UserID),
					glog.String("topic", ev.Topic),
					glog.Int("data_size", len(ev.Data)),
				)
			}
		}
	}
}

// onForceSyncUser 强制同步一次sdk状态给sdk, 不需要校验shardInfo, topic等信息
func (ex *exchange) onForceSyncUser(ev *ForceSyncUserEvent) {
	acceptor := GetAcceptorMgr().Get(ev.acceptorID)
	if isNil(acceptor) {
		glog.Info(context.Background(), "onForceSyncUser cannot find acceptor", glog.String("id", ev.acceptorID))
		return
	}

	glog.Debug(context.Background(), "onForceSyncUser", glog.Int64("uid", ev.uid), glog.String("acceptor", ev.acceptorID))

	nowNano := nowUnixNano()
	msg := &envelopev1.SubscribeResponse{
		Cmd: envelopev1.Command_COMMAND_SYNC,
		Header: &envelopev1.Header{
			Type:          envelopev1.PushType_PUSH_TYPE_PRIVATE,
			RequestTimeE9: nowNano,
			TraceId:       newUUID(),
			NodeId:        globalNodeID,
		},
	}

	user := GetUserMgr().GetUser(ev.uid)
	if !isNil(user) {
		if user.CanForceSync() {
			wsSyncInc("force_sync_exist_user", acceptor.AppID())
			muser := user.ToMessageUser()
			ex.buildUser(msg, acceptor, muser)
			_ = acceptor.Send(msg)
		} else {
			wsSyncInc("force_sync_ignore_user", acceptor.AppID())
		}
	} else {
		wsSyncInc("force_sync_miss_user", acceptor.AppID())
		msgUser := &envelopev1.User{
			MemberId:      ev.uid,
			Version:       uint64(nowNano),
			SessionSize:   0,
			SessionParams: nil,
		}
		msg.Users = append(msg.Users, msgUser)
		_ = acceptor.Send(msg)
	}
}

func (ex *exchange) onSyncOneUser(ev *SyncOneUserEvent) {
	user := ev.User
	if isNil(user) {
		return
	}

	WSCounterInc("exchange", "sync_one")
	muser, allTopics := user.Build()

	acceptors := GetAcceptorMgr().GetAll()
	if len(acceptors) == 0 {
		glog.Debug(context.Background(), "ignore sync user info because no acceptor", glog.Int64("uid", user.GetMemberID()))
		return
	}

	reqTimeE9 := time.Now().UnixNano()
	traceId := newUUID()

	conf := getDynamicConf()

	for _, acc := range acceptors {
		if !checkUserShard(acc, user.GetMemberID()) {
			continue
		}

		msg := &envelopev1.SubscribeResponse{
			Cmd: envelopev1.Command_COMMAND_SYNC,
			Header: &envelopev1.Header{
				Type:          envelopev1.PushType_PUSH_TYPE_PRIVATE,
				RequestTimeE9: reqTimeE9,
				TraceId:       traceId,
				NodeId:        globalNodeID,
			},
		}

		if ev.Action != nil {
			ex.buildActions(msg, acc, []*Action{ev.Action})
		}

		if containsAnyInOrderedList(acc.Topics(), allTopics) {
			ex.buildUser(msg, acc, muser)
		}

		if len(msg.Events) > 0 || len(msg.Users) > 0 {
			ex.sendMsgToAcceptor(acc, msg)

			wsSyncInc("sync_one", acc.AppID())
			if conf.Log.CanWriteSyncLog(acc.AppID()) {
				glog.Info(context.Background(), "sync_one_user", glog.String("acceptor_id", acc.ID()), glog.String("addr", acc.Address()), glog.Any("users", msg.Users), glog.Any("events", msg.Events))
			}
		}
	}
}

// onBatchSyncUser 批量同步用户数据到sdk
func (ex *exchange) onBatchSyncUser(events []*SyncOneUserEvent) {
	defer func() {
		if e := recover(); e != nil {
			dumpPanic("exchange", e)
		}
	}()

	if len(events) == 0 {
		return
	}

	nowNano := nowUnixNano()

	// 聚合后用户数据
	type batchUser struct {
		uid       int64
		actions   []*Action        // 聚合后action
		muser     *envelopev1.User // 构建后用户数据
		topics    []string         // 所有topic
		forceSync bool             // 用户不存在, 需要强制同步给所有acceptors
	}

	// 聚合用户数据,取最后一个user
	users := make([]*batchUser, 0, len(events))
	usersMap := make(map[int64]*batchUser, len(events))
	for _, ev := range events {
		user, ok := usersMap[ev.User.GetMemberID()]
		if !ok {
			user = &batchUser{uid: ev.User.GetMemberID()}
			users = append(users, user)
			usersMap[ev.User.GetMemberID()] = user
		}
		if ev.Action != nil {
			user.actions = append(user.actions, ev.Action)
		}
	}

	for _, u := range users {
		user := GetUserMgr().GetUser(u.uid)
		if user != nil {
			muser, allTopics := user.Build()
			u.forceSync = false
			u.muser = muser
			u.topics = allTopics
		} else {
			u.forceSync = true
			u.muser = &envelopev1.User{
				MemberId:      u.uid,
				Version:       uint64(nowNano),
				SessionSize:   0,
				SessionParams: nil,
			}
		}
	}

	acceptors := GetAcceptorMgr().GetAll()
	if len(acceptors) == 0 {
		glog.Debug(context.Background(), "ignore sync user info because no acceptor")
		return
	}

	WSCounterInc("exchange", "batch_sync_user")

	reqTimeE9 := time.Now().UnixNano()
	traceId := newUUID()

	for _, acc := range acceptors {
		msg := &envelopev1.SubscribeResponse{
			Cmd: envelopev1.Command_COMMAND_SYNC,
			Header: &envelopev1.Header{
				Type:          envelopev1.PushType_PUSH_TYPE_PRIVATE,
				RequestTimeE9: reqTimeE9,
				TraceId:       traceId,
				NodeId:        globalNodeID,
			},
		}

		for _, u := range users {
			if !checkUserShard(acc, u.uid) {
				continue
			}
			ex.buildActions(msg, acc, u.actions)

			if u.forceSync || containsAnyInOrderedList(acc.Topics(), u.topics) {
				ex.buildUser(msg, acc, u.muser)
			}
		}

		if len(msg.Events) > 0 || len(msg.Users) > 0 {
			ex.sendMsgToAcceptor(acc, msg)
			wsSyncInc("sync_batch", acc.AppID())
		}
	}
}

func (ex *exchange) buildActions(msg *envelopev1.SubscribeResponse, acc Acceptor, actions []*Action) {
	focusEvents := acc.FocusEvents()
	if len(actions) == 0 || focusEvents == 0 {
		return
	}

	for _, action := range actions {
		isFocusEvent := focusEvents&uint64(action.Type) != 0
		if !isFocusEvent {
			continue
		}

		// 订阅和取消订阅事件需要过滤不关注的topic
		topics := ex.filterTopics(acc.Topics(), action.Topics)
		if action.Type.IsAny(ActionSessionSub, ActionSessionUnsub) && len(topics) == 0 {
			continue
		}

		mevent := &envelopev1.Event{
			Type:      envelopev1.EventType(action.Type),
			EventId:   action.ActionID,
			Timestamp: action.Timestamp,
			UserId:    action.UserID,
			SessionId: action.SessionID,
			Topics:    topics,
		}

		msg.Events = append(msg.Events, mevent)
	}
}

func (ex *exchange) buildUser(msg *envelopev1.SubscribeResponse, acc Acceptor, muser *envelopev1.User) {
	if muser == nil {
		return
	}

	x := &envelopev1.User{
		MemberId:      muser.MemberId,
		Version:       muser.Version,
		SessionSize:   muser.SessionSize,
		SessionParams: muser.SessionParams,
		CreateTimeE9:  muser.CreateTimeE9,
		Topics:        ex.filterTopics(acc.Topics(), muser.Topics),
	}
	msg.Users = append(msg.Users, x)
}

func (ex *exchange) onSyncAllUser(ev *SyncAllUserEvent) {
	acceptor := GetAcceptorMgr().Get(ev.acceptorID)
	if isNil(acceptor) {
		glog.Error(context.TODO(), "onSyncAllUser fail, invalid accpetor")
		return
	}

	users := ev.users
	if len(users) == 0 {
		users = GetUserMgr().GetAllUsers()
	}

	if len(users) == 0 {
		return
	}

	wsSyncInc("sync_all_users", acceptor.AppID())

	ex.doSyncUsers(acceptor, users)
	glog.Infof(
		context.TODO(),
		"onSyncAllUser id=%v appId=%v, size=%v",
		acceptor.ID(), acceptor.AppID(), len(users),
	)
}

func (ex *exchange) doSyncUsers(acceptor Acceptor, users []User) {
	if len(users) == 0 {
		return
	}

	msg := &envelopev1.SubscribeResponse{
		Cmd: envelopev1.Command_COMMAND_SYNC,
		Header: &envelopev1.Header{
			Type:          envelopev1.PushType_PUSH_TYPE_PRIVATE,
			RequestTimeE9: time.Now().UnixNano(),
			TraceId:       newUUID(),
			NodeId:        globalNodeID,
		},
	}

	for _, u := range users {
		if !checkUserShard(acceptor, u.GetMemberID()) {
			continue
		}

		info := u.ToMessageUser()
		if containsAnyInOrderedList(acceptor.Topics(), info.Topics) {
			info.Topics = ex.filterTopics(acceptor.Topics(), info.Topics)
			msg.Users = append(msg.Users, info)
		}
	}

	ex.sendAndSplitMsg(acceptor, msg)
}

// sendAndSplitMsg 消息包数据量如果过大,分割后再发送
func (ex *exchange) sendAndSplitMsg(acc Acceptor, msg *envelopev1.SubscribeResponse) int {
	size := len(msg.Users) + len(msg.Events)
	if size == 0 {
		return 0
	}

	maxSyncMsgSize := getDynamicConf().MaxSyncMsgSize

	if maxSyncMsgSize == 0 || size <= maxSyncMsgSize {
		ex.sendMsgToAcceptor(acc, msg)
		return 1
	}

	WSCounterInc("exchange", "split_msg")

	result := 0
	// split msg
	step := maxSyncMsgSize
	userSize := len(msg.Users)
	for startIdx := 0; startIdx < userSize; startIdx += step {
		endIdx := startIdx + step
		if endIdx > userSize {
			endIdx = userSize
		}

		m := &envelopev1.SubscribeResponse{
			Cmd:    msg.Cmd,
			Header: msg.Header,
		}
		for i := startIdx; i < endIdx; i++ {
			m.Users = append(m.Users, msg.Users[i])
		}
		ex.sendMsgToAcceptor(acc, m)
		result++
	}

	eventSize := len(msg.Events)
	for startIdx := 0; startIdx < eventSize; startIdx += step {
		endIdx := startIdx + step
		if endIdx > eventSize {
			endIdx = eventSize
		}
		m := &envelopev1.SubscribeResponse{
			Cmd:    msg.Cmd,
			Header: msg.Header,
		}

		for i := startIdx; i < endIdx; i++ {
			m.Events = append(m.Events, msg.Events[i])
		}

		ex.sendMsgToAcceptor(acc, m)
		result++
	}

	return result
}

// sendMsgToAcceptor 发送通道满后,消息会丢弃,需要延迟同步全量用户数据
func (ex *exchange) sendMsgToAcceptor(acc Acceptor, msg *envelopev1.SubscribeResponse) {
	if err := acc.Send(msg); err != nil {
		if errors.Is(err, errAcceptorSendDiscard) {
			ex.lastWriteAcceptorFailTime.Store(nowUnixNano())
		}
	}
}

func (ex *exchange) onSyncConfig(ev *SyncConfigEvent) {
	WSCounterInc("exchange", "sync_config")

	if ev.acceptorID != "" {
		acc := gAcceptorMgr.Get(ev.acceptorID)
		if acc == nil {
			return
		}
		ex.doSyncConfig(acc)
	} else {
		acceptors := gAcceptorMgr.GetAll()
		for _, acc := range acceptors {
			ex.doSyncConfig(acc)
		}
		glog.Debug(context.Background(), "onSyncConfig", glog.Int64("count", int64(len(acceptors))))
	}
}

func (ex *exchange) doSyncConfig(acc Acceptor) {
	cfg := gConfigMgr.GetSdkConf()
	if cfg == nil {
		glog.Info(context.Background(), "sdk config is nil, ignore to sync")
		return
	}

	if cfg.Disable {
		glog.Info(context.Background(), "SyncConfig disable", glog.String("acceptor", acc.AppID()))
		return
	}

	options := cfg.GetOptionsByAppID(acc.AppID())
	disable := options["disable"]
	if disable == "true" {
		glog.Info(context.Background(), "SyncConfig disable", glog.String("acceptor", acc.AppID()), glog.Any("options", options))
		return
	}

	// 没有任何配置,忽略同步
	if len(options) == 0 {
		return
	}

	msg := &envelopev1.SubscribeResponse{
		Cmd: envelopev1.Command_COMMAND_SYNC_CONFIG,
		Header: &envelopev1.Header{
			RequestId: xid.New().String(),
			NodeId:    globalNodeID,
		},
		Config: &envelopev1.Config{
			Options: options,
		},
	}

	err := acc.Send(msg)
	glog.Info(context.Background(),
		"sync_config",
		glog.String("app_id", acc.AppID()),
		glog.String("connector_id", acc.ID()),
		glog.Any("config", msg.Config),
		glog.NamedError("err", err),
	)
}

func (ex *exchange) filterTopics(orderedFocusTopics []string, topics []string) []string {
	if len(topics) == 0 {
		return topics
	}

	res := make([]string, 0, len(topics))
	for _, t := range topics {
		if containsInOrderedList(orderedFocusTopics, t) {
			res = append(res, t)
		}
	}

	return res
}

// checkUserShard 判断用户是否需要shard
func checkUserShard(a Acceptor, uid int64) bool {
	userShardIndex := a.UserShardIndex()
	userShardTotal := a.UserShardTotal()
	if userShardIndex == -1 || userShardTotal <= 0 {
		return true
	}

	// 判断是否匹配灰度节点
	if userShardIndex == userShardTotal {
		return getDynamicConf().IsGray(uid)
	}

	// 判断是否匹配非灰度节点
	index := uint64(uid) % uint64(userShardTotal)
	return int(index) == userShardIndex
}
