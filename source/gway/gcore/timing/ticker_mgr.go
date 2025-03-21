package timing

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Ticker interface {
	ID() string
	Value() interface{}
}

// TickerFunc ticker callback, Do not do time-consuming operations,if need, post to another goroutines.
type TickerFunc func(t Ticker)

type TickerManager interface {
	Start()
	Stop()

	Size() int
	// Create 创建ticker,可传入一个唯一的外部id,如果为空则忽略此id,如果不为空则会建立额外索引,可通过RemoveByID删除ticker
	Create(id string, value interface{}) Ticker
	Remove(t Ticker)
	RemoveByID(id string) Ticker
}

func NewTickerManager(duration, precision time.Duration, balance bool, cb TickerFunc) TickerManager {
	if duration < 0 || precision < 0 || duration < precision {
		panic(errors.New("invalid ticker duration"))
	}

	if cb == nil {
		panic(errors.New("invalid ticker callback"))
	}

	count := int(math.Ceil(float64(duration) / float64(precision)))
	duration = precision * time.Duration(count)
	slots := make([]*tickerList, count)
	for i := 0; i < count; i++ {
		slots[i] = newTickerList()
	}

	tm := &tickerManager{
		tickers:   make(map[string]*ticker),
		slots:     slots,
		duration:  duration,
		precision: precision,
		balance:   balance,
		cb:        cb,
	}

	return tm
}

var tickerPool = sync.Pool{
	New: func() interface{} {
		return &ticker{}
	},
}

type ticker struct {
	prev *ticker
	next *ticker

	id    string
	value interface{}
}

func (t *ticker) ID() string {
	return t.id
}

func (t *ticker) Value() interface{} {
	return t.value
}

func newTickerList() *tickerList {
	dummy := tickerPool.Get().(*ticker)
	dummy.next = dummy
	dummy.prev = dummy

	l := &tickerList{
		head: dummy,
	}

	return l
}

// tickerList bidirectional acyclic linked list with dummy node
type tickerList struct {
	head *ticker
}

func (l *tickerList) InsertBack(t *ticker) {
	tail := l.head.prev
	tail.next = t
	t.prev = tail
	t.next = l.head
	l.head.prev = t
}

func (l *tickerList) InsertFrontList(head, tail *ticker) {
	first := l.head.next

	tail.next = first
	first.prev = tail

	l.head.next = head
	head.prev = l.head
}

func (l *tickerList) RemoveBackList(node *ticker) (head, tail *ticker) {
	// node must not be l.head
	oldTail := l.head.prev

	newTail := node.prev
	newTail.next = l.head
	l.head.prev = newTail

	return node, oldTail
}

type tickerManager struct {
	mux     sync.Mutex
	tickers map[string]*ticker // ticker map
	slots   []*tickerList      // all slots
	cursor  int                // slot cursor
	size    int32              //
	running int32              //

	precision time.Duration //
	duration  time.Duration // ticker duration
	balance   bool          // auto balance slots size
	cb        TickerFunc    //
	wg        sync.WaitGroup
}

func (tm *tickerManager) Start() {
	if atomic.CompareAndSwapInt32(&tm.running, 0, 1) {
		tm.wg.Add(1)
		go tm.loop()
	}
}

func (tm *tickerManager) Stop() {
	atomic.StoreInt32(&tm.running, 0)
	tm.wg.Wait()
}

func (tm *tickerManager) Size() int {
	return int(atomic.LoadInt32(&tm.size))
}

func (tm *tickerManager) Create(id string, value interface{}) Ticker {
	tm.mux.Lock()
	if id != "" {
		if res, ok := tm.tickers[id]; ok {
			tm.mux.Unlock()
			return res
		}
	}
	t := tickerPool.Get().(*ticker)
	t.id = id
	t.value = value
	index := tm.cursor - 1
	if index < 0 {
		index = len(tm.slots) - 1
	}
	tm.slots[index].InsertBack(t)
	atomic.AddInt32(&tm.size, 1)
	if id != "" {
		tm.tickers[id] = t
	}
	tm.mux.Unlock()
	return t
}

func (tm *tickerManager) Remove(tk Ticker) {
	t, ok := tk.(*ticker)
	if !ok {
		return
	}

	tm.mux.Lock()
	tm.doRemove(t)
	tm.mux.Unlock()
}

func (tm *tickerManager) RemoveByID(id string) Ticker {
	tm.mux.Lock()
	t, ok := tm.tickers[id]
	if ok {
		tm.doRemove(t)
	}
	tm.mux.Unlock()
	return t
}

func (tm *tickerManager) doRemove(t *ticker) {
	if t.id != "" {
		delete(tm.tickers, t.id)
	}

	t.prev.next = t.next
	t.next.prev = t.prev
	t.prev = nil
	t.next = nil
	atomic.AddInt32(&tm.size, -1)
	tickerPool.Put(t)
}

func (tm *tickerManager) loop() {
	defer tm.wg.Done()

	start := time.Now()
	tickers := make([]Ticker, 0, 10)
	for {
		tickers = tickers[:0]
		tm.mux.Lock()

		slot := tm.slots[tm.cursor]
		tm.cursor = (tm.cursor + 1) % len(tm.slots)

		if tm.balance {
			max := int(math.Ceil(float64(tm.size) / float64(len(tm.slots))))
			num := 0
			node := slot.head.next
			for node != slot.head && num < max {
				tickers = append(tickers, node)
				node = node.next
				num++
			}

			if node != slot.head {
				// do balance, move to next slots to continue to invoke next time
				head, tail := slot.RemoveBackList(node)
				next := tm.slots[tm.cursor]
				next.InsertFrontList(head, tail)
			}
		} else {
			for node := slot.head.next; node != slot.head; node = node.next {
				tickers = append(tickers, node)
			}
		}

		tm.mux.Unlock()

		// invoke all tickers
		for _, t := range tickers {
			tm.cb(t)
		}

		if atomic.LoadInt32(&tm.running) == 0 {
			break
		}

		now := time.Now()
		count := now.Sub(start)/tm.precision + 1
		next := start.Add(tm.precision * count)
		interval := next.Sub(now)
		if interval < 0 || interval > tm.precision {
			interval = tm.precision
		}

		time.Sleep(interval)
	}
}
