package dispatcher

import (
	"context"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

// AsyncEventDispatcher is a async event dispatcher
type AsyncEventDispatcher struct {
	ctx context.Context
	wg  sync.WaitGroup
	observer.BaseListenerManager
}

// NewAsyncEventDispatcher create a async event dispatcher
func NewAsyncEventDispatcher(ctx context.Context) *AsyncEventDispatcher {
	return &AsyncEventDispatcher{
		ctx:                 ctx,
		BaseListenerManager: observer.NewBaseManager(),
	}
}

// Dispatch event currentcy
func (aed *AsyncEventDispatcher) Dispatch(event observer.Event) error {
	if event == nil {
		return nil
	}
	eventType := reflect.TypeOf(event).Elem()
	aed.Mutex.RLock()
	defer aed.Mutex.RUnlock()
	ls, loaded := aed.ListenersCache[eventType]
	if !loaded {
		return nil
	}

	var err error
	for _, l := range ls {
		aed.wg.Add(1)
		go func(l observer.EventListener) {
			defer aed.wg.Done()
			if err = l.OnEvent(event); err != nil {

			}
		}(l)
	}

	aed.wg.Wait()
	return err
}
