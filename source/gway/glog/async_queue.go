package glog

import (
	"sync/atomic"
	"unsafe"
)

// lockfreeQueue implements lock-free FIFO freelist based queue.
// ref: https://dl.acm.org/citation.cfm?doid=248052.248106
// copyright: https://github.com/golang-design/lockfree
type lockfreeQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
	len  uint64
}

// newQueue creates a new lock-free queue.
func newQueue() *lockfreeQueue {
	head := directItem{next: nil, v: nil} // allocate a free item
	return &lockfreeQueue{
		tail: unsafe.Pointer(&head), // both head and tail points
		head: unsafe.Pointer(&head), // to the free item
	}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *lockfreeQueue) Enqueue(v interface{}) {
	i := &directItem{next: nil, v: v} // allocate new item
	var last, lastnext *directItem
	for {
		last = loaditem(&q.tail)
		lastnext = loaditem(&last.next)
		if loaditem(&q.tail) == last { // are tail and next consistent?
			if lastnext == nil { // was tail pointing to the last node?
				if casitem(&last.next, lastnext, i) { // try to link item at the end of linked list
					casitem(&q.tail, last, i) // enqueue is done. try swing tail to the inserted node
					atomic.AddUint64(&q.len, 1)
					return
				}
			} else { // tail was not pointing to the last node
				casitem(&q.tail, last, lastnext) // try swing tail to the next node
			}
		}
	}
}

// Dequeue removes and returns the value at the head of the queue.
// It returns nil if the queue is empty.
func (q *lockfreeQueue) Dequeue() interface{} {
	var first, last, firstnext *directItem
	for {
		first = loaditem(&q.head)
		last = loaditem(&q.tail)
		firstnext = loaditem(&first.next)
		if first == loaditem(&q.head) { // are head, tail and next consistent?
			if first == last { // is queue empty?
				if firstnext == nil { // queue is empty, couldn't dequeue
					return nil
				}
				casitem(&q.tail, last, firstnext) // tail is falling behind, try to advance it
			} else { // read value before cas, otherwise another dequeue might free the next node
				v := firstnext.v
				if casitem(&q.head, first, firstnext) { // try to swing head to the next node
					atomic.AddUint64(&q.len, ^uint64(0))
					return v // queue was not empty and dequeue finished.
				}
			}
		}
	}
}

// Length returns the length of the queue.
func (q *lockfreeQueue) Length() uint64 {
	return atomic.LoadUint64(&q.len)
}

type directItem struct {
	next unsafe.Pointer
	v    interface{}
}

func loaditem(p *unsafe.Pointer) *directItem {
	return (*directItem)(atomic.LoadPointer(p))
}
func casitem(p *unsafe.Pointer, old, new *directItem) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
