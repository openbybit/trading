package ws

// MsgType is message type.
type MsgType int8

const (
	// MsgTypeReply is reply message.
	MsgTypeReply MsgType = 1
	// MsgTypePush is push message.
	MsgTypePush MsgType = 2
)

// Message is a websocket message.
type Message struct {
	Type MsgType
	Data []byte
}
