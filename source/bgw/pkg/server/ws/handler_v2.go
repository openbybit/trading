package ws

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

type requestV2 struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type responseV2 struct {
	Success bool        `json:"success"`
	Message string      `json:"ret_msg"`
	Request *requestV2  `json:"request"`
	Data    interface{} `json:"data"`
	ConnId  string      `json:"conn_id"`
}

type handlerV2 struct {
}

func (h *handlerV2) Handle(ctx context.Context, sess Session, r io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			dumpPanic("handler_v2 panic", fmt.Errorf("%v", e))
		}
	}()

	req := &requestV2{}
	rsp := &responseV2{}
	if err = readRequest(r, req); err != nil {
		WSCounterInc("handler_v2", "read_fail")
		return h.sendResponseV2(sess, req, rsp, err)
	}

	if !sess.Allow() {
		WSCounterInc("handler_v2", "session_limit")
		return h.sendResponseV2(sess, req, rsp, errReqLimit)
	}

	return h.Invoke(ctx, sess, req, rsp)
}

func (h *handlerV2) Invoke(ctx context.Context, sess Session, req *requestV2, rsp *responseV2) error {
	WSCounterInc("handler_v2", req.Op)
	switch req.Op {
	case opTest:
		rsp.Data = []byte(time.Now().String())
		return h.sendResponseV2(sess, req, rsp, nil)
	case opPing:
		rsp.Message = opPong
		return h.sendResponseV2(sess, req, rsp, nil)
	case opLogin:
		if len(req.Args) == 0 {
			return h.sendResponseV2(sess, req, rsp, errParamsErr)
		}
		err := onLogin(ctx, sess, req.Args[0])
		return h.sendResponseV2(sess, req, rsp, err)
	case opSubscribe:
		topics := h.parseTopics(req.Args)
		successes, fails, changed, err := onSubscribe(sess, topics)
		rsp.Data = map[string]interface{}{
			"successes": successes,
			"fails":     fails,
		}
		sendErr := h.sendResponseV2(sess, req, rsp, err)
		gPublicMgr.OnSubscribe(sess, changed)
		return sendErr
	case opUnsubscribe:
		topics := h.parseTopics(req.Args)
		_, _, err := onUnsubscribe(sess, topics)
		return h.sendResponseV2(sess, req, rsp, err)
	case opInput:
		if len(req.Args) < 2 {
			return h.sendResponseV2(sess, req, rsp, errParamsErr)
		}
		err := onInput(sess, "", toString(req.Args[0]), toString(req.Args[1]))
		return h.sendResponseV2(sess, req, rsp, err)
	default:
		return h.sendResponseV2(sess, req, rsp, errParamsErr)
	}
}

func (h *handlerV2) parseTopics(args []string) []string {
	topics := make([]string, 0, len(args))
	for i := range args {
		if t := strings.TrimSpace(args[i]); t == "" {
			continue
		} else {
			topics = append(topics, t)
		}
	}

	return topics
}

func (h *handlerV2) sendResponseV2(sess Session, req *requestV2, rsp *responseV2, err error) error {
	if err != nil {
		glog.Debug(context.Background(), "handle_v2 fail", glog.NamedError("err", err), glog.Any("req", req))
		WSCounterInc("handler_v2", req.Op)
		rsp.Success = false
		rsp.Message = err.Error()
	} else {
		rsp.Success = true
	}

	rsp.Request = req
	rsp.ConnId = sess.ID()
	return sendResponse(sess, rsp)
}
