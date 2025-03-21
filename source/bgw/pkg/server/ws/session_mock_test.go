package ws

import "time"

func newMockSession(uid int64) *MockSession {
	id, shortID := newSessionID()
	cli := NewClient(&ClientConfig{IP: "127.0.0.1:8080"})
	if uid > 0 {
		cli.SetMemberId(uid)
	}
	return &MockSession{
		id:        id,
		shortID:   shortID,
		startTime: time.Now(),
		client:    cli,
		running:   true,
	}
}

type MockSession struct {
	id        string
	shortID   string
	client    Client
	startTime time.Time
	lastMsg   *Message
	running   bool
	chanFull  bool // 模拟通道已经满了
}

func (s *MockSession) ID() string {
	return s.id
}

func (s *MockSession) ShortID() string {
	return s.shortID
}

func (s *MockSession) Allow() bool {
	return true
}

func (s *MockSession) IsAuthed() bool {
	return s.client.GetMemberId() > 0
}

func (s *MockSession) SetMember(uid int64) error {
	s.client.SetMemberId(uid)
	return nil
}

func (s *MockSession) ProtocolVersion() versionType {
	return version2
}

func (s *MockSession) GetClient() Client {
	return s.client
}

func (s *MockSession) GetStatus() SessionStatus {
	return SessionStatus{}
}

func (s *MockSession) GetStartTime() time.Time {
	return s.startTime
}

func (s *MockSession) Write(msg *Message) error {
	if !s.running {
		return errSessionClosed
	}

	if s.chanFull {
		return errSessionWriteChannelDiscard
	}

	s.lastMsg = msg
	return nil
}

func (s *MockSession) Stop() {
	s.running = false
}

func (s *MockSession) OnTick() {
}

func (s *MockSession) SetChanFull(v bool) {
	s.chanFull = v
}
