package getcd

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EventListener interface {
	Listen(key string, listener ...observer.EventListener)
	ListenWithChildren(key string, listener observer.EventListener)
	Wait()
}

// EventListener EventListener
type eventListener struct {
	ctx    context.Context
	client Client
	keys   *container.HashSet
	wg     sync.WaitGroup
}

// NewEventListener returns a EventListener instance
func NewEventListener(ctx context.Context, client Client) EventListener {
	return &eventListener{
		ctx:    ctx,
		client: client,
		keys:   container.NewSet(),
	}
}

// Listen Listen on a spec key
// this method will return true when spec key deleted,
// this method will return false when deep layer connection lose
func (l *eventListener) Listen(key string, listener ...observer.EventListener) {
	if l.keys.Contains(key) {
		log.Println("etcd key has already been listened", key)
		return
	}

	l.keys.Add(key)

	l.wg.Add(1)
	go func(key string, listener ...observer.EventListener) {
		l.listen(key, listener...)
	}(key, listener...)
}

func (l *eventListener) listen(key string, listener ...observer.EventListener) {
	defer l.wg.Done()
	for {
		wc, err := l.client.Watch(key)
		if err != nil {
			log.Printf("watch key %s error: %v\n", key, err)
			return
		}

		select {
		// client stopped
		case <-l.client.Done():
			log.Println("etcd client stopped", key)
			return

		// client ctx stop
		case <-l.client.GetCtx().Done():
			log.Println("etcd client ctx cancel", key)
			return

		// handle etcd events
		case e, ok := <-wc:
			if !ok {
				log.Println("etcd watch-chan closed", key)
				return
			}

			if e.Err() != nil {
				log.Println("etcd watch error:", key, e.Err())
				continue
			}

			for _, event := range e.Events {
				l.handleEvents(event, listener...)
			}
		}
	}
}

// return true means the event type is DELETE
// return false means the event type is CREATE || UPDATE
func (l *eventListener) handleEvents(event *clientv3.Event, listeners ...observer.EventListener) bool {
	re := &observer.DefaultEvent{
		Key:   string(event.Kv.Key),
		Value: string(event.Kv.Value),
	}

	switch event.Type {
	case mvccpb.PUT:
		for _, listener := range listeners {
			if event.IsCreate() {
				re.Action = observer.EventTypeAdd
				err := listener.OnEvent(re)
				if err != nil {
					log.Println("[etcd]listener fire event error", event.Kv.Key)
				}
			} else {
				re.Action = observer.EventTypeUpdate
				err := listener.OnEvent(re)
				if err != nil {
					log.Println("[etcd]listener fire event error", event.Kv.Key)
				}
			}
		}
		return false
	case mvccpb.DELETE:
		re.Action = observer.EventTypeDel
		for _, listener := range listeners {
			re.Action = observer.EventTypeDel
			err := listener.OnEvent(re)
			if err != nil {
				log.Println("[etcd]listener fire event error", event.Kv.Key)
			}
		}
		return true
	default:
		log.Println("[etcd]unknown type", event.Kv.Key)
		return false
	}
}

// listenWithPrefix listens on a set of key with spec prefix
func (l *eventListener) listenWithPrefix(prefix string, listener ...observer.EventListener) {
	defer l.wg.Done()
	retryWait := time.Second

retry:
	wc, err := l.client.WatchWithPrefix(prefix)
	if err != nil {
		log.Println("etcd WatchWithPrefix error", err, prefix)
		return
	}

	for {
		select {
		case <-l.client.Done():
			log.Println("etcd client stopped", prefix)
			return
		case <-l.client.GetCtx().Done():
			log.Println("etcd client ctx cancel", prefix)
			return
		case e, ok := <-wc:
			if !ok {
				for {
					if retryWait < time.Minute {
						retryWait *= 2
					} else {
						retryWait = time.Second
						log.Println("etcd WatchWithPrefix chan error, on max retryWait", prefix)
					}
					log.Println("etcd watch closed, retry watch", prefix, retryWait)
					time.Sleep(retryWait)

					ok, err := l.handlePrefixEvent(prefix, listener...)
					if err != nil {
						return
					}
					if ok {
						goto retry
					}
				}
			}

			if e.Err() != nil {
				log.Println("etcd watch error", prefix, e.Err())
				continue
			}

			for _, event := range e.Events {
				l.handleEvents(event, listener...)
			}
		}
	}
}

// ListenWithChildren listens on a set of key with spec prefix
func (l *eventListener) ListenWithChildren(key string, listener observer.EventListener) {
	if l.keys.Contains(key) {
		log.Println("etcd key has already been listened", key)
		return
	}

	l.keys.Add(key)

	_, err := l.handlePrefixEvent(key, listener)
	if err != nil {
		log.Println("Get new node path content error", key, err)
	}

	l.wg.Add(1)
	go func(key string, listener observer.EventListener) {
		l.listenWithPrefix(key, listener)
	}(key, listener)
}

func (l *eventListener) handlePrefixEvent(prefix string, listeners ...observer.EventListener) (bool, error) {
	keys, values, err := l.client.GetChildren(prefix)
	if err != nil {
		if errors.Is(err, ErrNilETCDV3Client) {
			log.Println("ListenWithPrefix quit", prefix, err)
			return false, err
		}
		log.Println("Get new node path content error", prefix, err)
		return false, nil
	}

	for i, k := range keys {
		for _, eventListener := range listeners {
			err := eventListener.OnEvent(&observer.DefaultEvent{
				Key:    k,
				Action: observer.EventTypeAdd,
				Value:  values[i],
			})
			if err != nil {
				log.Println("event fire err", prefix, err)
			}
		}
	}
	return true, nil
}

// Wait Wait for all listener to stop
func (l *eventListener) Wait() {
	l.wg.Wait()
}
