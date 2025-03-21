package observer

import (
	"context"
	"reflect"
)

type EventListener interface {
	Prioritizer
	// OnEvent handle this event
	OnEvent(e Event) error
	// GetEventType listen which event type
	GetEventType() reflect.Type
}

type EmptyListener struct{}

func (el EmptyListener) OnEvent(e Event) error {
	panic("")
}

func (el EmptyListener) GetEventType() reflect.Type {
	return nil
}

func (el EmptyListener) GetPriority() int {
	return 0
}

// ConditionalEventListener only handle the event which it can handle
type ConditionalEventListener interface {
	EventListener
	// Accept will make the decision whether it should handle this event
	Accept(e Event) bool
}

// Prioritizer is the abstraction of priority.
type Prioritizer interface {
	// GetPriority will return the priority
	// The lesser the faster
	GetPriority() int
}

// LogEventListener is singleton
type LogEventListener struct {
	ctx context.Context
}

func (l *LogEventListener) GetPriority() int {
	return 0
}

func (l *LogEventListener) OnEvent(e Event) error {
	return nil
}

func (l *LogEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&BaseEvent{})
}

// NoopEventListener is dummy
type NoopEventListener struct{}

func (l *NoopEventListener) GetPriority() int {
	return -1
}

func (l *NoopEventListener) OnEvent(e Event) error {
	return nil
}

func (l *NoopEventListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&BaseEvent{})
}
