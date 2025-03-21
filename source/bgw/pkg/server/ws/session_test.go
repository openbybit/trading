package ws

import (
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/fasthttp/websocket"
	"github.com/stretchr/testify/assert"
)

func TestSession(t *testing.T) {
	c := NewClient(&ClientConfig{})
	s := newSession(nil, c, version2)
	assert.NotEmpty(t, s.ID())
	assert.NotEmpty(t, s.ShortID())
	assert.True(t, s.Allow())
	_ = s.SetMember(1)
	assert.True(t, s.IsAuthed())
	assert.EqualValues(t, version2, s.ProtocolVersion())
	assert.Equal(t, SessionStatus{WriteCount: 0, DropCount: 0, MaxIdleTime: sessionMaxIdleTime}, s.GetStatus())
	assert.Equal(t, false, s.IsRunning())

	msg := &Message{}
	s.conn = &websocket.Conn{}
	err := s.write(msg)
	assert.NotNil(t, err)
}

func TestSessionOnTick(t *testing.T) {
	c := NewClient(&ClientConfig{})
	s := newSession(nil, c, version2)
	s.OnTick()
	_ = s.SetMember(1)
	s.OnTick()
}

func TestParseMaxActiveTime(t *testing.T) {
	assert.Equal(t, sessionMaxIdleTime, parseMaxActiveTime(""))
	assert.Equal(t, sessionMinIdleTime, parseMaxActiveTime("1s"))
	assert.Equal(t, sessionMaxIdleTime, parseMaxActiveTime("20m"))
	assert.Equal(t, time.Minute, parseMaxActiveTime("1m"))
}

func TestSessionTick(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	getAppConf().AuthTickEnable = true
	old := sessionMaxAuthTime
	sessionMaxAuthTime = 1
	s1 := newSession(nil, NewClient(&ClientConfig{}), version2)
	s1.running = 1
	time.Sleep(time.Millisecond)
	s1.OnTick()
	assert.Truef(t, !s1.IsRunning(), "auth time")
	sessionMaxAuthTime = old

	s2 := newSession(nil, NewClient(&ClientConfig{}), version2)
	s2.running = 1
	s2.maxIdleTime = 1
	_ = s2.SetMember(1)
	time.Sleep(time.Millisecond)
	s2.OnTick()
	assert.Truef(t, !s2.IsRunning(), "idle time")
}
