package ws

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

type UserMgr interface {
	IsInBlackList(uid int64) bool // 用户是否被封禁

	Size() int
	GetUser(uid int64) User
	GetUsers(uids []int64) []User
	GetAllUsers() []User

	Bind(uid int64, sess Session) (User, error)
	Unbind(uid int64, sessId string) User

	FillStatus(st *Status)
}

func GetUserMgr() UserMgr {
	return gUserMgr
}

var gUserMgr = newUserMgr()

type userMgr struct {
	users sync.Map
	size  int32
}

func newUserMgr() *userMgr {
	return &userMgr{}
}

func (um *userMgr) IsInBlackList(uid int64) bool {
	_, ok := getDynamicConf().UidBlackList[uid]
	return ok
}

func (um *userMgr) Size() int {
	return int(atomic.LoadInt32(&um.size))
}

func (um *userMgr) GetUser(uid int64) User {
	res, ok := um.users.Load(uid)
	if ok {
		user, ok := res.(User)
		if ok {
			return user
		}
	}

	return nil
}

func (um *userMgr) GetUsers(uids []int64) []User {
	res := make([]User, 0, len(uids))
	for _, uid := range uids {
		user := um.GetUser(uid)
		if user != nil {
			res = append(res, user)
		}
	}
	return res
}

func (um *userMgr) GetAllUsers() []User {
	res := make([]User, 0, um.Size())
	um.users.Range(func(key, value interface{}) bool {
		user, ok := value.(User)
		if ok {
			res = append(res, user)
		}
		return true
	})

	return res
}

func (um *userMgr) Bind(uid int64, sess Session) (User, error) {
	const retryBindMax = 10

	// 添加过程有可能失败,在高并发场景下,当用户session连接数变为零时，突然有来了一个新连接, 由于删除用户时,并没有锁用户
	// 可能存在一种场景: 用户从manager中删除了,但session连接数不为零(新的连接加入到了一个即将删除的用中)
	// 因此: 用户上有个deleted标识,每个用户连接数只能有一次从非零到零
	for i := 0; i < retryBindMax; i++ {
		user := um.getOrCreateUser(uid)
		err := user.Add(sess)
		if err != nil && errors.Is(err, errUserDeleted) {
			// 用户即将被删除,需要删除后重试
			um.deleteUser(uid)
			continue
		}

		if err != nil {
			WSCounterInc("user_manager", "max_per_user_limit")
			glog.Info(context.Background(), "session exceed user max size", glog.Int64("uid", uid))
			return nil, err
		}
		return user, nil
	}

	// 理论上走不到这里
	WSErrorInc("user_manager", "bind_fail")
	return nil, errUserDeleted
}

func (um *userMgr) getOrCreateUser(uid int64) User {
	user := um.GetUser(uid)
	if user != nil {
		return user
	}

	tmpUser := User(newUser(uid))
	resUser, loaded := um.users.LoadOrStore(uid, tmpUser)
	user, _ = resUser.(User)
	if user == nil {
		// 异常情况
		WSErrorInc("user_manager", "create_user_fail")
		user = tmpUser
	}

	if !loaded {
		// 新建用户
		atomic.AddInt32(&um.size, 1)
		WSCounterInc("user_manager", "add")
		glog.Info(context.Background(), "new user", glog.Int64("uid", uid))
	}

	WSGauge(float64(um.Size()), "user_manager", "users")
	return user
}

func (um *userMgr) Unbind(uid int64, sessId string) User {
	if uid == 0 {
		return nil
	}
	user := um.GetUser(uid)
	if user == nil {
		return nil
	}

	empty := user.Remove(sessId)
	if empty {
		um.deleteUser(uid)
	}

	return user
}

// 删除用户,并发安全且可重入,多次调用不会导致异常
func (um *userMgr) deleteUser(uid int64) {
	user := um.GetUser(uid)
	if user != nil && user.IsDeleted() {
		_, loaded := um.users.LoadAndDelete(uid)
		if loaded {
			// 删除成功才修改计数
			atomic.AddInt32(&um.size, -1)
			WSGauge(float64(um.Size()), "user_manager", "users")
			WSCounterInc("user_manager", "delete")
			glog.Info(context.Background(), "del user", glog.Int64("uid", uid), glog.Duration("duration", time.Duration(nowUnixNano()-user.GetCreateTime())))
		}
	}
}

func (um *userMgr) FillStatus(st *Status) {
	st.UserCount = um.Size()
	allUsers := um.GetAllUsers()
	users := make(map[int64]string)
	for _, u := range allUsers {
		topics := u.GetTopics()
		sort.Strings(topics)
		users[u.GetMemberID()] = strconv.Itoa(len(topics)) + ":" + strings.Join(topics, ",")
	}
	st.Users = users
}
