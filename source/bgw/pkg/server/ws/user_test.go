package ws

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	uid := int64(1)
	u := newUser(uid)
	assert.EqualValues(t, uid, u.GetMemberID())
	assert.NotZero(t, u.GetCreateTime())
	assert.Zero(t, u.Size())
	assert.True(t, u.Empty())
	assert.False(t, u.IsDeleted())
	assert.Nil(t, u.getJoinParams())
	assert.Nil(t, u.GetParams())

	// add session1
	topicsList1 := []string{"t1", "t2"}
	c1 := NewClient(&ClientConfig{})
	c1.Subscribe(topicsList1)
	s1 := newSession(nil, c1, version2)
	assert.Nil(t, u.Add(s1))

	// add session2
	topicsList2 := []string{"t3", "t4"}
	c2 := NewClient(&ClientConfig{})
	c2.Subscribe(topicsList2)
	s2 := newSession(nil, c2, version2)
	assert.Nil(t, u.Add(s2))

	assert.Zero(t, len(u.GetParams()))

	// build session
	t.Run("test build", func(t *testing.T) {
		u.Build()
		topics := newTopics()
		topics.Add(topicsList1...)
		topics.Add(topicsList2...)
		t1 := topics.Values()
		sort.Strings(t1)
		assert.EqualValues(t, t1, u.GetTopics())
	})

	t.Run("test rebuild", func(t *testing.T) {
		s := newMockSession(1)
		s.client = NewClient(&ClientConfig{Params: map[string]string{"key-a": "aa"}})
		u := newUser(1)
		_ = u.Add(s)
		u.rebuildParams()
	})

	// test filter session
	t.Run("filter by session id", func(t *testing.T) {
		fs2 := u.FilterSessions(false, "", s2.ID())
		assert.Equal(t, 1, len(fs2))
		assert.Equal(t, s2.ID(), fs2[0].ID())

		notFind := u.FilterSessions(false, "", "not_exist_id")
		assert.Nil(t, notFind)
	})

	t.Run("filter by all", func(t *testing.T) {
		assert.Equal(t, 2, len(u.FilterSessions(true, "", "")))
	})

	t.Run("filter by topic", func(t *testing.T) {
		fs1 := u.FilterSessions(false, "t1", "")
		assert.Equal(t, 1, len(fs1))
		assert.Equal(t, s1.ID(), fs1[0].ID())
	})

	// remove session
	t.Run("remote", func(t *testing.T) {
		assert.EqualValues(t, 2, len(u.GetSessions()))
		u.Remove(s1.ID())
		assert.EqualValues(t, 1, u.Size())
		u.Remove(s2.ID())
		assert.EqualValues(t, 0, u.Size())
	})

	// test add fail
	t.Run("add_fail", func(t *testing.T) {
		u := newUser(uid)
		// deleted
		u.deleted = 1
		assert.Equal(t, errUserDeleted, u.Add(newMockSession(uid)))
		// over max
		u1 := newUser(1)
		sconf := getDynamicConf()
		sconf.MaxSessionsPerUser = 1
		assert.Nil(t, u1.Add(newMockSession(uid)))
		assert.True(t, isError(u1.Add(newMockSession(uid)).(CodeError), errTooManySession))
	})

	t.Run("CanForceSync", func(t *testing.T) {
		u := newUser(uid)
		assert.True(t, u.CanForceSync())
		assert.False(t, u.CanForceSync())
	})
}
