package glog

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	defaultAsyncMaxSize  = 1024 * 1024           // 默认异步日志大小
	defaultMaxCloseTime  = time.Second * 30      // 默认最大等待时间
	defaultAsyncInterval = time.Millisecond * 50 // 这个值过大容易导致日志丢失
)

var ErrMessageDiscard = errors.New("message discard")

func ignoreDiscard(entry Entry, fields []Field) {}

type asyncNode struct {
	entry  zapcore.Entry
	fields []zapcore.Field
}

type asyncWriter struct {
	zapcore.Core
	queue        *lockfreeQueue //
	maxSize      uint64         // 最大数量
	interval     time.Duration  // 当没有数据时休眠间隔
	maxCloseTime time.Duration  // 退出最大等待时间
	pool         sync.Pool      // node pool
	onDiscard    DiscardFunc    // 超过最大数量时直接丢弃
	running      int32          // 运行状态
}

func newAsyncWriter(raw zapcore.Core, maxSize uint64, interval time.Duration, maxCloseTime time.Duration, onDiscard DiscardFunc) zapcore.Core {
	if maxSize <= 0 {
		maxSize = defaultAsyncMaxSize
	}

	if interval <= 0 {
		interval = defaultAsyncInterval
	}

	if maxCloseTime <= 0 {
		maxCloseTime = defaultMaxCloseTime
	}

	if onDiscard == nil {
		onDiscard = ignoreDiscard
	}

	res := &asyncWriter{
		Core:         raw,
		queue:        newQueue(),
		maxSize:      maxSize,
		interval:     interval,
		maxCloseTime: maxCloseTime,
		onDiscard:    onDiscard,
		running:      1,
		pool: sync.Pool{
			New: func() interface{} { return &asyncNode{} },
		},
	}

	go res.loop()

	return res
}

func (l *asyncWriter) isRunning() bool {
	return atomic.LoadInt32(&l.running) == 1
}

func (l *asyncWriter) Close() error {
	atomic.StoreInt32(&l.running, 0)
	if l.queue.Length() > 0 {
		start := time.Now()
		for time.Since(start) < l.maxCloseTime {
			if l.queue.Length() == 0 {
				break
			}
			time.Sleep(time.Millisecond * 30)
		}
	}

	return nil
}

func (l *asyncWriter) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if !l.isRunning() || l.queue.Length() > l.maxSize {
		l.onDiscard(ent, fields)
		return ErrMessageDiscard
	}

	node := l.pool.Get().(*asyncNode)
	node.entry = ent
	node.fields = fields
	l.queue.Enqueue(node)

	return nil
}

func (l *asyncWriter) loop() {
	for {
		item := l.queue.Dequeue()
		if item == nil {
			time.Sleep(l.interval)
			continue
		}

		node := item.(*asyncNode)

		_ = l.Core.Write(node.entry, node.fields)
		l.pool.Put(node)
	}
}
