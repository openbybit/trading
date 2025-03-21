package ws

import (
	"errors"
	"strings"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
)

func TestSessionManager(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	t.Run("add fail", func(t *testing.T) {
		assert.NotNil(t, GetSessionMgr())

		sconf := getDynamicConf()

		m := newSessionMgr()
		// add nil
		assert.NotNil(t, m.AddSession(nil))
		// ip limit
		ip := "blackip"
		sconf.IpBlackList[ip] = struct{}{}
		s1 := newSession(nil, NewClient(&ClientConfig{IP: ip}), version2)
		assert.True(t, errors.Is(m.AddSession(s1), errIPInBlacklist))
		sconf.IpBlackList = make(StringSet)

		// duplicate add
		s2 := newMockSession(1)
		assert.Nil(t, m.AddSession(s2))
		duplicateErr := m.AddSession(s2)
		assert.True(t, strings.Contains(duplicateErr.Error(), "duplicate uid"))

		// maxSessions
		sconf.MaxSessions = 1
		assert.True(t, errors.Is(m.AddSession(newMockSession(2)), errTooManySession))
		sconf.MaxSessions = defaultMaxSessions

		// MaxSessionsPerIp
		sconf.MaxSessionsPerIp = 1
		assert.True(t, errors.Is(m.AddSession(newMockSession(1)), errTooManySessionPerIP))
		sconf.MaxSessionsPerIp = defaultMaxSessionsPerIp
	})

	t.Run("basic", func(t *testing.T) {
		m := newSessionMgr()

		s1 := newSession(nil, NewClient(&ClientConfig{IP: "127.0.0.1"}), version2)
		_ = m.AddSession(s1)
		assert.Equal(t, 1, m.Size())
		assert.Equal(t, 1, len(m.GetSessions()))
		assert.Equal(t, s1, m.GetSession(s1.ID()))
		assert.Nil(t, m.GetSession("not_exist"))

		s2 := newSession(nil, NewClient(&ClientConfig{IP: "127.0.0.1"}), version2)
		_ = m.AddSession(s2)

		m.DelSession(s1.ID())
		m.DelSession(s1.ID())

		st := &Status{}
		m.FillStatus(st)
		assert.Equal(t, m.Size(), st.SessionCount)
		m.Close()
	})

	t.Run("create&delete session", func(t *testing.T) {
		m := newSessionMgr()
		s1 := newSession(nil, NewClient(&ClientConfig{IP: "127.0.0.1"}), version2)
		_ = m.AddSession(s1)

		_, _ = GetUserMgr().Bind(1, s1)
		m.DelSession(s1.ID())
	})
}
