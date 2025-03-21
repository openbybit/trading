package pool

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

const (
	ready = iota
	closing
	closed
)

// ErrClosed is the error resulting if the pool is closed via pool.Close().
var (
	ErrClosed    = errors.New("pool is closed")
	ErrFetchFast = errors.New("fast fetch conn error, try again later")
)

// Pool interface describes a pool implementation.
// An ideal pool is threadsafe and easy to use.
//
//go:generate mockgen -source=pool.go -destination=pool_mock.go -package=pool
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error. we guarantee the conn.Value() isn't nil when conn isn't nil.
	Get(context.Context) (Conn, error)

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable. You can't make concurrent calls Close and Get method.
	// It will be cause panic.
	Close()

	// Status returns the current status of the pool.
	Status() string

	// Closed return whether the pool is closed.
	Closed() bool

	// ForceClose close pool in closing status whether its ref is not zero.
	ForceClose() bool
}

type pool struct {
	// atomic, used to get connection random
	index uint32

	// atomic, the current physical connection of pool
	current int32

	// atomic, the using logic connection of pool
	// logic connection = physical connection * MaxConcurrentStreams
	ref int32

	// pool options
	opt Options

	// all of created physical connections
	conns []*conn

	// the server address is to create connection.
	address string

	// this is close flag
	closeStatus int32

	// control the atomic var current's concurrent read write.
	sync.RWMutex
}

// New return a connection pool.
func New(ctx context.Context, address string, opts ...Option) (Pool, error) {
	if address == "" {
		return nil, errors.New("invalid address settings")
	}
	o := DefaultOptions
	for _, opt := range opts {
		opt(&o)
	}

	if o.MaxIdle <= 0 || o.MaxActive <= 0 || o.MaxIdle > o.MaxActive {
		return nil, errors.New("invalid maximum settings")
	}
	if o.MaxConcurrentStreams <= 0 {
		return nil, errors.New("invalid maximun settings")
	}

	p := &pool{
		index:       0,
		current:     int32(o.MaxIdle),
		ref:         0,
		opt:         o,
		conns:       make([]*conn, o.MaxActive),
		address:     address,
		closeStatus: ready,
	}

	for i := 0; i < p.opt.MaxIdle; i++ {
		c, err := p.dial(ctx)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("dial is not able to fill the pool: %w", err)
		}
		p.conns[i] = p.wrapConn(c, false)
	}

	return p, nil
}

func (p *pool) incrRef() int32 {
	newRef := atomic.AddInt32(&p.ref, 1)
	if newRef == math.MaxInt32 {
		panic(fmt.Sprintf("overflow ref: %d", newRef))
	}
	return newRef
}

func (p *pool) decrRef() {
	if p == nil {
		return
	}
	newRef := atomic.AddInt32(&p.ref, -1)
	if newRef < 0 {
		panic(fmt.Sprintf("negative ref: %d", newRef))
	}
	if newRef == 0 && atomic.LoadInt32(&p.current) > int32(p.opt.MaxIdle) {
		p.Lock()
		if atomic.LoadInt32(&p.ref) == 0 {
			atomic.StoreInt32(&p.current, int32(p.opt.MaxIdle))
			p.deleteFrom(p.opt.MaxIdle)
		}
		p.Unlock()
	}

	if atomic.LoadInt32(&p.ref) == 0 {
		if atomic.CompareAndSwapInt32(&p.closeStatus, closing, closed) {
			p.deleteFrom(0)
		}
	}
}

func (p *pool) reset(index int) {
	cc := p.conns[index]
	if cc == nil {
		return
	}
	_ = cc.reset()
	p.conns[index] = nil
}

func (p *pool) deleteFrom(begin int) {
	for i := begin; i < p.opt.MaxActive; i++ {
		p.reset(i)
	}
}

// Get see Pool interface.
func (p *pool) Get(ctx context.Context) (Conn, error) {
	if atomic.LoadInt32(&p.closeStatus) >= closing {
		return nil, ErrClosed
	}
	current := int(atomic.LoadInt32(&p.current))
	for i := 0; i < current; i++ {
		cc, err := p.fetchFast(ctx)
		if err == nil {
			return cc, nil
		}
	}

	return nil, ErrFetchFast
}

func (p *pool) fetchFast(ctx context.Context) (conn Conn, resErr error) {

	// if resErr != nil or conn == nil, decrease reference to avoid reference leak.
	defer func() {
		if resErr != nil || conn == nil {
			p.decrRef()
		}
	}()

	// the first selected from the created connections
	nextRef := p.incrRef()
	current := atomic.LoadInt32(&p.current)
	if current == 0 {
		return nil, ErrClosed
	}

	if nextRef <= current*int32(p.opt.MaxConcurrentStreams) {
		next := atomic.AddUint32(&p.index, 1) % uint32(current)
		return p.getConn(next)
	}

	// the number connection of pool is reach to max active
	if current == int32(p.opt.MaxActive) {
		// the second if reuse is true, select from pool's connections
		if p.opt.Reuse {
			next := atomic.AddUint32(&p.index, 1) % uint32(current)
			return p.getConn(next)
		}

		// the third create one-time connection
		c, err := p.dial(ctx)
		return p.wrapConn(c, true), err
	}

	// the fourth create new connections given back to pool
	p.Lock()
	current = atomic.LoadInt32(&p.current)
	if current < int32(p.opt.MaxActive) && nextRef > current*int32(p.opt.MaxConcurrentStreams) {
		// 2 times the incremental or the remain incremental
		increment := current
		if current+increment > int32(p.opt.MaxActive) {
			increment = int32(p.opt.MaxActive) - current
		}
		var i int32
		var err error
		for i = 0; i < increment; i++ {
			c, er := p.dial(ctx)
			if er != nil {
				err = er
				break
			}
			p.reset(int(current + i))
			p.conns[current+i] = p.wrapConn(c, false)
		}
		current += i
		atomic.StoreInt32(&p.current, current)
		if err != nil {
			p.Unlock()
			return nil, err
		}
	}
	p.Unlock()
	next := atomic.AddUint32(&p.index, 1) % uint32(current)
	return p.getConn(next)
}

// Close see Pool interface.
func (p *pool) Close() {
	p.Lock()
	defer p.Unlock()

	if atomic.LoadInt32(&p.closeStatus) >= closing {
		return
	}
	atomic.StoreInt32(&p.closeStatus, closing)

	if atomic.LoadInt32(&p.ref) == 0 {
		p.deleteFrom(0)
		atomic.StoreInt32(&p.closeStatus, closed)
	}
}

// Status see Pool interface.
func (p *pool) Status() string {
	return fmt.Sprintf("address: %s, status: %d, index: %d, current: %d, ref: %d, option: %v",
		p.address, atomic.LoadInt32(&p.closeStatus), atomic.LoadUint32(&p.index), atomic.LoadInt32(&p.current), atomic.LoadInt32(&p.ref), p.opt)
}

func (p *pool) getConn(index uint32) (Conn, error) {
	p.RLock()
	defer p.RUnlock()

	cc := p.conns[index]
	if cc.validState() {
		return cc, nil
	}

	cc.Reset()
	// retry to get connection
	if cc.validState() {
		return cc, nil
	}
	return nil, ErrFetchFast
}

func (p *pool) dial(ctx context.Context) (*grpc.ClientConn, error) {
	return Dial(ctx, p.address, p.opt.DialOptions...)
}

func (p *pool) Closed() bool {
	return atomic.LoadInt32(&p.closeStatus) == closed
}

func (p *pool) ForceClose() bool {
	if !atomic.CompareAndSwapInt32(&p.closeStatus, closing, closed) {
		return false
	}

	p.Lock()
	defer p.Unlock()
	p.deleteFrom(0)
	return true
}
