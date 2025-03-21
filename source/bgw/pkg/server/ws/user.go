package ws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/deadlock"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
)

var (
	errUserDeleted = errors.New("user deleted") // 用户已经被剔除
)

// Params merge all sessions param
type Params map[string][]string

// User user is session manager
type User interface {
	GetMemberID() int64
	GetCreateTime() int64
	Size() int
	Empty() bool
	IsDeleted() bool
	GetTopics() []string
	GetParams() Params
	GetSessions() []Session
	FilterSessions(all bool, topic string, sessionID string) []Session
	Add(s Session) error
	Remove(id string) bool
	// Build 构建消息
	Build() (msg *envelopev1.User, allTopics []string)
	ToMessageUser() *envelopev1.User
	CanForceSync() bool
}

type authedUser struct {
	deadlock.RWMutex

	uniqueID   string       // 每次创建用户会生成一个唯一ID,仅用于debug
	uid        int64        //
	createTime int64        // 创建时间,纳秒,初始化后不会改变
	sessions   atomic.Value // 每次session增减会拷贝新数组,避免加锁读取
	// params 聚合所有session参数
	joinParams atomic.Value // map[string]string 以逗号聚合所有values
	params     atomic.Value // map[string][]string
	// cache topics for acceptor
	version uint64   // cache构建版本,起始值为当前纳秒时间戳,保证新建用户一定大于历史版本号(纳秒级)
	topics  []string // cache数据
	// 标记user被删除,当用户连接数从非零变到零时,需要从user_mgr中删除
	// 为了避免删除过程瞬间再有新的连接加入产生并发问题,用户此次加入会失败
	// 只会被设置一次
	deleted       int32
	lastForceTime int64 // 上次强制同步时间
}

func newUser(uid int64) *authedUser {
	nowNano := time.Now().UnixNano()
	return &authedUser{
		uniqueID:   newUUID(),
		uid:        uid,
		version:    uint64(nowNano),
		createTime: nowNano,
		deleted:    0,
	}
}

// GetMemberID return the member id of the session
func (u *authedUser) GetMemberID() int64 {
	return u.uid
}

func (u *authedUser) GetCreateTime() int64 {
	return u.createTime
}

func (u *authedUser) Size() int {
	return len(u.GetSessions())
}

func (u *authedUser) Empty() bool {
	return len(u.GetSessions()) == 0
}

// IsDeleted 标识用户是否被删除
func (u *authedUser) IsDeleted() bool {
	return atomic.LoadInt32(&u.deleted) == 1
}

func (u *authedUser) GetTopics() []string {
	u.RLock()
	res := u.topics
	u.RUnlock()
	return res
}

// CanForceSync 强制同步用户数据,一段时间内最多一次,避免业务使用不当或状态异常导致频繁同步状态
func (u *authedUser) CanForceSync() bool {
	forceTime := atomic.LoadInt64(&u.lastForceTime)
	now := nowUnixNano()
	if now-forceTime > int64(time.Minute*5) {
		atomic.StoreInt64(&u.lastForceTime, now)
		return true
	}

	return false
}

func (u *authedUser) ToMessageUser() *envelopev1.User {
	u.RLock()
	defer u.RUnlock()
	sessions := u.GetSessions()
	return u.toMessageUser(sessions)
}

func (u *authedUser) toMessageUser(sessions []Session) *envelopev1.User {
	return &envelopev1.User{
		MemberId:      u.uid,
		Topics:        u.topics,
		Version:       u.version,
		CreateTimeE9:  u.createTime,
		SessionSize:   uint32(len(sessions)),
		SessionParams: u.getJoinParams(),
	}
}

func (u *authedUser) Build() (*envelopev1.User, []string) {
	u.Lock()
	defer u.Unlock()
	u.version++

	sessions := u.GetSessions()
	// build topics
	topics := newTopics()
	for _, s := range sessions {
		topics.Merge(s.GetClient().GetTopics())
	}

	oldTopics := u.topics
	u.topics = getConfigMgr().IgnorePublicTopics(topics.Values())
	sort.Strings(u.topics)

	// build all topics
	allTopicMap := newTopics()
	allTopicMap.Add(oldTopics...)
	allTopicMap.Add(u.topics...)
	allTopics := allTopicMap.Values()

	res := u.toMessageUser(sessions)

	return res, allTopics
}

func (u *authedUser) getJoinParams() map[string]string {
	v, ok := u.joinParams.Load().(map[string]string)
	if ok {
		return v
	}

	return nil
}

func (u *authedUser) GetParams() Params {
	v, ok := u.params.Load().(Params)
	if ok {
		return v
	}

	return nil
}

func (u *authedUser) getSessionByID(sessId string) Session {
	sessions := u.GetSessions()
	for _, s := range sessions {
		if s.ID() == sessId {
			return s
		}
	}

	return nil
}

func (u *authedUser) Add(s Session) (err error) {
	u.Lock()
	defer u.Unlock()

	if u.IsDeleted() {
		WSErrorInc("user", "deleted")
		return errUserDeleted
	}

	exists := u.getSessionByID(s.ID()) != nil
	if exists {
		WSErrorInc("user", "duplicate_add_session")
		glog.Error(context.Background(), fmt.Sprintf("user duplicate add session, uid=%v, sessId=%v", u.uid, s.ID()))
		return nil
	}

	sconf := getDynamicConf()
	sessions := u.GetSessions()
	if len(sessions) >= sconf.MaxSessionsPerUser {
		return newCodeErrFrom(errTooManySession, "exceed user max size, uid=%v", u.uid)
	}

	_ = s.SetMember(u.uid)
	sessions = append(sessions, s)
	u.sessions.Store(sessions)
	u.rebuildParams()

	return
}

// Remove 通过id删除session，并重新构建参数,返回session是否为空
func (u *authedUser) Remove(id string) bool {
	u.Lock()
	defer u.Unlock()
	oldSessions := u.GetSessions()
	newSessions := make([]Session, 0, len(oldSessions))
	var sess Session
	for _, s := range oldSessions {
		if s.ID() == id {
			sess = s
		} else {
			newSessions = append(newSessions, s)
		}
	}

	if sess != nil {
		u.sessions.Store(newSessions)
		u.rebuildParams()
	}
	empty := len(newSessions) == 0

	if empty {
		atomic.StoreInt32(&u.deleted, 1)
	}

	return empty
}

func (u *authedUser) rebuildParams() {
	sessions := u.GetSessions()
	params := make(Params)
	for _, s := range sessions {
		sp := s.GetClient().GetParams()
		for k, v := range sp {
			params[k] = append(params[k], v)
		}
	}

	for k, v := range params {
		params[k] = distinctString(v)
	}

	joins := make(map[string]string)
	for k, v := range params {
		joins[k] = strings.Join(v, ",")
	}

	u.params.Store(params)
	u.joinParams.Store(joins)
	// glog.Debug(context.Background(), fmt.Sprintf("rebuild params,session_size: %v, %v, %v", len(sessions), joins, params))
}

func (u *authedUser) GetSessions() []Session {
	sess, ok := u.sessions.Load().([]Session)
	if ok {
		return sess
	}

	return nil
}

func (u *authedUser) FilterSessions(all bool, topic string, sessionID string) []Session {
	sessions := u.GetSessions()
	switch {
	case sessionID != "":
		for _, s := range sessions {
			if s.ID() == sessionID {
				return []Session{s}
			}
		}
		return nil
	case all:
		return sessions
	default:
		res := make([]Session, 0, len(sessions))
		for _, s := range sessions {
			if s.GetClient().HasTopic(topic) {
				res = append(res, s)
			}
		}
		return res
	}
}
