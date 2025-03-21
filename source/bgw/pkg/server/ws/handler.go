package ws

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/core"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/ban"
	"bgw/pkg/service/masque"
)

var (
	// errUnknownErr          = newCodeErr(10000, "An unknown error occurred while processing the request.")
	errParamsErr           = newCodeErr(10001, "Params Error")
	errNotAuthorized       = newCodeErr(10002, "Request not authorized")
	errTooManySession      = newCodeErr(10003, "Too many requests")
	errInvalidSign         = newCodeErr(10004, "Invalid sign")
	errDeniedAPIKey        = newCodeErr(10005, "Permission denied for current apikey")
	errTooManySessionPerIP = newCodeErr(10008, "Exceeded IP rate limit.")
	errIPInBlacklist       = newCodeErr(10010, "Request IP mismatch.")
	errEmptyParameter      = newCodeErr(11005, "Empty parameter.")
	errServiceNotAvailable = newCodeErr(10016, "Service not available")
	// custom err code
	errRepeatedAuth     = newCodeErr(20001, "Repeat auth")            // 重复认证,注: 客户端有依赖,不能随意文本内容
	errAuthFail         = newCodeErr(20002, "Auth fail")              // login auth fail
	errReqLimit         = newCodeErr(20003, "Request limit exceeded") // 调用频率受限
	errUserBanned       = newCodeErr(20004, "User banned")            // 被拉黑
	errNoServiceByTopic = newCodeErr(20005, "No service by topic")    // 无法通过topic找到对应的服务
)

const (
	errTopicSubscribeFailFormat = `Batch subscription partially succeeded and partially failed.Successful subscriptions are as follows:%v. Subscription to the following topics failed because the topic does not exist or there is a subscription conflict:%v.`
)

var (
	userServiceLimit rateLimit
	logLimit         = newRateLimit(100, time.Minute)
)

func setUserServiceRateLimit(rateLimit int64) {
	userServiceLimit.Set(int64(rateLimit), time.Second)
}

type versionType int8

const (
	versionNone     versionType = 0
	version2        versionType = 2
	version3        versionType = 3
	version5        versionType = 5  // 格式同v3
	versionOptionV1 versionType = 10 // 期权v1,trade/option/usdc/private/v1
	versionOptionV3 versionType = 11 // 期权v3,/unified/private/v3
)

func (v versionType) String() string {
	switch v {
	case version2:
		return "v2"
	case version3:
		return "v3"
	case version5:
		return "v5"
	case versionOptionV1:
		return "option_v1"
	case versionOptionV3:
		return "option_v3"
	default:
		return "unknown"
	}
}

func parseVersion(v string) versionType {
	v = strings.TrimSpace(v)
	switch v {
	case "v2":
		return version2
	case "v3":
		return version3
	case "v5":
		return version5
	case "option_v1":
		return versionOptionV1
	case "option_v3":
		return versionOptionV3
	default:
		return versionNone
	}
}

func parsePathAndVersion(p string) (string, versionType) {
	var version versionType
	var path string
	idx := strings.IndexByte(p, ':')
	if idx != -1 {
		version = parseVersion(p[:idx])
		path = strings.TrimSpace(p[idx+1:])
	} else {
		path = p
		if strings.Contains(p, "/v5/") {
			version = version5
		} else if strings.Contains(p, "/v3/") || strings.HasSuffix(p, "/v3") {
			version = version3
		} else {
			version = version2
		}
	}

	/*兼容期权通信协议
	ws2.bybit.com/trade/option/usdc/private/v1   web/app
	stream.bybit.com/unified/private/v3 统保普通做市商（同上面一套集群）

	stream.bybit.com/trade/option/usdc/private/v1   期权普通做市商
	stream.bybit.com/trade/option/usdc/ex00/private/v1 期权极简00集群
	stream.bybit.com/trade/option/usdc/ex01/private/v1 期权极简01
	*/
	switch {
	case path == "/unified/private/v3":
		version = versionOptionV3
	case strings.HasPrefix(path, "/trade/option/usdc") && strings.HasSuffix(path, "/private/v1"):
		version = versionOptionV1
	}

	return path, version
}

var (
	gHandlerV2     = &handlerV2{}
	gHandlerV3     = &handlerV3{}
	gHandlerOption = &handlerOption{}
)

func getHandler(v versionType) Handler {
	switch v {
	case version2:
		return gHandlerV2
	case version3:
		return gHandlerV3
	case version5:
		return gHandlerV3
	case versionOptionV1, versionOptionV3:
		return gHandlerOption
	default:
		return gHandlerV2
	}
}

const (
	opTest        = "test"
	opPing        = "ping"
	opPong        = "pong"
	opAuth        = "auth"
	opLogin       = "login"
	opSubscribe   = "subscribe"
	opUnsubscribe = "unsubscribe"
	opTrade       = "trade"
	opInput       = "input"
)

// Handler is the interface that must be implemented by a websocket handler.
type Handler interface {
	Handle(ctx context.Context, sess Session, r io.Reader) error
}

func readRequest(reader io.Reader, req interface{}) error {
	err := jsonDecode(reader, req)
	if err != nil {
		if err == io.EOF {
			// One value is expected in the message.
			return io.ErrUnexpectedEOF
		}

		return errParamsErr
	}

	return nil
}

func sendResponse(sess Session, rsp interface{}) error {
	data, err := jsonMarshal(rsp)
	if err != nil {
		WSCounterInc("error", "send_rsp_marshal")
		return nil
	}

	return sess.Write(&Message{
		Type: MsgTypeReply,
		Data: data,
	})
}

func onLogin(ctx context.Context, sess Session, token string) CodeError {
	if len(token) == 0 {
		WSCounterInc("login", "no_token")
		return newCodeErrFrom(errEmptyParameter, "empty token")
	}

	if sess.IsAuthed() {
		WSCounterInc("login", "repeat_auth")
		// glog.Debug(context.Background(), "repeated login", glog.Int64("uid", sess.GetClient().GetMemberId()))
		return errRepeatedAuth
	}

	if !userServiceLimit.Allow() {
		WSCounterInc("login", "rate_limit")
		return errReqLimit
	}

	if getAppConf().EnableMockLogin {
		uid, err := decodeUserIDFromToken(token)
		if err == nil {
			err = bindUser(uid, sess)
			return toCodeErr(err)
		}
	}

	originUrl := sess.GetClient().GetReferer() + sess.GetClient().GetPath()
	uid, err := verifyToken(ctx, token, originUrl)
	if err != nil {
		return err
	}

	if err := bindUser(uid, sess); err != nil {
		WSCounterInc("login", "bind_user_fail")
		return toCodeErr(err)
	}
	return nil
}

func verifyToken(ctx context.Context, token, url string) (int64, CodeError) {
	m, err := getMasqueService()
	if err != nil {
		WSErrorInc("login", "no_masque_service")
		glog.Error(ctx,
			"[login] getMasqueService",
			glog.String("token", token),
			glog.String("error", err.Error()),
		)
		return 0, errServiceNotAvailable
	}

	now := time.Now()
	resp, err := m.MasqueTokenInvoke(ctx, "", token, url, masque.WeakAuth)
	WSHistogram(now, "login", "masque")

	if err != nil {
		WSErrorInc("login", "masque_invoke_error")
		glog.Error(context.Background(),
			"[login] invoke masque fail",
			glog.String("token", token),
			glog.String("error", err.Error()),
		)
		return 0, newCodeErrFrom(errAuthFail, "error=%v", err.Error())
	}

	if resp.Error != nil && resp.Error.ErrorCode != 0 {
		if resp.Error.ErrorCode == 12 {
			WSCounterInc("login", "invalid_token")
			return 0, errParamsErr
		}
		WSErrorInc("login", "masque_result_error")
		code := int64(resp.Error.ErrorCode)
		err := resp.Error.String()
		glog.Error(context.Background(),
			"[login] invoke masque fail, resp error",
			glog.String("token", token), glog.Int64("code", code),
			glog.String("error", err),
		)
		return 0, newCodeErrFrom(errAuthFail, "code=%v,error=%v", resp.Error.ErrorCode, resp.Error.ErrorResult)
	}

	if resp.UserId == 0 {
		WSErrorInc("login", "masque_empty_uid")
		glog.Error(context.Background(), "[login] invoke masque fail, invalid userId", glog.String("token", token))
		return 0, newCodeErrFrom(errParamsErr, "invalid userId")
	}

	if GetUserMgr().IsInBlackList(resp.UserId) {
		WSCounterInc("login", "masque_user_ban")
		if logLimit.Allow() {
			glog.Info(context.Background(), "[login] uid is banned", glog.Int64("uid", resp.UserId))
		}

		return 0, errUserBanned
	}

	return resp.UserId, nil
}

func onAuth(sess Session, apiKey string, expires int64, signature string) CodeError {
	if apiKey == "" || signature == "" {
		return newCodeErrFrom(errParamsErr, "apikey or signature is empty")
	}

	// check expired
	now := time.Now()
	if now.UnixNano()/1e6 > expires {
		return newCodeErrFrom(errParamsErr, "request expired")
	}

	if sess.IsAuthed() {
		WSCounterInc("auth", "repeat")
		// glog.Info(context.Background(), "repeated auth", glog.Int64("uid", sess.GetClient().GetMemberId()))
		return errRepeatedAuth
	}

	o, err := getUserService()
	if err != nil || isNil(o) {
		glog.Error(context.TODO(), "[auth] GetOpenapiService error", glog.String("error", err.Error()))
		WSErrorInc("auth", "no_user_service")
		return newCodeErrFrom(errServiceNotAvailable, "no openapi service")
	}

	// check apikey
	member, err := o.VerifyAPIKey(context.TODO(), apiKey, sess.GetClient().GetXOriginFrom())
	WSHistogram(now, "auth", "verify_api")

	if err != nil {
		if e, ok := err.(berror.BizErr); ok { // biz error
			wsCounter.Inc("auth", cast.ToString(e.GetCode()))
			glog.Debug(context.TODO(),
				"[auth] VerifyAPIKey fail",
				glog.String("apikey", apiKey),
				glog.String("ip", sess.GetClient().GetIP()),
				glog.String("sign", signature),
				glog.Int64("expires", expires),
				glog.String("error", err.Error()),
			)
		} else {
			glog.Error(context.TODO(),
				"[auth] VerifyAPIKey error",
				glog.String("apikey", apiKey),
				glog.String("ip", sess.GetClient().GetIP()),
				glog.String("sign", signature),
				glog.Int64("expires", expires),
				glog.String("error", err.Error()),
			)
			WSErrorInc("auth", "verify_api_fail")
		}
		return newCodeErrFrom(errDeniedAPIKey, "err=%v", err)
	}

	if isBan, _ := checkUserIsBanned(member.MemberId); isBan {
		return errUserBanned
	}

	// 内部黑名单
	if GetUserMgr().IsInBlackList(member.MemberId) {
		WSCounterInc("auth", "in_blacklist")
		return errUserBanned
	}

	// check sign
	payload := fmt.Sprintf("%s%d", "GET/realtime", expires)

	signTyp := sign.TypeHmac
	if member.ExtInfo != nil {
		signTyp = sign.Type(member.ExtInfo.Flag)
	}

	if err := sign.Verify(signTyp, []byte(member.LoginSecret), []byte(payload), signature); err != nil {
		WSCounterInc("auth", "sign_fail")
		glog.Info(context.Background(), "[auth] verify sign fail",
			glog.Int64("uid", member.MemberId),
			glog.String("ip", sess.GetClient().GetIP()),
			glog.String("type", string(signTyp)),
			glog.String("payload", payload),
			glog.String("sign", signature),
			glog.String("secret", member.LoginSecret),
		)
		return errInvalidSign
	}

	err = bindUser(member.MemberId, sess)
	if err != nil {
		WSCounterInc("auth", "bind_user_fail")
		glog.Info(context.Background(), "[auth] bind user failed", glog.Int64("uid", member.MemberId), glog.String("ip", sess.GetClient().GetIP()), glog.String("err", err.Error()))
		return toCodeErr(err)
	}

	sess.GetClient().SetAPIKey(apiKey)
	glog.Debug(context.Background(), "onAuth success", glog.Int64("uid", member.MemberId))
	return nil
}

// checkUserIsBanned 校验用户是否被封禁, 返回true表示被封禁,error用于标识请求是否报错
func checkUserIsBanned(uid int64) (bool, error) {
	sconf := getDynamicConf()
	if sconf.DisableBan {
		WSCounterInc("ban", "disable")
		return false, nil
	}

	banSvc, err := getBanService()
	if err != nil {
		WSErrorInc("ban", "no_service")
		return false, err
	}

	start := time.Now()
	status, err := banSvc.GetMemberStatus(context.TODO(), uid)
	WSHistogram(start, "ban", "get_member_status")

	if err == nil && status != nil {
		if status.LoginBanType == int32(ban.BantypeLogin) {
			WSCounterInc("ban", "ban_login")
			return true, nil
		}
	} else {
		WSErrorInc("ban", "result_error")
		glog.Error(context.TODO(), "[ban] result error", glog.Any("error", err))
	}

	return false, nil
}

func bindUser(uid int64, sess Session) error {
	user, err := GetUserMgr().Bind(uid, sess)
	if err == nil {
		action := newAction(ActionSessionOnline, uid, sess.ID(), nil)
		DispatchEvent(NewSyncOneUserEvent(user, action))
		glog.Info(
			context.Background(),
			"user login success",
			glog.Int64("uid", uid),
			glog.String("sessionID", sess.ID()),
			glog.String("clientInfo", sess.GetClient().String()),
			glog.String("protocol", sess.ProtocolVersion().String()),
		)
	}

	return err
}

var v5TopicConflictMap = map[string][]string{
	"order":             {"order.inverse", "order.linear", "order.option", "order.spot"},
	"execution":         {"execution.inverse", "execution.linear", "execution.option", "execution.spot"},
	"position":          {"position.inverse", "position.linear", "position.option", "position.spot"},
	"order.inverse":     {"order"},
	"order.linear":      {"order"},
	"order.option":      {"order"},
	"order.spot":        {"order"},
	"execution.inverse": {"execution"},
	"execution.linear":  {"execution"},
	"execution.option":  {"execution"},
	"execution.spot":    {"execution"},
	"position.inverse":  {"position"},
	"position.linear":   {"position"},
	"position.option":   {"position"},
	"position.spot":     {"position"},
}

// checkTopicConflict 校验topic是否有冲突
func checkTopicConflict(newTopics []string, oldTopics []string) (successes []string, fails []string) {
	dict := make(map[string]struct{})
	for _, t := range oldTopics {
		dict[t] = struct{}{}
	}

	for _, t := range newTopics {
		list, ok := v5TopicConflictMap[t]
		if !ok {
			// 不需要冲突检测,比如wallet
			successes = append(successes, t)
			continue
		}

		conflict := false

		for _, v := range list {
			if _, ok := dict[v]; ok {
				conflict = true
				break
			}
		}

		if !conflict {
			dict[t] = struct{}{}
			successes = append(successes, t)
		} else {
			fails = append(fails, t)
		}
	}

	return
}

func onSubscribe(sess Session, topics []string) (successes []string, fails []string, changed []string, err CodeError) {
	if len(topics) == 0 {
		WSCounterInc("subscribe", "invalid_params")
		return nil, nil, nil, errEmptyParameter
	}

	user := GetUserMgr().GetUser(sess.GetClient().GetMemberId())
	if getConfigMgr().HasPrivateTopics(topics) {
		// 如果含有私有推送topic,订阅前需要先鉴权
		if user == nil {
			WSCounterInc("subscribe", "not_auth")
			return nil, nil, nil, errNotAuthorized
		}
	}

	appConf := getAppConf()

	// 校验topic是否合法
	if !appConf.DisableSubscribeCheck {
		successes, fails = getConfigMgr().CheckTopics(topics)
		if len(fails) > 0 {
			WSCounterInc("subscribe", "invalid_topic")
			err = newCodeErrFrom(errParamsErr, "fail_topics=%v", fails)
		}
	} else {
		successes = topics
	}

	// 校验topic注册是否冲突
	if appConf.EnableTopicConflictCheck {
		var tmpFails []string
		successes, tmpFails = checkTopicConflict(successes, sess.GetClient().GetTopics().Values())
		fails = append(fails, tmpFails...)
		if len(fails) > 0 {
			err = newCodeErr(errParamsErr.Code(), errTopicSubscribeFailFormat, successes, fails)
		}
	}

	changed = sess.GetClient().Subscribe(successes)

	if len(changed) == 0 {
		return
	}

	if user != nil {
		action := newAction(ActionSessionSub, user.GetMemberID(), sess.ID(), changed)
		DispatchEvent(NewSyncOneUserEvent(user, action))
	}

	glog.Info(context.Background(),
		"user subscribe topic success",
		glog.String("client", sess.GetClient().String()),
		glog.String("sessionID", sess.ID()),
		glog.Any("topics", topics),
		glog.Any("changed", changed),
	)

	return
}

func onUnsubscribe(sess Session, topics []string) (successes []string, fails []string, err CodeError) {
	if len(topics) == 0 {
		WSCounterInc("unsubscribe", "invalid_params")
		return nil, nil, errEmptyParameter
	}

	changed := sess.GetClient().Unsubscribe(topics)
	if len(changed) == 0 {
		return
	}

	gPublicMgr.OnUnsubscribe(sess, changed)

	user := GetUserMgr().GetUser(sess.GetClient().GetMemberId())
	if user != nil {
		action := newAction(ActionSessionUnsub, user.GetMemberID(), sess.ID(), changed)
		DispatchEvent(NewSyncOneUserEvent(user, action))
	}

	glog.Info(context.Background(), "user unsubscribe topic success",
		glog.String("client", sess.GetClient().String()),
		glog.String("sessionID", sess.ID()),
		glog.Any("topics", topics),
	)
	return topics, nil, nil
}

// onInput 上行数据
func onInput(sess Session, reqId string, topic string, data string) CodeError {
	topic = strings.TrimSpace(topic)
	if len(topic) == 0 || len(data) == 0 {
		WSCounterInc("input", "invalid_params")
		return errEmptyParameter
	}

	if len(data) > getDynamicConf().InputDataSize {
		WSCounterInc("input", "too_large_data_size")
		return errParamsErr
	}

	user := GetUserMgr().GetUser(sess.GetClient().GetMemberId())
	if user == nil {
		WSCounterInc("input", "not_auth")
		return errNotAuthorized
	}

	acceptorsByTopic := GetAcceptorMgr().GetByTopics([]string{topic})
	// appid->acceptor, 相同appid只需要选择一个节点
	acceptorMap := make(map[string][]Acceptor, len(acceptorsByTopic))
	for _, acc := range acceptorsByTopic {
		if checkUserShard(acc, user.GetMemberID()) {
			acceptorMap[acc.AppID()] = append(acceptorMap[acc.AppID()], acc)
		}
	}

	if len(acceptorMap) == 0 {
		glog.Debug(context.Background(), "onInput fail, no acceptor", glog.String("topic", topic))
		WSCounterInc("input", "no_acceptor")
		return newCodeErrFrom(errNoServiceByTopic, "topic: %v", topic)
	}

	uid := user.GetMemberID()
	acceptors := make([]Acceptor, 0, len(acceptorMap))
	for _, list := range acceptorMap {
		index := uid % int64(len(list))
		acceptors = append(acceptors, list[index])
	}

	ev := SyncInputEvent{ReqID: reqId, UserID: user.GetMemberID(), SessID: sess.ID(), Topic: topic, Data: data, Acceptors: acceptors}
	if err := DispatchEvent(&ev); err != nil {
		return errServiceNotAvailable
	}

	return nil
}

func onTrade(s Session, args []interface{}) ([]byte, map[string]string, error) {
	cli := s.GetClient()
	apikey := cli.GetAPIKey()
	if apikey == "" {
		glog.Debug(context.TODO(), "invalid apikey")
		return nil, nil, errDeniedAPIKey
	}
	if len(args) < 2 {
		return nil, nil, errEmptyParameter
	}
	var (
		route  string
		header = make(map[string]string, 5)
	)

	route = toString(args[0])
	data, ok := args[1].(map[string]interface{})
	if !ok {
		glog.Debug(context.TODO(), "body not json", glog.Any("body", args[1]))
		return nil, nil, errParamsErr
	}
	payload, err := jsonMarshal(data)
	if err != nil {
		glog.Debug(context.TODO(), "body JsonMarshal error", glog.Any("body", args[1]), glog.String("error", err.Error()))
		return nil, nil, errParamsErr
	}

	if len(args) >= 3 {
		headers, ok := args[2].(map[string]interface{})
		if !ok {
			glog.Debug(context.TODO(), "unmarshal metadata fail", glog.Any("headers", args[2]))
			return nil, nil, errParamsErr
		}
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				header[k] = vs
			}
		}
	}

	ctx := ctxBufferPool.Get()
	defer ctxBufferPool.Put(ctx)

	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	uri.SetPath(route)
	uri.SetHost(cli.GetHost())
	ctx.Request.SetURI(uri)
	ctx.Request.Header.SetMethod(http.MethodPost)
	for k, v := range header {
		ctx.Request.Header.Set(k, v)
	}
	ctx.Request.Header.Set(constant.HeaderAPIKey, apikey)
	ctx.Request.SetBody(payload)

	md := metadata.MDFromContext(ctx)
	defer metadata.Release(md)

	md.ReqInitTime = time.Now()
	md.UID = cli.GetMemberId()
	md.Method = http.MethodPost
	md.Path = route
	md.WssFlag = true
	md.BrokerID = cli.GetBrokerID()
	md.Extension.URI = uri.String()
	md.Extension.Host = cli.GetHost()
	md.Extension.UserAgent = cli.GetUserAgent()
	md.Extension.Referer = cli.GetReferer()
	md.Extension.RemoteIP = cli.GetIP()

	chain := filter.GlobalChain()
	chain, err = chain.AppendNames(filter.IPRateLimitFilterKey) // add default ip limiter
	if err != nil {
		return nil, nil, errServiceNotAvailable
	}
	f := func(ctx *types.Ctx) error {
		return tradeHandle(ctx)
	}
	ch := chain.Finally(f)
	if err := ch(ctx); err != nil {
		return nil, nil, err
	}

	header = make(map[string]string)
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = string(value)
	})

	return ctx.Response.Body(), header, nil
}

func tradeHandle(ctx *types.Ctx) error {
	ctrl := core.GetController(ctx)
	provider := core.NewCtxRouteDataProvider(ctx, nil, nil)
	handler, _ := ctrl.GetHandler(ctx, provider)
	if handler == nil {
		return berror.ErrRouteNotFound
	}

	return handler(ctx)
}

var (
	ctxBufferPool = ctxPool{
		sync.Pool{
			New: func() interface{} {
				return new(types.Ctx)
			},
		},
	}
)

type ctxPool struct {
	sync.Pool
}

func (c *ctxPool) Get() *types.Ctx {
	ctx, _ := c.Pool.Get().(*types.Ctx)
	return ctx
}

func (c *ctxPool) Put(ctx *types.Ctx) {
	if ctx == nil {
		return
	}
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctx.ResetUserValues()
	c.Pool.Put(ctx)
}
