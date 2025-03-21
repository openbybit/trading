package ws

import (
	"time"

	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
)

type EventType uint8

type Event interface {
	Type() EventType
}

type ActionType uint8

const (
	// session action
	ActionUnknown        = ActionType(0)
	ActionSessionOnline  = ActionType(envelopev1.EventType_EVENT_TYPE_SESSION_ONLINE)
	ActionSessionOffline = ActionType(envelopev1.EventType_EVENT_TYPE_SESSION_OFFLINE)
	ActionSessionSub     = ActionType(envelopev1.EventType_EVENT_TYPE_SESSION_SUBSCRIBE)
	ActionSessionUnsub   = ActionType(envelopev1.EventType_EVENT_TYPE_SESSION_UNSUBSCRIBE)
)

func (at ActionType) IsAny(targets ...ActionType) bool {
	for _, t := range targets {
		if at == t {
			return true
		}
	}

	return false
}

func isFocusEvents(focusEvents uint64, actions uint64) bool {
	return (focusEvents & actions) != 0
}

type Action struct {
	ActionID  string
	Type      ActionType
	Timestamp int64
	UserID    int64
	SessionID string
	Topics    []string
}

func newAction(typ ActionType, userId int64, sessionId string, topics []string) *Action {
	return &Action{
		ActionID:  newUUID(),
		Timestamp: time.Now().UnixNano(),
		Type:      typ,
		UserID:    userId,
		SessionID: sessionId,
		Topics:    topics,
	}
}

const (
	EventTypeUnknown = EventType(iota)
	EventTypeSyncOneUser
	EventTypeSyncManyUser
	EventTypeSyncAllUser
	EventTypeForceSyncUser
	EventTypeSyncConfig
	EventTypeSyncInput // 客户端上行数据同步
)

// ForceSyncUserEvent 强制同步一次用户数据给对应的SDK,如果找不到用户则同步空值
type ForceSyncUserEvent struct {
	uid        int64
	acceptorID string
}

func (ev *ForceSyncUserEvent) Type() EventType {
	return EventTypeForceSyncUser
}

func NewForceSyncUserEvent(uid int64, acceptorID string) *ForceSyncUserEvent {
	return &ForceSyncUserEvent{
		uid:        uid,
		acceptorID: acceptorID,
	}
}

// SyncOneUserEvent 同步单个用户数据,同时会附带一些action event
type SyncOneUserEvent struct {
	User   User
	Action *Action
}

func (ev *SyncOneUserEvent) Type() EventType {
	return EventTypeSyncOneUser
}

func NewSyncOneUserEvent(u User, action *Action) *SyncOneUserEvent {
	return &SyncOneUserEvent{User: u, Action: action}
}

type SyncAllUserEvent struct {
	acceptorID string
	users      []User
}

func (ev *SyncAllUserEvent) Type() EventType {
	return EventTypeSyncAllUser
}

func NewSyncAllUserEvent(acceptorID string) *SyncAllUserEvent {
	return &SyncAllUserEvent{acceptorID: acceptorID}
}

func NewSyncAllUserEventWithUsers(acceptorID string, users []User) *SyncAllUserEvent {
	return &SyncAllUserEvent{acceptorID: acceptorID, users: users}
}

type SyncConfigEvent struct {
	acceptorID string
}

func (ev *SyncConfigEvent) Type() EventType {
	return EventTypeSyncConfig
}

func NewSyncConfigEvent(id string) *SyncConfigEvent {
	return &SyncConfigEvent{acceptorID: id}
}

type SyncInputEvent struct {
	ReqID     string
	UserID    int64
	SessID    string
	Topic     string
	Data      string
	Acceptors []Acceptor
}

func (ev *SyncInputEvent) Type() EventType {
	return EventTypeSyncInput
}
