package ws

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

const (
	optionErrSUCCESS = "0"
	// 参数异常,参数传入错误
	optionErrPARAMS_ERROR = "10001"
	// 没有对应指令的权限,未鉴权执行private操作或者账户权限受限
	// nolint
	optionErrNOPERMISSION = "3303001"
	// 鉴权失败,public/auth指令返回token认证失败
	optionErrAUTHFAIL = "3303002"
	// 在建立链接时带入token,但是token认证失败(连接会建立,但是会给客户端这个响应)
	// nolint
	optionErrLINKNOAUTH = "3303003"
	// 指令处理,响应未知异常
	optionErrCOMMANDHANDLE_UNKNOWERROR = "3303004"
	// 连接已经鉴权通过,在连接绑定鉴权后,重复发起鉴权
	optionErrAUTH_REPEAT = "3303005"
	// APIkey验签失败
	optionErrVERIFY_SIGN_FAIL = "3303006"
	// 指令调用频率限制
	optionErrREQ_COUNT_LIMIT = "3303007"
	// 非本 Zone 用户
	// nolint
	optionErrUID_ZONE_LIMIT = "10005"
	// 单uid建立的ws连接数限制
	// nolint
	optionErrUID_WS_CONNECT_LIMIT = "3303008"
)

const (
	optionRspTypeINCREMENT     = "INCREMENT"
	optionRspTypeCOMMAND_RESP  = "COMMAND_RESP"
	optionRspTypeAUTH_RESP     = "AUTH_RESP"
	optionRspTypeMergedMessage = "MergedMessage"
)

type requestOption struct {
	Id   string   `json:"id"`
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type responseOption struct {
	Type    string      `json:"type"`
	Id      string      `json:"id,omitempty"`
	ConnId  string      `json:"conn_id"`
	Success bool        `json:"success"`
	RetMsg  string      `json:"ret_msg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type responsePongOption struct {
	Id   string   `json:"id"`
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// https://confluence.yijin.io/pages/viewpage.action?pageId=42674799
type handlerOption struct {
}

func (h *handlerOption) Handle(ctx context.Context, sess Session, r io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			dumpPanic("handler_option panic", fmt.Errorf("%v", e))
		}
	}()

	req := &requestOption{}
	if err := readRequest(r, req); err != nil {
		// 编码错误,需要断开连接
		return err
	}

	if !sess.Allow() {
		rsp := &responseOption{
			Type:    optionRspTypeCOMMAND_RESP,
			Success: false,
			RetMsg:  optionErrREQ_COUNT_LIMIT,
		}
		_ = sendResponse(sess, rsp)
		WSCounterInc("handler_option", "session_limit")
		if getDynamicConf().Log.EnableSessionLimit {
			glog.Error(context.Background(), "session limit", glog.Any("req", req), glog.String("sessid", sess.ID()), glog.String("client", sess.GetClient().String()))
		}
		return nil
	}

	WSCounterInc("handler_"+sess.ProtocolVersion().String(), req.Op)

	if req.Op == opPing {
		now := time.Now().UnixNano() / 1e6
		rsp := &responsePongOption{
			Id:   req.Id,
			Op:   opPong,
			Args: []string{strconv.FormatInt(now, 10)},
		}

		return sendResponse(sess, rsp)
	}

	rsp := &responseOption{Id: req.Id, ConnId: sess.ID()}
	if data, err := h.Invoke(ctx, sess, req, rsp); err != nil {
		glog.Debug(ctx, "handle_option fail", glog.NamedError("err", err), glog.Any("req", req))
		cerr := toCodeErr(err)
		rsp.RetMsg = cerr.Error()
		rsp.Success = false
		WSCounterInc("handler_option", req.Op)
	} else {
		rsp.Success = true
		rsp.Data = data
	}

	return sendResponse(sess, rsp)
}

func (h *handlerOption) Invoke(ctx context.Context, sess Session, req *requestOption, rsp *responseOption) (interface{}, error) {
	args := req.Args
	switch req.Op {
	case opAuth:
		rsp.Type = optionRspTypeAUTH_RESP
		if len(args) != 3 {
			return nil, errParamsErr
		}
		apiKey := toString(args[0])
		expires := toInt64(args[1])
		signature := toString(args[2])
		err := onAuth(sess, apiKey, expires, signature)
		if err != nil {
			rsp.RetMsg = h.mappingErr(err)
			if isError(err, errRepeatedAuth) {
				// success 设置为true
				return nil, nil
			}

			return nil, err
		}

		rsp.RetMsg = optionErrSUCCESS
		return nil, nil
	case opLogin:
		rsp.Type = optionRspTypeAUTH_RESP
		if len(args) != 1 {
			return nil, errParamsErr
		}
		token := toString(args[0])
		err := onLogin(ctx, sess, token)
		if err != nil {
			rsp.RetMsg = h.mappingErr(err)
			if isError(err, errRepeatedAuth) {
				// success 设置为true
				return nil, nil
			}
			return nil, err
		}
		rsp.RetMsg = optionErrSUCCESS
		return nil, nil
	case opSubscribe:
		rsp.Type = optionRspTypeCOMMAND_RESP
		topics := args
		successes, fails, _, err := onSubscribe(sess, topics)
		if err != nil {
			return nil, err
		}

		if sess.ProtocolVersion() == versionOptionV1 {
			if fails == nil {
				fails = make([]string, 0)
			}
			return map[string]interface{}{
				"successTopics": successes,
				"failTopics":    fails,
			}, nil
		}
		return nil, err
	case opUnsubscribe:
		rsp.Type = optionRspTypeCOMMAND_RESP
		topics := args
		successes, fails, err := onUnsubscribe(sess, topics)
		if err != nil {
			return nil, err
		}

		if sess.ProtocolVersion() == versionOptionV1 {
			if fails == nil {
				fails = make([]string, 0)
			}
			return map[string]interface{}{
				"successTopics": successes,
				"failTopics":    fails,
			}, nil
		}
		return nil, nil
	default:
		rsp.Type = optionRspTypeCOMMAND_RESP
		return nil, errParamsErr
	}
}

func (h *handlerOption) mappingErr(err CodeError) string {
	switch err.Code() {
	case errRepeatedAuth.Code():
		return optionErrAUTH_REPEAT
	case errReqLimit.Code():
		return optionErrREQ_COUNT_LIMIT
	case errParamsErr.Code():
		return optionErrPARAMS_ERROR
	case errDeniedAPIKey.Code():
		return optionErrVERIFY_SIGN_FAIL
	case errAuthFail.Code():
		return optionErrAUTHFAIL
	default:
		return optionErrCOMMANDHANDLE_UNKNOWERROR
	}
}
