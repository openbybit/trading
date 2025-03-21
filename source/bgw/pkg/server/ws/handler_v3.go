package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

type requestV3 struct {
	ReqID string        `json:"req_id"`
	Op    string        `json:"op"`
	Args  []interface{} `json:"args"`
}

type responseV3 struct {
	ReqID   string      `json:"req_id,omitempty"`
	Success bool        `json:"success"`
	RetMsg  string      `json:"ret_msg"`
	OP      string      `json:"op"`
	Args    interface{} `json:"args,omitempty"` //
	ConnID  string      `json:"conn_id,omitempty"`
}

type responsePongV3 struct {
	ReqID  string   `json:"req_id,omitempty"`
	Op     string   `json:"op,omitempty"`
	Args   []string `json:"args,omitempty"`
	ConnID string   `json:"conn_id,omitempty"`
}

type responseTradeV3 struct {
	ReqID   string          `json:"req_id,omitempty"`
	Success bool            `json:"success"`
	RetMsg  string          `json:"ret_msg"`
	OP      string          `json:"op"`
	Args    interface{}     `json:"args,omitempty"` //
	Body    json.RawMessage `json:"body,omitempty"`
	Header  interface{}     `json:"header,omitempty"`
	ConnID  string          `json:"conn_id,omitempty"`
}

// https://c1ey4wdv9g.larksuite.com/wiki/wikusCQyGH75qjb0KxqDrl1NLvb?sheet=11WwLo
type handlerV3 struct {
}

func (h *handlerV3) Handle(ctx context.Context, sess Session, r io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			dumpPanic("handler_v3 panic", fmt.Errorf("%v", e))
		}
	}()

	req := &requestV3{}

	if err = readRequest(r, req); err != nil {
		WSCounterInc("handler_v3", "read_fail")
		return h.sendResponseV3(sess, req, err)
	}

	if !sess.Allow() {
		WSCounterInc("handler_v3", "session_limit")
		return h.sendResponseV3(sess, req, errReqLimit)
	}

	return h.Invoke(ctx, sess, req)
}

func (h *handlerV3) Invoke(ctx context.Context, sess Session, req *requestV3) error {
	WSCounterInc("handler_v3", req.Op)
	args := req.Args
	switch req.Op {
	case opTrade:
		body, header, err := onTrade(sess, args)
		rsp := &responseTradeV3{
			OP:     opTrade,
			ReqID:  req.ReqID,
			ConnID: sess.ID(),
		}
		if err == nil {
			rsp.Success = true
			rsp.Body = body
			rsp.Header = header
		} else {
			rsp.Success = false
			rsp.RetMsg = err.Error()
		}
		return sendResponse(sess, rsp)
	case opPing:
		now := time.Now().UnixNano() / 1e6
		pongRsp := &responsePongV3{
			Args: []string{strconv.FormatInt(now, 10)},
		}
		pongRsp.Op = opPong
		pongRsp.ReqID = req.ReqID
		pongRsp.ConnID = sess.ID()
		return sendResponse(sess, pongRsp)
	case opAuth:
		if len(args) != 3 {
			return h.sendResponseV3(sess, req, errParamsErr)
		}
		apiKey := toString(args[0])
		expires := toInt64(args[1])
		signature := toString(args[2])
		err := onAuth(sess, apiKey, expires, signature)
		return h.sendResponseV3(sess, req, err)
	case opLogin:
		if len(args) != 1 {
			return h.sendResponseV3(sess, req, errParamsErr)
		}
		token := toString(args[0])
		err := onLogin(ctx, sess, token)
		return h.sendResponseV3(sess, req, err)
	case opSubscribe:
		topics := toStringList(args)
		_, _, changed, err := onSubscribe(sess, topics)
		sendErr := h.sendResponseV3(sess, req, err)
		gPublicMgr.OnSubscribe(sess, changed)
		return sendErr
	case opUnsubscribe:
		topics := toStringList(args)
		_, _, err := onUnsubscribe(sess, topics)
		return h.sendResponseV3(sess, req, err)
	case opInput:
		if len(req.Args) < 2 {
			return h.sendResponseV3(sess, req, errParamsErr)
		}
		err := onInput(sess, req.ReqID, toString(args[0]), toString(args[1]))
		return h.sendResponseV3(sess, req, err)
	default:
		return h.sendResponseV3(sess, req, errParamsErr)
	}
}

func (h *handlerV3) sendResponseV3(sess Session, req *requestV3, err error) error {
	rsp := &responseV3{
		OP:     req.Op,
		ReqID:  req.ReqID,
		ConnID: sess.ID(),
	}

	if err != nil {
		glog.Debug(context.Background(), "handle_v3 fail", glog.NamedError("err", err), glog.Any("req", req))
		WSCounterInc("handler_v3", req.Op)
		cerr := toCodeErr(err)
		rsp.RetMsg = cerr.Error()
		rsp.Success = false
	} else {
		rsp.Success = true
	}

	return sendResponse(sess, rsp)
}
