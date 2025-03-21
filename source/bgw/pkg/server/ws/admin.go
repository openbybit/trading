package ws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"

	"bgw/pkg/service/masque"
)

var (
	errInvalidID  = errors.New("invalid id")
	errNotFoundID = errors.New("cannot find id")
)

func setupAdmin() {
	// curl http://127.0.0.1:6480/admin?test
	gapp.RegisterAdmin("test", "测试", onAdminTest)
	gapp.RegisterAdmin("acceptor_list", "查询SDK连接,参数:[id]", onAdminGetAcceptorList)
	gapp.RegisterAdmin("acceptor_status", "查询SDK状态, 参数:[id]", onAdminGetAcceptorStatus)
	gapp.RegisterAdmin("acceptor_dump", "dump SDK端 数据, 参数: id", onAdminAcceptorDump)
	gapp.RegisterAdmin("metrics", "查看SDK业务端埋点", onAdminMetrics)
	gapp.RegisterAdmin("user_info", "查看用户数据, 参数: uid", onAdminUserInfo)
	gapp.RegisterAdmin("sync_user", "同步用户数据, 参数: uid", onAdminSyncUser)
	gapp.RegisterAdmin("session_per_user", "查看用户session连接数排行", onAdminSessionPerUsr)
	gapp.RegisterAdmin("config", "查看配置", onAdminConfig)
	gapp.RegisterAdmin("check_gray", "判断某个uid是否是灰度shard(internal/qianqian/option)", onAdminCheckGray)
	gapp.RegisterAdmin("verify_login", "验证登录是否正常", onAdminVerifyLogin)
	gapp.RegisterAdmin("verify_auth", "验证Auth是否正常", onAdminVerifyAuth)
	gapp.RegisterAdmin("kick_user", "踢出用户", onAdminKickUser)
	gapp.RegisterAdmin("debug", "调试", onAdminDebug)
	gapp.RegisterAdmin("public_info", "公有推送信息", onAdminPublicInfo)
	gapp.RegisterAdmin("get_all_topics", "获取topic信息", onAdminGetTopics)
}

func newAdminReq(typ envelopev1.Admin_Type, args string) *envelopev1.SubscribeResponse {
	msg := envelopev1.SubscribeResponse{
		Header: &envelopev1.Header{
			RequestId: newUUID(),
		},
		Cmd: envelopev1.Command_COMMAND_ADMIN,
		Admin: &envelopev1.Admin{
			Type: typ,
			Args: args,
		},
	}
	return &msg
}

func onAdminTest(args gapp.AdminArgs) (interface{}, error) {
	return map[string]interface{}{
		"now": time.Now().String(),
	}, nil
}

// 获取acceptors列表信息
// acceptor_list [app_id]
func onAdminGetAcceptorList(args gapp.AdminArgs) (interface{}, error) {
	appID := args.GetStringBy("appid")
	if appID == "" && args.ParamSize() > 0 {
		appID = args.GetStringAt(0)
	}

	accID := args.GetStringBy("id")

	var acceptors []Acceptor
	if appID != "" {
		acceptors = GetAcceptorMgr().GetByAppID(appID)
	} else if accID != "" {
		acc := GetAcceptorMgr().Get(accID)
		if acc != nil {
			acceptors = append(acceptors, acc)
		}
	} else {
		acceptors = GetAcceptorMgr().GetAll()
	}

	sort.Slice(acceptors, func(i, j int) bool {
		a1 := acceptors[i]
		a2 := acceptors[j]
		if a1.AppID() == a2.AppID() {
			return a1.UserShardIndex() < a2.UserShardIndex()
		}

		return a1.AppID() < a2.AppID()
	})

	type AcceptorInfo struct {
		AppID       string   `json:"app_id"`               //
		Topics      string   `json:"topics"`               //
		ShardTotal  int      `json:"shard_total"`          //
		FocusEvents int      `json:"focus_events"`         //
		Extensions  string   `json:"extensions,omitempty"` // 扩展信息
		Nodes       []string `json:"nodes,omitempty"`      // 节点信息: id/ip/shardIndex/createTime/{diff}
	}

	res := make(map[string]*AcceptorInfo)
	for _, acc := range acceptors {
		ext := encodeMapToString(acc.Extensions())
		info := res[acc.AppID()]
		topics := acc.Topics()
		topics = append(topics, acc.PublicTopics()...)
		if info == nil {
			info = &AcceptorInfo{
				AppID:       acc.AppID(),
				Topics:      strings.Join(topics, ","),
				ShardTotal:  acc.UserShardTotal(),
				FocusEvents: int(acc.FocusEvents()),
				Extensions:  ext,
			}
			res[acc.AppID()] = info
		}

		// build different node info
		b := bytes.NewBuffer(nil)
		if acc.UserShardTotal() != info.ShardTotal {
			writeKvToBuffer(b, "shard_total", acc.UserShardTotal(), '=')
		}
		if acc.FocusEvents() != uint64(info.FocusEvents) {
			writeKvToBuffer(b, "focus_events", acc.FocusEvents(), '=')
		}
		if ext != info.Extensions {
			b.WriteString(ext)
		}

		createTime := acc.CreateTime().Format(time.RFC3339)
		node := fmt.Sprintf("%s/%s/%d/%s/%s", acc.ID(), acc.Address(), acc.UserShardIndex(), createTime, b.String())
		info.Nodes = append(info.Nodes, node)
	}

	return res, nil
}

// writeKvToBuffer 辅助方法
func writeKvToBuffer(b *bytes.Buffer, k string, value interface{}, sep byte) {
	b.WriteString(k)
	b.WriteByte(sep)
	b.WriteString(fmt.Sprint(value))
	b.WriteByte(' ')
}

// onAdminGetAcceptorStatus 查看状态
func onAdminGetAcceptorStatus(args gapp.AdminArgs) (interface{}, error) {
	if args.ParamSize() == 0 {
		return nil, errInvalidID
	}
	id := args.GetStringAt(0)
	acceptors := getAcceptorsByAnyID(id)
	if len(acceptors) == 0 {
		return nil, fmt.Errorf("%w, %v", errNotFoundID, id)
	}

	type Item struct {
		ID     string            `json:"id,omitempty"`
		Status map[string]string `json:"status,omitempty"`
		Error  string            `json:"error,omitempty"`
	}

	type Result struct {
		Items []Item `json:"items"`
	}

	res := &Result{}

	for _, acc := range acceptors {
		req := newAdminReq(envelopev1.Admin_TYPE_STATUS, "")
		rsp, err := acc.SendAdmin(req)
		item := Item{
			ID: acc.ID(),
		}
		if err == nil {
			item.Status = rsp.Admin.Result.Status
		} else {
			item.Error = err.Error()
		}

		res.Items = append(res.Items, item)
	}

	return res, nil
}

// onAdminAcceptorDump 通知打印日志
func onAdminAcceptorDump(args gapp.AdminArgs) (interface{}, error) {
	if args.ParamSize() == 0 {
		return nil, errInvalidID
	}

	id := args.GetStringAt(0)
	acceptors := getAcceptorsByAnyID(id)
	if len(acceptors) == 0 {
		return nil, fmt.Errorf("%w, %v", errNotFoundID, id)
	}

	type Result struct {
		Success int `json:"success"`
		Failure int `json:"failure"`
	}

	res := &Result{}

	for _, acc := range acceptors {
		req := newAdminReq(envelopev1.Admin_TYPE_STATUS, "")
		_, err := acc.SendAdmin(req)
		if err != nil {
			res.Failure++
		} else {
			res.Success++
		}
	}

	return res, nil
}

// metrics connector_id
func onAdminMetrics(args gapp.AdminArgs) (interface{}, error) {
	if args.ParamSize() == 0 {
		return nil, errInvalidID
	}
	id := args.GetStringAt(0)
	acc := GetAcceptorMgr().Get(id)
	if acc == nil {
		return nil, fmt.Errorf("%w, %v", errNotFoundID, id)
	}
	req := newAdminReq(envelopev1.Admin_TYPE_METRIC, "")
	rsp, err := acc.SendAdmin(req)
	if err != nil {
		return nil, fmt.Errorf("get metrics fail, err=%v", err)
	}

	return rsp.Admin.Result.Metrics, nil
}

// user_info uid [connector_id]
func onAdminUserInfo(args gapp.AdminArgs) (interface{}, error) {
	if args.ParamSize() == 0 {
		type Result struct {
			UserCount    int `json:"user_count"`
			SessionCount int `json:"session_count"`
		}

		res := &Result{}
		res.UserCount = GetUserMgr().Size()
		res.SessionCount = GetSessionMgr().Size()

		return res, nil
	}

	// 支持一次性查询多个uid
	uidStr := args.GetStringAt(0)
	if uidStr == "" {
		return nil, errInvalidID
	}

	uidList := strings.Split(uidStr, ",")

	var acceptors []Acceptor
	if args.ParamSize() > 1 {
		accId := args.GetStringAt(1)
		acceptors = getAcceptorsByAnyID(accId)
	}

	type ConnectorInfo struct {
		ConnectorID string           `json:"connector_id"`
		User        *envelopev1.User `json:"user,omitempty"`
		Remote      *envelopev1.User `json:"remote,omitempty"`
	}

	type SessionInfo struct {
		ID          string    `json:"id"`            // session_id
		WriteCount  int64     `json:"write_count"`   // 总发送次数
		DropCount   int64     `json:"drop_count"`    // 丢弃数量
		MaxIdleTime string    `json:"max_idle_time"` // 最大空闲时间
		StartTime   time.Time `json:"start_time"`    // 起始时间
		Duration    string    `json:"duration"`      // 连接时长
		Version     string    `json:"version"`       // 协议版本
		Path        string    `json:"path"`          // 连接路径
		IP          string    `json:"ip"`            // 客户端地址
		Topics      string    `json:"topics"`        // 订阅topics
	}

	type UserInfo struct {
		UserID     int64            `json:"user_id"`              // 用户ID
		User       *envelopev1.User `json:"user,omitempty"`       // 聚合后用户数据
		Sessions   []SessionInfo    `json:"sessions,omitempty"`   // 每个连接信息
		Connectors []*ConnectorInfo `json:"connectors,omitempty"` // 对应SDK信息
	}

	results := make([]*UserInfo, 0, len(uidList))

	for _, uidStr := range uidList {
		uidStr = strings.TrimSpace(uidStr)
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil {
			continue
		}

		item := &UserInfo{UserID: uid}

		user := GetUserMgr().GetUser(uid)
		if user != nil {
			item.User = user.ToMessageUser()
			sessions := user.GetSessions()
			for _, s := range sessions {
				client := s.GetClient()
				st := s.GetStartTime()
				status := s.GetStatus()
				session := SessionInfo{
					ID:          s.ID(),
					WriteCount:  status.WriteCount,
					DropCount:   status.DropCount,
					MaxIdleTime: status.MaxIdleTime.String(),
					StartTime:   st,
					Duration:    time.Since(st).String(),
					Version:     s.ProtocolVersion().String(),
					Path:        client.GetPath(),
					IP:          client.GetIP(),
					Topics:      strings.Join(client.GetTopics().Values(), ","),
				}
				item.Sessions = append(item.Sessions, session)
			}
		}

		if len(acceptors) > 0 {
			for _, acc := range acceptors {
				req := newAdminReq(envelopev1.Admin_TYPE_USER_INFO, uidStr)
				req.Admin.Args = uidStr
				rsp, err := acc.SendAdmin(req)
				if err != nil {
					return nil, fmt.Errorf("get user fail, %v", err)
				}

				result := rsp.Admin.Result
				item.Connectors = append(item.Connectors, &ConnectorInfo{
					ConnectorID: acc.ID(),
					User:        result.User,
					Remote:      result.RemoteUser,
				})
			}
		}
		results = append(results, item)
	}

	return results, nil
}

// 查询服务器的config或connector的config config [connector_id]
func onAdminConfig(args gapp.AdminArgs) (interface{}, error) {
	typ := args.GetStringBy("type")
	if typ == "sdk" {
		connectorId := ""
		if args.ParamSize() > 0 {
			connectorId = args.GetStringAt(0)
		} else {
			connectorId = args.GetStringBy("id")
		}
		if connectorId != "" {
			acc := GetAcceptorMgr().Get(connectorId)
			if acc == nil {
				return nil, fmt.Errorf("cannot find acceptor, %v", connectorId)
			}
			req := newAdminReq(envelopev1.Admin_TYPE_CONFIG, "")
			rsp, err := acc.SendAdmin(req)
			if err != nil {
				return nil, fmt.Errorf("get config fail, err=%v", err)
			}

			return rsp.Admin.Result.Config, nil
		} else {
			conf := gConfigMgr.GetSdkConf()
			if conf == nil {
				return nil, fmt.Errorf("no config")
			}
			return conf, nil
		}
	} else {
		conf := getDynamicConf()
		return conf, nil
	}
}

// onAdminSyncUser 同步用户数据， sync_user [app_id or connector_id]
func onAdminSyncUser(args gapp.AdminArgs) (interface{}, error) {
	id := args.GetStringBy("id")
	if id == "" && args.ParamSize() > 0 {
		id = args.GetStringAt(0)
	}

	if id == "" {
		return nil, errInvalidID
	}

	acceptors := getAcceptorsByAnyID(id)

	if len(acceptors) == 0 {
		return nil, fmt.Errorf("no acceptors available")
	}

	for _, acc := range acceptors {
		_ = DispatchEvent(NewSyncAllUserEvent(acc.ID()))
	}

	return nil, nil
}

// onAdminSessionPerUsr 用于查看机构的用户最大连接数
func onAdminSessionPerUsr(args gapp.AdminArgs) (interface{}, error) {
	type Result struct {
		ApiKey       string `json:"api_key,omitempty"`
		SessionCount int64  `json:"session_count,omitempty"`
		User         int64  `json:"uid,omitempty"`
	}

	apiKeyHelper := make(map[string]int64)
	uidHelper := make(map[string]int64)
	mgr := GetUserMgr()
	if mgr == nil {
		return nil, nil
	}
	for _, v := range mgr.GetAllUsers() {
		for _, s := range v.GetSessions() {
			cli := s.GetClient()
			if cli == nil {
				continue
			}
			key := cli.GetAPIKey()
			if key == "" {
				continue
			}

			if _, ok := apiKeyHelper[key]; ok {
				apiKeyHelper[key] += 1
			} else {
				apiKeyHelper[key] = 1
				uidHelper[key] = v.GetMemberID()
			}
		}
	}

	if len(apiKeyHelper) == 0 {
		return apiKeyHelper, nil
	}

	var res []Result

	for k, v := range apiKeyHelper {
		res = append(res, Result{
			ApiKey:       k,
			SessionCount: v,
			User:         uidHelper[k],
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].SessionCount > res[j].SessionCount
	})
	return res, nil
}

func getAcceptorsByAnyID(id string) []Acceptor {
	acc := GetAcceptorMgr().Get(id)
	if acc != nil {
		return []Acceptor{acc}
	} else {
		return GetAcceptorMgr().GetByAppID(id)
	}
}

// onAdminCheckGray 判断某个uid是否是灰度shard
func onAdminCheckGray(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	if uid == 0 {
		uid = args.GetInt64By("uid")
	}

	if uid == 0 {
		return nil, fmt.Errorf("empty uid")
	}

	isGray := getDynamicConf().IsGray(uid)

	type result struct {
		IsGray bool `json:"is_gray"`
	}

	return &result{IsGray: isGray}, nil
}

func onAdminVerifyLogin(args gapp.AdminArgs) (interface{}, error) {
	token := args.GetStringAt(0)
	if token == "" {
		token = args.GetStringBy("token")
	}
	if token == "" {
		return nil, fmt.Errorf("invalid token")
	}

	m, err := getMasqueService()
	if err != nil {
		return nil, fmt.Errorf("invalid masque service")
	}

	resp, err := m.MasqueTokenInvoke(context.Background(), "", token, "", masque.WeakAuth)
	result := map[string]interface{}{
		"response": resp,
		"error":    err,
	}

	return result, nil
}

func onAdminVerifyAuth(args gapp.AdminArgs) (interface{}, error) {
	apiKey := args.GetStringAt(0)
	if apiKey == "" {
		apiKey = args.GetStringBy("api_key")
	}

	o, err := getUserService()
	if err != nil || isNil(o) {
		return nil, fmt.Errorf("invalid openapi service")
	}

	// check apikey
	member, err := o.VerifyAPIKey(context.TODO(), apiKey, "")
	result := map[string]interface{}{
		"result": member,
		"error":  err,
	}

	return result, nil
}

// onAdminKickUser 根据uid踢出用户
func onAdminKickUser(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	if uid == 0 {
		uid = args.GetInt64By("uid")
	}

	if uid == 0 {
		return nil, fmt.Errorf("invalid uid")
	}

	user := GetUserMgr().GetUser(uid)
	sessions := user.GetSessions()
	for _, sess := range sessions {
		sess.Stop()
	}

	return map[string]interface{}{"uid": uid, "size": len(sessions)}, nil
}

func onAdminDebug(args gapp.AdminArgs) (interface{}, error) {
	mode := args.GetStringBy("mode")
	switch mode {
	case "log":
		now := nowUnixNano()
		gMetricsMgr.Send(&metricsMessage{
			Type:          metricsTypePush,
			AppID:         "test-appid",
			RemoteAddr:    "127.0.0.1",
			WsStartTimeE9: now,
			WsEndTimeE9:   now,
			Push: &envelopev1.PushMessage{
				UserId:        1,
				Topic:         "test-topic",
				TraceId:       "",
				MessageId:     "test-msgid",
				RequestTimeE9: now,
				InitTimeE9:    now,
				SdkTimeE9:     now,
				Data:          []byte("test-data"),
			},
		})
	}

	return "ok", nil
}

// onAdminPublicInfo 获取公有推送信息
func onAdminPublicInfo(args gapp.AdminArgs) (interface{}, error) {
	return gPublicMgr.Info(), nil
}

func onAdminGetTopics(args gapp.AdminArgs) (interface{}, error) {
	return gConfigMgr.GetAllTopics(), nil
}
