package ws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/fasthttp/websocket"
)

var (
	sessionMaxAuthTime = time.Second * 60 // 最大等待auth时间,超过此时间没有Auth则会强制关闭连接
	sessionMaxIdleTime = time.Minute * 10 // 最大等待空闲时间,超过此时间没有收发消息则会强制关闭连接
	sessionMinIdleTime = time.Second * 30 // 最小时间
)

const (
	paramKeyMaxActiveTime = "max_active_time" // 空闲时间
)

var (
	errSessionClosed              = newCodeErr(1000, "session closed")
	errSessionWriteChannelDiscard = newCodeErr(1001, "session write channel discard")
)

var maxSessionID = uint64(0)

func newSessionID() (string, string) {
	id := atomic.AddUint64(&maxSessionID, 1)
	sid := strconv.FormatUint(id, 36)
	return fmt.Sprintf("%s-%s", globalNodeID, sid), sid
}

type SessionStatus struct {
	WriteCount  int64
	DropCount   int64
	MaxIdleTime time.Duration
}

// Session write message to client.
type Session interface {
	ID() string
	ShortID() string
	// Allow check command rate limit
	Allow() bool
	IsAuthed() bool
	SetMember(uid int64) error
	ProtocolVersion() versionType
	GetClient() Client
	GetStatus() SessionStatus
	GetStartTime() time.Time
	Write(msg *Message) error
	Stop()
	OnTick()
}

type session struct {
	conn        *WSConn
	id          string
	shortId     string
	version     versionType
	handler     Handler
	client      Client
	running     int32
	lastTime    int64
	sendCh      chan *Message
	closeCh     chan struct{}
	limit       rateLimit     //
	writeCount  int64         // 总发送量
	dropCount   int64         // 丢弃数量
	startTime   time.Time     // 起始时间
	maxIdleTime time.Duration // 超过此时间则会强制断开连接,误差取决于ticker的执行周期
}

func newSession(conn *WSConn, client Client, version versionType) *session {
	sconf := getDynamicConf()

	maxActiveTime := ""
	if client != nil {
		maxActiveTime = client.GetParams()[paramKeyMaxActiveTime]
	}

	id, shortId := newSessionID()
	s := &session{
		conn:        conn,
		id:          id,
		shortId:     shortId,
		version:     version,
		handler:     getHandler(version),
		sendCh:      make(chan *Message, sconf.SessionBufferSize),
		closeCh:     make(chan struct{}),
		running:     0,
		lastTime:    nowUnixNano(),
		client:      client,
		startTime:   time.Now(),
		maxIdleTime: parseMaxActiveTime(maxActiveTime),
	}
	s.limit.Set(int64(sconf.SessionCmdRateLimit), sconf.SessionCmdRatePeriod)

	return s
}

func parseMaxActiveTime(activeTime string) time.Duration {
	d, err := time.ParseDuration(activeTime)
	if err != nil {
		return sessionMaxIdleTime
	}

	if d < sessionMinIdleTime {
		d = sessionMinIdleTime
	}
	if d > sessionMaxIdleTime {
		d = sessionMaxIdleTime
	}

	return d
}

func (s *session) ID() string {
	return s.id
}

func (s *session) ShortID() string {
	return s.shortId
}

func (s *session) GetStatus() SessionStatus {
	return SessionStatus{
		WriteCount:  atomic.LoadInt64(&s.writeCount),
		DropCount:   atomic.LoadInt64(&s.dropCount),
		MaxIdleTime: s.maxIdleTime,
	}
}

func (s *session) GetStartTime() time.Time {
	return s.startTime
}

func (s *session) IsAuthed() bool {
	return s.client.GetMemberId() > 0
}

func (s *session) Allow() bool {
	return s.limit.Allow()
}

func (s *session) ProtocolVersion() versionType {
	return s.version
}

// SetMember set member id.
func (s *session) SetMember(uid int64) error {
	s.client.SetMemberId(uid)
	atomic.StoreInt64(&s.lastTime, nowUnixNano())
	return nil
}

func (s *session) GetClient() Client {
	return s.client
}

func (s *session) IsRunning() bool {
	return atomic.LoadInt32(&s.running) == 1
}

func (s *session) Run() {
	atomic.StoreInt32(&s.running, 1)
	go s.loopRecv()
	s.loopSend()
	s.Stop()
}

func (s *session) Write(msg *Message) error {
	if !s.IsRunning() {
		atomic.AddInt64(&s.dropCount, 1)
		WSCounterInc("session", "write_closed")
		return errSessionClosed
	}

	if msg.Type == MsgTypeReply {
		s.sendCh <- msg
		return nil
	}

	select {
	case s.sendCh <- msg:
		return nil
	default:
		atomic.AddInt64(&s.dropCount, 1)
		WSErrorInc("session", "write_discard")
		return errSessionWriteChannelDiscard
	}
}

func (s *session) Stop() {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		glog.Info(context.Background(), "session stop", glog.String("sess_id", s.id), glog.Int64("uid", s.client.GetMemberId()), glog.String("client", s.client.String()))
		s.client.Close()
		gSessionMgr.DelSession(s.id)
		gPublicMgr.OnSessionStop(s)
		close(s.closeCh)
		WSCounterInc("session", "stop")
	}
}

func (s *session) loopRecv() {
	defer func() {
		s.Stop()

		if err := recover(); err != nil {
			glog.Error(context.Background(), "session loop recv panic", glog.String("sess_id", s.id), glog.Int64("uid", s.client.GetMemberId()), glog.Any("error", err))
			WSErrorInc("session", "read_panic")
		}
	}()

	for s.IsRunning() {
		_, r, err := s.conn.NextReader()
		if err != nil {
			break
		}

		atomic.StoreInt64(&s.lastTime, nowUnixNano())

		err = s.handler.Handle(context.Background(), s, r)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, errSessionClosed) {
				glog.Debug(context.Background(), "session read eof", glog.String("sess_id", s.id), glog.Int64("uid", s.client.GetMemberId()), glog.String("error", err.Error()))
				break
			}

			WSErrorInc("session", "handle_fail")
			glog.Info(context.Background(), "session read error", glog.String("sess_id", s.id), glog.Int64("uid", s.client.GetMemberId()), glog.String("error", err.Error()))
		}
	}
}

func (s *session) loopSend() {
	for {
		select {
		case msg := <-s.sendCh:

			if err := s.write(msg); err != nil {
				return
			}
		case <-s.closeCh:
			return
		}
	}
}

func (s *session) write(msg *Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			glog.Error(context.Background(), "session write panic", glog.String("sess_id", s.id), glog.Int64("uid", s.client.GetMemberId()), glog.Any("error", r))
			WSErrorInc("session", "write_panic")
			err = errors.New("session write panic")
		}
	}()

	start := time.Now()
	startUnixNano := start.UnixNano()
	atomic.StoreInt64(&s.lastTime, startUnixNano)

	_ = s.conn.SetWriteDeadline(start.Add(time.Second))
	err = s.conn.WriteMessage(websocket.TextMessage, msg.Data)
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			WSCounterInc("session", "write_timeout")
			glog.Info(context.Background(), "session_write_timeout", glog.Int64("uid", s.client.GetMemberId()), glog.String("sess_id", s.id), glog.String("client", s.client.String()), glog.String("err", err.Error()))
		} else if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, websocket.ErrCloseSent) {
			WSCounterInc("session", "write_error")
			glog.Info(context.Background(), "session_write_error", glog.Int64("uid", s.client.GetMemberId()), glog.String("sess_id", s.id), glog.String("client", s.client.String()), glog.String("err", err.Error()))
		}
	}

	if msg.Type == MsgTypePush {
		s.writeCount++
	}

	wsDefaultLatencyE6(time.Duration(nowUnixNano()-startUnixNano), "session", "write")

	return err
}

func (s *session) OnTick() {
	if !s.IsAuthed() && getAppConf().AuthTickEnable {
		duration := time.Duration(nowUnixNano() - atomic.LoadInt64(&s.lastTime))
		if duration > sessionMaxAuthTime {
			glog.Info(context.Background(),
				"kick session by auth time",
				glog.String("sess_id", s.id),
				glog.Int64("uid", s.client.GetMemberId()),
				glog.String("client", s.client.String()),
				glog.String("duration", time.Duration(duration).String()),
				glog.String("start_time", s.startTime.Format(time.RFC3339)),
			)
			WSCounterInc("session", "kick_by_auth")
			s.Stop()
		}
	} else {
		duration := time.Duration(nowUnixNano() - atomic.LoadInt64(&s.lastTime))
		if duration > s.maxIdleTime {
			glog.Info(context.Background(),
				"kick session by idle time",
				glog.String("sess_id", s.id),
				glog.Int64("uid", s.client.GetMemberId()),
				glog.String("client", s.client.String()),
				glog.String("duration", time.Duration(duration).String()),
				glog.String("max_idle_time", s.maxIdleTime.String()),
				glog.String("start_time", s.startTime.Format(time.RFC3339)),
			)
			WSCounterInc("session", "kick_by_idle")
			s.Stop()
		}
	}
}
