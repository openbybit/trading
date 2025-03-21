package ws

import (
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
)

func TestUserMgr(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	const userSize = 5
	um := newUserMgr()

	for i := 1; i <= userSize; i++ {
		sess := newSession(nil, NewClient(nil), version2)
		if _, err := um.Bind(int64(i), sess); err != nil {
			t.Error("first bind failed", err)
		}
	}

	assert.Equal(t, userSize, um.Size())

	for i := 1; i <= userSize; i++ {
		user := um.GetUser(int64(i))
		assert.NotNil(t, user)
		assert.EqualValues(t, 1, user.Size())
	}

	t.Run("GetUsers", func(t *testing.T) {
		assert.Equal(t, 2, len(um.GetUsers([]int64{1, 2})))
	})

	t.Run("get all users", func(t *testing.T) {
		assert.Equal(t, userSize, len(um.GetAllUsers()))
	})

	t.Run("not exists", func(t *testing.T) {
		assert.Equal(t, userSize, len(um.GetAllUsers()))
	})

	t.Run("is_in_blacklist", func(t *testing.T) {
		assert.False(t, um.IsInBlackList(1))
	})
}

func TestBind(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	um := newUserMgr()
	uid := int64(1)
	sess := newSession(nil, NewClient(nil), version2)
	_, err := um.Bind(uid, sess)
	if err != nil {
		t.Error("bind failed")
	}

	// 多次重复bind会忽略
	_, err = um.Bind(uid, sess)
	assert.Nil(t, err, "ignore duplicate bind")

	assert.NotNilf(t, um.Unbind(uid, sess.ID()), "unbind normal")
	// unbind幂等,可以多次调用
	assert.Nilf(t, um.Unbind(uid, sess.ID()), "idempotent unbind")

	// unbind zero
	assert.Nilf(t, um.Unbind(0, ""), "unbind zero uid")
}

func TestGetAllUsers(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	count := int64(10)
	um := newUserMgr()
	for i := int64(0); i < count; i++ {
		um.getOrCreateUser(i)
	}

	users := um.GetAllUsers()
	if len(users) != int(count) {
		t.Error("invalid size")
	}

	for _, u := range users {
		user := um.GetUser(u.GetMemberID())
		if user == nil {
			t.Error("invalid user", u.GetMemberID())
		}
	}
}
