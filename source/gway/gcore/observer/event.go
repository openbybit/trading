package observer

import (
	"fmt"
	"io"
	"time"
)

// EventType means SourceObjectEventType
type EventType int

const (
	// EventTypeAdd means add event
	EventTypeAdd = iota
	// EventTypeDel means del event
	EventTypeDel
	// EventTypeUpdate means update event
	EventTypeUpdate
)

// Event is the interface that wraps the basic Event methods.
type Event interface {
	fmt.Stringer
	GetSource() interface{}
	GetTimestamp() time.Time
}

// BaseEvent is the base implementation of Event
// You should never use it directly
type BaseEvent struct {
	Source    interface{}
	Timestamp time.Time
}

// GetSource return the source
func (b *BaseEvent) GetSource() interface{} {
	return b.Source
}

// GetTimestamp return the Timestamp when the event is created
func (b *BaseEvent) GetTimestamp() time.Time {
	return b.Timestamp
}

// String return a human readable string representing this event
func (b *BaseEvent) String() string {
	return fmt.Sprintf("BaseEvent[source = %#v]", b.Source)
}

// NewBaseEvent create an BaseEvent instance
// and the Timestamp will be current timestamp
func NewBaseEvent(source interface{}) *BaseEvent {
	return &BaseEvent{
		Source:    source,
		Timestamp: time.Now(),
	}
}

var serviceEventTypeStrings = [...]string{
	"add",
	"delete",
	"update",
}

// nolint
func (t EventType) String() string {
	return serviceEventTypeStrings[t]
}

// DefaultEvent defines common elements for service event
type DefaultEvent struct {
	io.Reader
	Action    EventType
	Key       string
	Value     string
	Timestamp time.Time
}

// GetSource return the source
func (d DefaultEvent) GetSource() interface{} {
	return d
}

// GetTimestamp return the Timestamp when the event is created
func (d DefaultEvent) GetTimestamp() time.Time {
	return d.Timestamp
}

// String return a human readable string representing this event
func (d DefaultEvent) String() string {
	return fmt.Sprintf("Event{Action{%s}, Key{%s}, Value{%s}}", d.Action, d.Key, d.Value)
}
