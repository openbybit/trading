package ws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"git.bybit.com/svc/stub/pkg/svc/common"
	"github.com/stretchr/testify/assert"

	"bgw/pkg/common/berror"
	"bgw/pkg/server/ws/mock"
	"bgw/pkg/service/ban"
	"bgw/pkg/service/masque"
	"bgw/pkg/service/openapi"
)

// test-6
// apiKey:     7jKvy9LPV7QY0oQM7o
// secretkey:  ZlWofaV3gwBIC4DxZYFZFNM3iJviL4V35HqO
func buildSign(secretKey string, unixTime string) string {
	signHash := hmac.New(sha256.New, []byte(secretKey))
	sign := "GET/realtime" + unixTime
	signHash.Write([]byte(sign))
	sigStr := hex.EncodeToString(signHash.Sum(nil))
	return sigStr
}

func TestAuth(t *testing.T) {
	const secretKey = "1XhRXBFSYebP6LjwFHhrmfFwNAI1W30vzon5"
	unixTime := strconv.FormatInt(time.Now().Add(time.Hour*1800).UnixMilli(), 10)
	// unixTime := "1758214477659"
	sign := buildSign(secretKey, unixTime)
	t.Log(unixTime, sign)
}

func TestGetHandler(t *testing.T) {
	vlist := []versionType{version2, version3, version5, versionOptionV1, versionOptionV3, versionType(-1)}
	for _, v := range vlist {
		assert.NotNil(t, getHandler(v))
	}
}

type mockEofReader struct {
}

func (r *mockEofReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func TestReadRequest(t *testing.T) {
	t.Run("io eof", func(t *testing.T) {
		err := readRequest(&mockEofReader{}, nil)
		assert.True(t, errors.Is(err, io.ErrUnexpectedEOF))
	})

	t.Run("invalid data", func(t *testing.T) {
		err := readRequest(strings.NewReader(`{"op":`), nil)
		assert.Equal(t, errParamsErr.Code(), toCodeErr(err).Code())
	})
}

func TestParsePathAndVersion(t *testing.T) {
	t.Run("parse version", func(t *testing.T) {
		type pair struct {
			Key     string
			Name    string
			Version versionType
		}
		list := []pair{
			{"unknown", "unknown", versionNone},
			{"v2", "v2", version2},
			{"v3", "v3", version3},
			{"v5", "v5", version5},
			{"option_v1", "option_v1", versionOptionV1},
			{"option_v3", "option_v3", versionOptionV3},
		}
		for _, x := range list {
			v := parseVersion(x.Key)
			assert.EqualValues(t, x.Version, v)
			assert.EqualValues(t, x.Name, x.Version.String())
		}
	})

	t.Run("version", func(t *testing.T) {
		type Pair struct {
			Path    string
			Version versionType
		}

		list := []Pair{
			{"/realtime_private", version2},
			{"/contract/private/v3", version3},
			{"/unified/private/v3", versionOptionV3},
			{"/trade/option/usdc/private/v1", versionOptionV1},
			{"/v5/private", version5},
		}
		for _, p := range list {
			x, v := parsePathAndVersion(p.Path)
			assert.Equal(t, p.Path, x)
			assert.Equal(t, p.Version, v)
		}

	})

	// 默认版本
	t.Run("default version", func(t *testing.T) {
		path := "/unknown_path"
		p, v := parsePathAndVersion(path)
		assert.Equal(t, version2, v)
		assert.Equal(t, path, p)
	})

	// 强制指定版本
	t.Run("assigned version", func(t *testing.T) {
		path := "v3:/v5/private"
		p, v := parsePathAndVersion(path)
		assert.Equal(t, version3, v)
		assert.Equal(t, "/v5/private", p)
	})
}

func TestSendResponse(t *testing.T) {
	// 无法序列化
	_ = sendResponse(nil, func() {})
}

func TestCheckTopicConflict(t *testing.T) {
	type pair struct {
		oldTopics []string
		newTopics []string
		success   []string
		fail      []string
	}

	const (
		topicOrder        = "order"
		topicOrderLinear  = "order.linear"
		topicOrderInverse = "order.inverse"
		topicPosition     = "position"
		topicWallet       = "wallet"
	)

	list := []pair{
		{
			oldTopics: []string{topicOrder},
			newTopics: []string{topicOrderLinear},
			success:   nil,
			fail:      []string{topicOrderLinear},
		},
		{
			oldTopics: []string{topicOrderLinear},
			newTopics: []string{topicOrder},
			success:   nil,
			fail:      []string{topicOrder},
		},
		{
			oldTopics: nil,
			newTopics: []string{topicOrder, topicOrderLinear},
			success:   []string{topicOrder},
			fail:      []string{topicOrderLinear},
		},
		{
			oldTopics: nil,
			newTopics: []string{topicOrderLinear, topicOrder},
			success:   []string{topicOrderLinear},
			fail:      []string{topicOrder},
		},
		{
			oldTopics: nil,
			newTopics: []string{topicWallet, topicOrderLinear, topicOrder, topicPosition},
			success:   []string{topicWallet, topicOrderLinear, topicPosition},
			fail:      []string{topicOrder},
		},
	}

	for index, v := range list {
		s, f := checkTopicConflict(v.newTopics, v.oldTopics)
		assert.EqualValuesf(t, v.success, s, "succ index: %v", index)
		assert.EqualValues(t, v.fail, f, "fail index: %v", index)
	}
}

func TestUserBan(t *testing.T) {
	sconf := getDynamicConf()
	sconf.DisableBan = true
	isBan, err := checkUserIsBanned(1)
	assert.Nil(t, err)
	assert.False(t, isBan)

	sconf.DisableBan = false
	t.Run("ban nil", func(t *testing.T) {
		svc, _ := getBanService()
		ban.SetBanService(nil)
		_, err = checkUserIsBanned(2)
		assert.Error(t, err)
		ban.SetBanService(svc)
	})

	t.Run("Ban type Login", func(t *testing.T) {
		ban.SetBanService(&mock.Ban{
			LoginBanType: 0,
		})
		ok, err := checkUserIsBanned(2)
		assert.Truef(t, ok, "err: %v", err)
		assert.NoErrorf(t, err, "err: %v", err)
	})

	t.Run("Ban false", func(t *testing.T) {
		ban.SetBanService(&mock.Ban{
			LoginBanType: 1,
		})
		ok, err := checkUserIsBanned(2)
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("ban rpc err", func(t *testing.T) {
		ban.SetBanService(&mock.Ban{
			Err: fmt.Errorf("test"),
		})
		ok, err := checkUserIsBanned(2)
		assert.False(t, ok)
		assert.NoError(t, err)
	})
}

func TestOnLogin(t *testing.T) {
	token := "123456"
	sess := newSession(nil, NewClient(&ClientConfig{}), version2)
	t.Run("no_token", func(t *testing.T) {
		err := onLogin(context.Background(), nil, "")
		assert.True(t, isError(err, errEmptyParameter))
	})

	t.Run("repeat_auth", func(t *testing.T) {
		s := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = s.SetMember(1)
		err := onLogin(context.Background(), s, token)
		assert.True(t, isError(err, errRepeatedAuth))
	})

	t.Run("rate_limit", func(t *testing.T) {
		userServiceLimit = *newRateLimit(1, 0)
		userServiceLimit.Allow()
		err := onLogin(context.Background(), sess, token)
		assert.True(t, isError(err, errReqLimit))
		userServiceLimit.Set(0, 0)
	})

	t.Run("masq nil", func(t *testing.T) {
		old, _ := getMasqueService()
		masque.SetMasqueService(nil)
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err, errServiceNotAvailable)
		masque.SetMasqueService(old)
	})

	t.Run("masq 12", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{
			Err: &common.Error{
				ErrorCode: 12,
			},
		})
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err, errParamsErr)
	})

	t.Run("masq 12", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{
			Err: &common.Error{
				ErrorCode: 10,
			},
		})
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err.Code(), 20002)
	})

	t.Run("masq uid 0", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{})
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err.Code(), 10001)
	})

	t.Run("masq rpc err", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{
			RpcErr: fmt.Errorf("test"),
		})
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err.Code(), 20002)
	})

	t.Run("masq", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{
			Uid: 12345,
		})
		err := onLogin(context.Background(), sess, "invalid")
		assert.Equal(t, err, nil)
	})

	t.Run("masq binduser", func(t *testing.T) {
		sess = newSession(nil, NewClient(&ClientConfig{}), version2)
		sconf := getDynamicConf()
		sconf.MaxSessionsPerUser = 0
		err := onLogin(context.Background(), sess, "invalid")
		assert.Error(t, err)
		sconf.MaxSessionsPerUser = 10000
	})

	t.Run("masq IsInBlackList", func(t *testing.T) {
		sess = newSession(nil, NewClient(&ClientConfig{}), version2)
		sconf := getDynamicConf()
		sconf.UidBlackList = Int64Set{12345: struct{}{}}
		err := onLogin(context.Background(), sess, "invalid")
		assert.Error(t, err)
		sconf.MaxSessionsPerUser = 10000
		sconf.UidBlackList = Int64Set{}
	})
}

func TestOnAuth(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	sess := newSession(nil, NewClient(&ClientConfig{}), version2)

	t.Run("repeat", func(t *testing.T) {
		s := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = s.SetMember(1)
		err := onAuth(s, "aa", time.Now().UnixMilli(), "bb")
		assert.True(t, isError(err, errRepeatedAuth))
	})

	t.Run("expires", func(t *testing.T) {
		err := onAuth(sess, "aa", 0, "bb")
		assert.True(t, isError(err, errParamsErr))
	})

	t.Run("no_service", func(t *testing.T) {
		// 避免panic
		gmetric.Init("ws")
		openapi.SetOpenapiService(nil)

		err := onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errServiceNotAvailable), "%v", err)
	})

	t.Run("verify_fail", func(t *testing.T) {
		openapi.SetOpenapiService(&mock.User{Err: berror.ErrBadSign})
		err := onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errDeniedAPIKey), "%v", err)

		openapi.SetOpenapiService(&mock.User{Err: fmt.Errorf("test")})
		err = onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errDeniedAPIKey), "%v", err)
	})

	t.Run("is_ban", func(t *testing.T) {
		openapi.SetOpenapiService(&mock.User{MemberId: 12345})
		ban.SetBanService(&mock.Ban{
			LoginBanType: 0,
		})
		err := onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errUserBanned), "%v", err)
		ban.SetBanService(&mock.Ban{LoginBanType: 2})
	})

	t.Run("in_blacklist", func(t *testing.T) {
		cfg := getDynamicConf()
		cfg.UidBlackList = Int64Set{12345: struct{}{}}
		err := onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errUserBanned), "%v", err)
		cfg.UidBlackList = Int64Set{}
	})

	t.Run("sign_fail", func(t *testing.T) {
		err := onAuth(sess, "aa", time.Now().UnixMilli(), "bb")
		assert.Truef(t, isError(err, errInvalidSign), "%v", err)
	})

	t.Run("bindUser", func(t *testing.T) {
		sconf := getDynamicConf()
		sconf.MaxSessionsPerUser = 0
		sess = newSession(nil, NewClient(&ClientConfig{}), version2)
		secret := "abc"
		openapi.SetOpenapiService(&mock.User{MemberId: 12345, LoginSecret: secret})
		expires := time.Now().UnixMilli() + 5000
		payload := fmt.Sprintf("%s%d", "GET/realtime", expires)
		s, _ := sign.Sign(sign.TypeHmac, []byte(secret), []byte(payload))
		err := onAuth(sess, "aa", expires, s)
		assert.Equal(t, 10003, err.(CodeError).Code())
		sconf.MaxSessionsPerUser = 10000
	})

	t.Run("normal", func(t *testing.T) {
		sess = newSession(nil, NewClient(&ClientConfig{}), version2)
		secret := "abc"
		openapi.SetOpenapiService(&mock.User{MemberId: 12345, LoginSecret: secret})
		expires := time.Now().UnixMilli() + 5000
		payload := fmt.Sprintf("%s%d", "GET/realtime", expires)
		s, _ := sign.Sign(sign.TypeHmac, []byte(secret), []byte(payload))
		err := onAuth(sess, "aa", expires, s)
		assert.NoError(t, err)
	})
}

func TestOnSubscribe(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	uid := int64(1)
	privateTopic := "private_topic"
	publicTopic := "public_topic"
	topics := []string{privateTopic}

	getConfigMgr().AddTopic(topicTypePrivate, []string{privateTopic})
	getConfigMgr().AddTopic(topicTypePublic, []string{publicTopic})

	sess := newSession(nil, NewClient(&ClientConfig{}), version2)
	_ = sess.SetMember(uid)

	appConf := getAppConf()
	appConf.SetDefaultTesting()

	t.Run("no_topic", func(t *testing.T) {
		_, _, _, err := onSubscribe(nil, nil)
		assert.True(t, isError(err, errEmptyParameter))
	})

	t.Run("not_auth", func(t *testing.T) {
		_, _, _, err := onSubscribe(sess, topics)
		assert.Error(t, err)
	})

	accId := "accid"
	_ = gAcceptorMgr.Add(newAcceptor(nil, accId, "appid", []string{privateTopic}, nil))
	_, _ = GetUserMgr().Bind(uid, sess)

	t.Run("invalid_topics", func(t *testing.T) {
		appConf.DisableSubscribeCheck = false
		_, _, _, err := onSubscribe(sess, []string{"invalid_topic"})
		assert.Truef(t, isError(err, errParamsErr), "err: %v", err)
		appConf.DisableSubscribeCheck = true
	})

	t.Run("topic conflict", func(t *testing.T) {
		getAppConf().EnableTopicConflictCheck = true
		sess := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = sess.SetMember(uid)
		_, _, _, err := onSubscribe(sess, []string{"order"})
		assert.Nil(t, err)
		_, _, _, err = onSubscribe(sess, []string{"order.linear"})
		assert.NotNil(t, err)
		getAppConf().EnableTopicConflictCheck = false
	})

	t.Run("success", func(t *testing.T) {
		_, _, _, err := onSubscribe(sess, []string{privateTopic})
		assert.Nil(t, err)
	})
	GetUserMgr().Unbind(uid, sess.ID())
	gAcceptorMgr.Remove(accId)
}

func TestOnUnsubscribe(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	uid := int64(1)
	topics := []string{"topic"}
	sess := newSession(nil, NewClient(&ClientConfig{}), version2)
	_ = sess.SetMember(uid)

	t.Run("no_topic", func(t *testing.T) {
		_, _, err := onUnsubscribe(nil, nil)
		assert.True(t, isError(err, errEmptyParameter))
	})

	t.Run("not_auth", func(t *testing.T) {
		_, _, err := onUnsubscribe(sess, topics)
		assert.Nil(t, err)
	})

	_, _ = GetUserMgr().Bind(uid, sess)

	// t.Run("invalid_topics", func(t *testing.T) {
	// 	disableSubscribeCheck = false
	// 	_, _, err := onSubscribe(sess, []string{"invalid_topic"})
	// 	assert.Truef(t, isError(err, errParamsErr), "err: %v", err)
	// })

	t.Run("success", func(t *testing.T) {
		_, _, err := onUnsubscribe(sess, []string{"topic"})
		assert.Nil(t, err)
	})

	GetUserMgr().Unbind(uid, sess.ID())
}

func TestOnInput(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	uid := int64(1)
	topic := "topic"
	sess := newSession(nil, NewClient(&ClientConfig{}), version2)
	_ = sess.SetMember(uid)

	t.Run("invalid_param", func(t *testing.T) {
		err := onInput(nil, "", "", "")
		assert.True(t, isError(err, errEmptyParameter))
	})

	t.Run("too_large_size", func(t *testing.T) {
		getDynamicConf().InputDataSize = 1
		err := onInput(nil, "reqid", topic, "aa")
		assert.Truef(t, isError(err, errParamsErr), "err: %v", err)
		getDynamicConf().InputDataSize = defaultInputDataSize
	})

	t.Run("not_auth", func(t *testing.T) {
		err := onInput(sess, "reqid", topic, "aa")
		assert.Truef(t, isError(err, errNotAuthorized), "err: %v", err)
	})

	_, _ = GetUserMgr().Bind(uid, sess)

	t.Run("no_service", func(t *testing.T) {
		err := onInput(sess, "reqid", topic, "aa")
		assert.Truef(t, isError(err, errNoServiceByTopic), "err: %v", err)
	})

	accId := "id"
	_ = gAcceptorMgr.Add(newAcceptor(nil, accId, "1", []string{topic}, nil))

	t.Run("success", func(t *testing.T) {
		err := onInput(sess, "reqid", topic, "aa")
		assert.Nil(t, err)
	})
	GetUserMgr().Unbind(uid, sess.ID())
	gAcceptorMgr.Remove(accId)
}

func TestOnTrade(t *testing.T) {
	t.Run("ctx pool", func(t *testing.T) {
		p := &ctxBufferPool
		x := p.Get()
		p.Put(x)
		// ignore
		p.Put(nil)

		s := newMockSession(123)
		_, _, err := onTrade(s, nil)
		assert.Error(t, err)

		s.client.SetAPIKey("aaa")
		_, _, err = onTrade(s, nil)
		assert.Error(t, err)

		args := []interface{}{
			"/option/usdc/openapi/ex/private/v1/batch-place-orders",
			"111",
		}
		_, _, err = onTrade(s, args)
		assert.Error(t, err)

		args = []interface{}{
			"/option/usdc/openapi/ex/private/v1/batch-place-orders",
			map[string]interface{}{
				"orderRequest": 123,
			},
		}
		_, _, err = onTrade(s, args)
		assert.Error(t, err)
	})

	t.Run("tradeHandle", func(t *testing.T) {
		p := &ctxBufferPool
		err := tradeHandle(p.Get())
		assert.Error(t, err)
	})
}

func TestHandlerV2(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	initExchange()
	getAppConf().SetDefaultTesting()
	uid := int64(1)

	verifyRsp := func(data []byte, op string, sessId string, msg string) {
		rsp := responseV2{}
		_ = json.Unmarshal(data, &rsp)
		assert.Truef(t, rsp.Success, "rsp success should be true")
		assert.Equal(t, sessId, rsp.ConnId)
		assert.Equalf(t, msg, rsp.Message, "msg: %v", rsp.Message)
	}

	verifyFail := func(data []byte, op string, sessId string, msg string) {
		rsp := responseV2{}
		_ = json.Unmarshal(data, &rsp)
		assert.Falsef(t, rsp.Success, "rsp success should be false")
		assert.Equal(t, sessId, rsp.ConnId)
		assert.Equalf(t, msg, rsp.Message, "msg: %v", rsp.Message)
	}

	t.Run("panic_recovery", func(t *testing.T) {
		data := `{"op": "login", "args": []}`
		_ = gHandlerV2.Handle(context.TODO(), nil, strings.NewReader(data))
	})

	t.Run("read request fail", func(t *testing.T) {
		_ = gHandlerV2.Handle(context.TODO(), newMockSession(1), nil)
	})

	t.Run("session_limit", func(t *testing.T) {
		sess := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = sess.SetMember(uid)
		sess.limit.Set(1, time.Second)
		sess.limit.Allow()
		_ = gHandlerV2.Handle(context.Background(), sess, nil)
	})

	t.Run("invalid args", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": []}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opLogin, sess.ID(), errParamsErr.Error())

		data = `{"op": "subscribe", "args": []}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opSubscribe, sess.ID(), errEmptyParameter.Error())

		data = `{"op": "unsubscribe", "args": []}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opUnsubscribe, sess.ID(), errEmptyParameter.Error())

		data = `{"op": "input", "args": []}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opUnsubscribe, sess.ID(), errParamsErr.Error())

		data = `{"op": "unknown", "args": []}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opUnsubscribe, sess.ID(), errParamsErr.Error())
	})

	t.Run("test", func(t *testing.T) {
		sess := newMockSession(uid)
		data := `{"op": "test"}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opTest, sess.ID(), "")
	})

	t.Run("ping", func(t *testing.T) {
		sess := newMockSession(uid)
		data := `{"op": "ping"}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opPing, sess.ID(), "pong")
	})

	t.Run("login", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": ["123456"]}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opLogin, sess.ID(), "")
	})

	t.Run("subscribe&unsubscribe", func(t *testing.T) {
		sess := newMockSession(uid)
		_, _ = GetUserMgr().Bind(uid, sess)
		//
		data := `{"op": "subscribe", "args": ["topic", ""]}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opSubscribe, sess.ID(), "")
		//
		data = `{"op": "unsubscribe", "args": ["topic"]}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opUnsubscribe, sess.ID(), "")

		GetUserMgr().Unbind(uid, sess.ID())
	})

	t.Run("input", func(t *testing.T) {
		sess := newMockSession(uid)

		accId := "id"
		topic := "topic"
		_ = gAcceptorMgr.Add(newAcceptor(nil, accId, "1", []string{topic}, nil))

		_, _ = GetUserMgr().Bind(uid, sess)
		data := `{"op": "input", "args": ["topic", "im data"]}`
		_ = gHandlerV2.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opInput, sess.ID(), "")
		GetUserMgr().Unbind(uid, sess.ID())
		gAcceptorMgr.Remove(accId)
	})
}

func TestHandlerV3(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	getAppConf().SetDefaultTesting()
	initExchange()
	uid := int64(1)

	verifyRsp := func(data []byte, op string, sessId string, msg string) {
		rsp := responseV3{}
		_ = json.Unmarshal(data, &rsp)
		assert.Truef(t, rsp.Success, "rsp success should be true")
		assert.Equalf(t, sessId, rsp.ConnID, "connid")
		assert.Equalf(t, msg, rsp.RetMsg, "msg: %v", rsp.RetMsg)
	}

	verifyFail := func(data []byte, op string, sessId string, msg string) {
		rsp := responseV3{}
		_ = json.Unmarshal(data, &rsp)
		assert.Falsef(t, rsp.Success, "rsp success should be false")
		assert.Equal(t, sessId, rsp.ConnID)
		assert.Equalf(t, msg, rsp.RetMsg, "msg: %v", rsp.RetMsg)
	}

	t.Run("panic_recovery", func(t *testing.T) {
		data := `{"op": "login", "args": []}`
		_ = gHandlerV3.Handle(context.TODO(), nil, strings.NewReader(data))
	})

	t.Run("read request fail", func(t *testing.T) {
		_ = gHandlerV3.Handle(context.TODO(), newMockSession(1), nil)
	})

	t.Run("session_limit", func(t *testing.T) {
		sess := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = sess.SetMember(uid)
		sess.limit.Set(1, time.Second)
		sess.limit.Allow()
		_ = gHandlerV3.Handle(context.Background(), sess, nil)
	})

	t.Run("invalid args", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": []}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opLogin, sess.ID(), errParamsErr.Error())

		data = `{"op": "auth", "args": []}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opAuth, sess.ID(), errParamsErr.Error())

		data = `{"op": "subscribe", "args": []}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opSubscribe, sess.ID(), errEmptyParameter.Error())

		data = `{"op": "unsubscribe", "args": []}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opUnsubscribe, sess.ID(), errEmptyParameter.Error())
	})

	t.Run("ping", func(t *testing.T) {
		sess := newMockSession(uid)
		data := `{"op": "ping"}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		rsp := responsePongV3{}
		_ = json.Unmarshal(sess.lastMsg.Data, &rsp)
		assert.Equalf(t, sess.ID(), rsp.ConnID, "connid")
		assert.Equal(t, 1, len(rsp.Args))
		assert.Equal(t, opPong, rsp.Op)
	})

	t.Run("login", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": ["123456"]}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opLogin, sess.ID(), "")
	})

	// t.Run("auth", func(t *testing.T) {
	// 	enableMockLogin = true
	// 	sess := newMockSession(0)
	// 	data := `{"op": "auth", "args": ["123456"]}`
	// 	gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
	// 	verifyRsp(sess.lastMsg.Data, opLogin, sess.ID(), "")
	// })

	t.Run("subscribe&unsubscribe", func(t *testing.T) {
		sess := newMockSession(uid)
		_, _ = GetUserMgr().Bind(uid, sess)
		//
		data := `{"op": "subscribe", "args": ["topic"]}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opSubscribe, sess.ID(), "")
		//
		data = `{"op": "unsubscribe", "args": ["topic"]}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opUnsubscribe, sess.ID(), "")

		GetUserMgr().Unbind(uid, sess.ID())
	})

	// TODO: ontrade

	t.Run("input", func(t *testing.T) {
		sess := newMockSession(uid)

		accId := "id"
		topic := "topic"
		_ = gAcceptorMgr.Add(newAcceptor(nil, accId, "1", []string{topic}, nil))

		_, _ = GetUserMgr().Bind(uid, sess)
		data := `{"op": "input", "args": ["topic", "im data"]}`
		_ = gHandlerV3.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opInput, sess.ID(), "")
		GetUserMgr().Unbind(uid, sess.ID())
		gAcceptorMgr.Remove(accId)
	})
}

func TestHandlerOption(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	getAppConf().SetDefaultTesting()
	uid := int64(1)

	verifyRsp := func(data []byte, op string, sessId string, msg string) {
		rsp := responseOption{}
		_ = json.Unmarshal(data, &rsp)
		assert.Truef(t, rsp.Success, "rsp success should be true")
		assert.Equalf(t, sessId, rsp.ConnId, "connid")
		assert.Equalf(t, msg, rsp.RetMsg, "msg: %v", rsp.RetMsg)
	}

	verifyFail := func(data []byte, op string, sessId string, msg string) {
		rsp := responseOption{}
		_ = json.Unmarshal(data, &rsp)
		assert.Falsef(t, rsp.Success, "rsp success should be false")
		assert.Equal(t, sessId, rsp.ConnId)
		assert.Equalf(t, msg, rsp.RetMsg, "msg: %v", rsp.RetMsg)
	}

	t.Run("panic_recovery", func(t *testing.T) {
		data := `{"op": "login", "args": []}`
		_ = gHandlerOption.Handle(context.TODO(), nil, strings.NewReader(data))
	})

	t.Run("read request fail", func(t *testing.T) {
		_ = gHandlerOption.Handle(context.TODO(), newMockSession(1), nil)
	})

	t.Run("session_limit", func(t *testing.T) {
		sess := newSession(nil, NewClient(&ClientConfig{}), version2)
		_ = sess.SetMember(uid)
		sess.limit.Set(1, time.Second)
		sess.limit.Allow()
		data := `{"op": "login", "args": []}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
	})

	t.Run("invalid args", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": []}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opLogin, sess.ID(), errParamsErr.Error())

		data = `{"op": "auth", "args": []}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opAuth, sess.ID(), errParamsErr.Error())

		data = `{"op": "subscribe", "args": []}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opSubscribe, sess.ID(), errEmptyParameter.Error())

		data = `{"op": "unsubscribe", "args": []}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyFail(sess.lastMsg.Data, opUnsubscribe, sess.ID(), errEmptyParameter.Error())
	})

	t.Run("ping", func(t *testing.T) {
		sess := newMockSession(uid)
		data := `{"op": "ping"}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		rsp := responsePongOption{}
		_ = json.Unmarshal(sess.lastMsg.Data, &rsp)
		assert.Equal(t, 1, len(rsp.Args))
		assert.Equal(t, opPong, rsp.Op)
	})

	t.Run("login", func(t *testing.T) {
		sess := newMockSession(0)
		data := `{"op": "login", "args": ["123456"]}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opLogin, sess.ID(), "0")
	})

	t.Run("subscribe&unsubscribe", func(t *testing.T) {
		sess := newMockSession(uid)
		_, _ = GetUserMgr().Bind(uid, sess)
		//
		data := `{"op": "subscribe", "args": ["topic"]}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opSubscribe, sess.ID(), "")
		//
		data = `{"op": "unsubscribe", "args": ["topic"]}`
		_ = gHandlerOption.Handle(context.Background(), sess, strings.NewReader(data))
		verifyRsp(sess.lastMsg.Data, opUnsubscribe, sess.ID(), "")

		GetUserMgr().Unbind(uid, sess.ID())
	})

	t.Run("test mapping error", func(t *testing.T) {
		type pair struct {
			Err CodeError
			Str string
		}
		list := []pair{
			{errRepeatedAuth, optionErrAUTH_REPEAT},
			{errReqLimit, optionErrREQ_COUNT_LIMIT},
			{errParamsErr, optionErrPARAMS_ERROR},
			{errDeniedAPIKey, optionErrVERIFY_SIGN_FAIL},
			{errAuthFail, optionErrAUTHFAIL},
			{errUserBanned, optionErrCOMMANDHANDLE_UNKNOWERROR},
		}
		for _, kv := range list {
			r := gHandlerOption.mappingErr(kv.Err)
			assert.Equal(t, kv.Str, r)
		}
	})
}
