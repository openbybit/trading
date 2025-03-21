package dispatcher

import (
	"context"
	"reflect"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

// DirectEventDispatcher Dispatcher event to listener direct
type DirectEventDispatcher struct {
	ctx context.Context
	observer.BaseListenerManager
}

// NewDirectEventDispatcher ac constructor of DirectEventDispatcher
func NewDirectEventDispatcher(ctx context.Context) observer.EventDispatcher {
	return &DirectEventDispatcher{
		ctx:                 ctx,
		BaseListenerManager: observer.NewBaseManager(),
	}
}

// Dispatch event directly
// it lookup the listener by event's type.
// if listener not found, it just return and do nothing
func (ded *DirectEventDispatcher) Dispatch(event observer.Event) error {
	if event == nil {
		return nil
	}
	eventType := reflect.TypeOf(event).Elem()
	ded.Mutex.RLock()
	defer ded.Mutex.RUnlock()
	listeners, loaded := ded.ListenersCache[eventType]
	if !loaded {
		return nil
	}
	for _, listener := range listeners {
		if ce, ok := listener.(observer.ConditionalEventListener); ok {
			if !ce.Accept(event) {
				// !NOTE: remove listener when reject event
				ded.RemoveEventListener(ce)
				continue
			}
		}

		if err := listener.OnEvent(event); err != nil {
			return err
		}
	}

	return nil
}
