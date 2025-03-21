package ws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"google.golang.org/grpc"
)

var (
	errAcceptorSendDiscard = errors.New("acceptor send message discard")
	errAcceptorClosed      = errors.New("acceptor closed")
)

var gReqPool = &sync.Pool{
	New: func() interface{} {
		return &envelopev1.SubscribeRequest{}
	},
}

func newSubscribeRequest() *envelopev1.SubscribeRequest {
	msg := gReqPool.Get().(*envelopev1.SubscribeRequest)
	msg.PushMessages = nil
	header := msg.Header
	msg.Reset()
	if header != nil {
		header.Reset()
		msg.Header = header
	} else {
		msg.Header = &envelopev1.Header{}
	}
	return msg
}

type Acceptor interface {
	ID() string
	AppID() string
	Topics() []string
	PublicTopics() []string
	UserShardIndex() int
	UserShardTotal() int
	FocusEvents() uint64           // 监听事件
	Address() string               // ip地址
	CreateTime() time.Time         // 创建时间
	Extensions() map[string]string // 扩展信息
	LastWriteFailTime() int64      // 最后一次写失败时间
	SendChannelRate() float64      // channel使用率

	Send(msg *envelopev1.SubscribeResponse) error
	// SendAdmin 发送admin消息,会阻塞等待返回结果,超时会返回timeout error
	SendAdmin(msg *envelopev1.SubscribeResponse) (*envelopev1.SubscribeRequest, error)
	Close()
}

type acceptorOptions struct {
	ShardIndex   int
	ShardTotal   int
	FocusEvents  int
	Address      string
	Extensions   map[string]string
	PublicTopics []string
}

func newAcceptor(stream grpc.ServerStream, connectorId, appId string, topics []string, opts *acceptorOptions) *acceptor {
	if opts == nil {
		opts = &acceptorOptions{}
	}

	sconf := getDynamicConf()

	sort.Strings(topics)
	sort.Strings(opts.PublicTopics)
	x := &acceptor{
		stream:         stream,
		createTime:     time.Now(),
		id:             connectorId,
		appId:          appId,
		topics:         topics,
		publicTopics:   opts.PublicTopics,
		userShardIndex: opts.ShardIndex,
		userShardTotal: opts.ShardTotal,
		focusEvents:    uint64(opts.FocusEvents),
		addr:           opts.Address,
		extensions:     opts.Extensions,
		sendCh:         make(chan *envelopev1.SubscribeResponse, sconf.AcceptorBufferSize),
		stopCh:         make(chan struct{}, 1),
		adminRsp:       make(map[string]*envelopev1.SubscribeRequest),
	}

	x.adminCnd = sync.NewCond(&x.adminMux)

	return x
}

type acceptor struct {
	stream            grpc.ServerStream
	createTime        time.Time
	appId             string
	id                string
	topics            []string
	userShardIndex    int
	userShardTotal    int
	focusEvents       uint64
	addr              string
	extensions        map[string]string // 扩展信息
	lastWriteFailTime atomic.Int64      // 最后一次写入管道失败时间
	publicTopics      []string          // 公有推送topic

	wg      sync.WaitGroup
	sendCh  chan *envelopev1.SubscribeResponse
	stopCh  chan struct{}
	running int32

	adminMux sync.RWMutex
	adminCnd *sync.Cond
	adminRsp map[string]*envelopev1.SubscribeRequest
}

func (a *acceptor) isRunning() bool {
	return atomic.LoadInt32(&a.running) == 1
}

func (a *acceptor) SendChannelRate() float64 {
	return 100.0 * float64(len(a.sendCh)) / float64(cap(a.sendCh))
}

func (a *acceptor) LastWriteFailTime() int64 {
	return a.lastWriteFailTime.Load()
}

func (a *acceptor) CreateTime() time.Time {
	return a.createTime
}

func (a *acceptor) ID() string {
	return a.id
}

func (a *acceptor) AppID() string {
	return a.appId
}

func (a *acceptor) Topics() []string {
	return a.topics
}

func (a *acceptor) PublicTopics() []string {
	return a.publicTopics
}

func (a *acceptor) UserShardIndex() int {
	return a.userShardIndex
}

func (a *acceptor) UserShardTotal() int {
	return a.userShardTotal
}

func (a *acceptor) FocusEvents() uint64 {
	return a.focusEvents
}

func (a *acceptor) Address() string {
	return a.addr
}

func (a *acceptor) Extensions() map[string]string {
	return a.extensions
}

func (a *acceptor) Start() {
	atomic.StoreInt32(&a.running, 1)

	a.wg.Add(2)
	go a.loopRecv()
	go a.loopSend()
}

func (a *acceptor) Close() {
	if atomic.CompareAndSwapInt32(&a.running, 1, 0) {
		close(a.stopCh)
	}
}

func (a *acceptor) Wait() {
	a.wg.Wait()
}

func (a *acceptor) Send(msg *envelopev1.SubscribeResponse) error {
	if !a.isRunning() {
		WSCounterInc("acceptor", "closed")
		return errAcceptorClosed
	}

	select {
	case a.sendCh <- msg:
		return nil
	default:
		glog.Errorf(context.TODO(), "acceptor send discard, appId=%s, connectorId=%s", a.appId, a.id)
		WSErrorInc("acceptor_discard", a.appId)
		a.lastWriteFailTime.Store(nowUnixNano())
		return errAcceptorSendDiscard
	}
}

func (a *acceptor) SendAdmin(req *envelopev1.SubscribeResponse) (rsp *envelopev1.SubscribeRequest, err error) {
	if req == nil || req.Admin == nil || req.Header == nil || req.Header.RequestId == "" {
		return nil, fmt.Errorf("invalid admin request")
	}

	doneCh := make(chan bool)

	go func() {
		rsp, err = a.doSendAdmin(req)
		if err != nil {
			glog.Errorf(context.Background(), "SendAdmin fail, type=%v, args=%v, err=%v", req.Admin.Type, req.Admin.Args, err)
		}
		doneCh <- true
	}()

	select {
	case <-doneCh:
		return
	case <-time.After(time.Millisecond * 600):
		glog.Errorf(context.Background(), "SendAdmin timeout, type=%v, args=%v, err=%v", req.Admin.Type, req.Admin.Args, err)
		a.adminCnd.Broadcast()
		err = fmt.Errorf("timeout %v, %v", req.Header.RequestId, req.Admin)
		return
	}
}

func (a *acceptor) doSendAdmin(req *envelopev1.SubscribeResponse) (*envelopev1.SubscribeRequest, error) {
	req.Cmd = envelopev1.Command_COMMAND_ADMIN

	glog.Infof(
		context.Background(),
		"ws: send_admin, appId=%v, acceptorId=%v, type=%v, args=%v",
		a.appId, a.id, req.Admin.Type, req.Admin.Args,
	)

	a.sendCh <- req
	// 阻塞等待数据返回
	start := time.Now()
	reqId := req.Header.RequestId
	for {
		a.adminMux.Lock()
		a.adminCnd.Wait()
		res := a.adminRsp[reqId]
		if res != nil {
			delete(a.adminRsp, reqId)
		}
		a.adminMux.Unlock()
		if res != nil {
			if res.Cmd == envelopev1.Command_COMMAND_ADMIN && res.Admin != nil && res.Admin.Type != envelopev1.Admin_TYPE_UNSPECIFIED {
				return res, nil
			}

			return nil, fmt.Errorf("invalid admin result, %v", res)
		}

		if time.Since(start) > time.Millisecond*500 {
			return nil, errors.New("timeout")
		}
	}
}

func (a *acceptor) loopRecv() {
	defer a.wg.Done()

	for {
		if atomic.LoadInt32(&a.running) == 0 {
			break
		}

		msg := newSubscribeRequest()
		if err := a.stream.RecvMsg(msg); err != nil {
			glog.Info(context.Background(), "acceptor recv msg fail", glog.String("id", a.id), glog.String("app_id", a.appId), glog.String("error", err.Error()))
			if !errors.Is(err, context.Canceled) {
				WSCounterInc("acceptor", "recv_error")
			}
			break
		}

		// 兼容老版本
		if msg.Cmd == envelopev1.Command_COMMAND_UNSPECIFIED && msg.Header != nil && msg.Header.Ack {
			msg.Cmd = envelopev1.Command_COMMAND_ACK
		}

		switch msg.Cmd {
		case envelopev1.Command_COMMAND_ADMIN:
			glog.Infof(context.Background(), "ws: recv_admin, appId=%v, acceptorId=%v, rsp=%v", a.appId, a.id, msg.Admin)
			if msg.Header != nil && msg.Header.RequestId != "" {
				a.adminMux.Lock()
				a.adminRsp[msg.Header.RequestId] = msg
				a.adminMux.Unlock()
			}
			// 唤醒一次所有阻塞的admin
			a.adminCnd.Broadcast()
		case envelopev1.Command_COMMAND_ACK:
			gReqPool.Put(msg)
		default:
			DispatchMessage(a, msg)
			gReqPool.Put(msg)
		}

	}

	a.onStop()
}

func (a *acceptor) loopSend() {
	defer a.wg.Done()

Loop:
	for {
		select {
		case v := <-a.sendCh:
			since := time.Now()
			if err := a.stream.SendMsg(v); err != nil {
				glog.Info(context.Background(), "acceptor send msg fail", glog.String("id", a.id), glog.String("app_id", a.appId), glog.String("error", err.Error()))
				if !errors.Is(err, context.Canceled) {
					WSCounterInc("acceptor", "send_error")
				}
				break Loop
			}
			WSHistogram(since, "acceptor", "send")
		case <-a.stopCh:
			glog.Info(context.Background(), "acceptor send stop", glog.String("id", a.id), glog.String("app_id", a.appId))
			WSCounterInc("acceptor", "stop")
			break Loop
		}
	}

	a.sendLeftMessage()
	a.onStop()
}

func (a *acceptor) sendLeftMessage() {
	sconf := getDynamicConf()
	if !sconf.EnableGracefulClose {
		glog.Info(context.Background(), "disable graceful close acceptor", glog.String("id", a.id))
		return
	}

	leftMsgSize := len(a.sendCh)
	if leftMsgSize == 0 {
		return
	}
	glog.Info(context.Background(), "send the remaining message before closing acceptor.", glog.String("id", a.id), glog.String("app_id", a.appId), glog.Int("size", leftMsgSize))
	for {
		select {
		case v := <-a.sendCh:
			if err := a.stream.SendMsg(v); err != nil {
				glog.Error(context.Background(), "stop to send remainning message", glog.String("id", a.id), glog.String("app_id", a.appId), glog.Int("size", leftMsgSize), glog.String("error", err.Error()))
				return
			}
		default:
			glog.Info(context.Background(), "send the remaining message finished.", glog.String("id", a.id), glog.String("app_id", a.appId), glog.Int("size", leftMsgSize))
			return
		}
	}
}

func (a *acceptor) onStop() {
	a.Close()
}
