package ws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/deadlock"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var gSessionMgr = newSessionMgr()

func GetSessionMgr() SessionMgr {
	return gSessionMgr
}

func newSessionMgr() *sessionMgr {
	m := &sessionMgr{
		sessions: make(map[string]Session),
		ips:      make(map[string]int),
	}

	return m
}

type SessionMgr interface {
	Size() int
	GetSessions() []Session
	GetSession(id string) Session
	AddSession(s Session) error
	DelSession(id string)
	Close()
}

type sessionMgr struct {
	mux      deadlock.RWMutex
	sessions map[string]Session
	ips      map[string]int
}

func (m *sessionMgr) Size() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.sessions)
}

func (m *sessionMgr) Close() {
	glog.Info(context.Background(), "start to close session_mgr")

	m.mux.Lock()
	var sessions []Session
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]Session)
	m.mux.Unlock()

	for _, s := range sessions {
		s.Stop()
	}

	glog.Info(context.Background(), "session_mgr close finished")
}

func (m *sessionMgr) GetSessions() []Session {
	m.mux.RLock()
	defer m.mux.RUnlock()

	sessions := make([]Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}

	return sessions
}

func (m *sessionMgr) GetSession(id string) Session {
	m.mux.RLock()
	defer m.mux.RUnlock()
	res, ok := m.sessions[id]
	if ok {
		return res
	}

	return nil
}

func (m *sessionMgr) AddSession(s Session) error {
	if isNil(s) {
		return fmt.Errorf("AddSession fail, nil session")
	}
	m.mux.Lock()
	err := m.doAddSession(s)
	m.mux.Unlock()

	return err
}

func (m *sessionMgr) doAddSession(s Session) error {
	if _, ok := m.sessions[s.ID()]; ok {
		return fmt.Errorf("%w, duplicate uid: %v", errParamsErr, s.ID())
	}

	sconf := getDynamicConf()

	if len(m.sessions) >= sconf.MaxSessions {
		WSCounterInc("session_manager", "max_limit")
		glog.Info(context.Background(), "session exceed max size limit", glog.Int64("size", int64(len(m.sessions))))
		return errTooManySession
	}

	ip := s.GetClient().GetIP()
	if _, ok := sconf.IpBlackList[ip]; ok {
		WSCounterInc("session_manager", "blacklist_limit")
		glog.Info(context.Background(), "session ip in blacklist", glog.String("ip", ip))
		return errIPInBlacklist
	}

	if m.ips[ip] >= sconf.MaxSessionsPerIp {
		if _, ok := sconf.IpWhiteList[ip]; !ok {
			WSCounterInc("session_manager", "max_ip_limit")
			glog.Info(context.Background(), "session exceed ip limit", glog.String("ip", ip))
			return errTooManySessionPerIP
		}
	}

	m.ips[ip]++
	m.sessions[s.ID()] = s
	WSGauge(float64(len(m.sessions)), "session_manager", "sessions")
	glog.Info(
		context.Background(),
		"add new session",
		glog.String("clientInfo", s.GetClient().String()),
		glog.String("sessionID", s.ID()),
		glog.String("protocol", s.ProtocolVersion().String()),
	)
	return nil
}

func (m *sessionMgr) DelSession(sessId string) {
	uid := int64(0)
	m.mux.Lock()
	s, ok := m.sessions[sessId]
	if !ok {
		m.mux.Unlock()
		return
	}
	delete(m.sessions, sessId)
	sessionSize := len(m.sessions)
	ip := s.GetClient().GetIP()
	ipCount := m.ips[ip] - 1
	if ipCount <= 0 {
		delete(m.ips, ip)
	} else {
		m.ips[ip] = ipCount
	}
	m.mux.Unlock()

	// 统计和日志
	WSGauge(float64(sessionSize), "session_manager", "sessions")

	status := s.GetStatus()
	glog.Info(context.Background(), "del session success",
		glog.Int64("uid", s.GetClient().GetMemberId()),
		glog.String("client", s.GetClient().String()),
		glog.String("sess_id", s.ID()),
		glog.Int64("sess_write_count", status.WriteCount),
		glog.Int64("sess_drop_count", status.DropCount),
		glog.String("sess_duration", time.Since(s.GetStartTime()).String()),
		glog.String("ip", ip),
		glog.Int("ip_count", ipCount),
	)

	// 从对应user中删除session
	uid = s.GetClient().GetMemberId()
	user := GetUserMgr().Unbind(uid, sessId)
	if user != nil {
		action := newAction(ActionSessionOffline, uid, sessId, nil)
		DispatchEvent(NewSyncOneUserEvent(user, action))
	}
}

func (m *sessionMgr) FillStatus(st *Status) {
	sessions := m.GetSessions()

	st.SessionCount = len(sessions)
	sessionInfos := make([]string, 0)
	for _, s := range sessions {
		topics := s.GetClient().GetTopics()
		info := fmt.Sprintf("userId: %v, topics: %v", s.GetClient().GetMemberId(), strings.Join(topics.Values(), ","))
		sessionInfos = append(sessionInfos, info)
	}
	st.Sessions = sessionInfos
}
