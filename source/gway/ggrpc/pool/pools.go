package pool

import (
	"container/list"
	"context"
	"errors"
	"log"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
)

var (
	errPoolNoAvaiableConn = errors.New("pool no available conn")
)

var global = NewPools()

func SetDefault(g Pools) {
	if g != nil {
		global = g
	}
}

func GetConn(ctx context.Context, target string) (Conn, error) {
	return global.GetConn(ctx, target)
}

func Remove(targets ...string) {
	global.Remove(targets...)
}

func Close() {
	global.Close()
}

// Pools is the interface that provides the pools methods
//
//go:generate mockgen -source=pools.go -destination=pools_mock.go -package=pool
type Pools interface {
	GetConn(ctx context.Context, target string) (Conn, error)
	Remove(targets ...string)
	Close()

	// GetStatus is used for Observability and returns pool status of target.
	// If target is "all", return all pools status.
	GetStatus(ctx context.Context, target string) (string, bool)
}

// NewPools returns a new Pools
func NewPools(opts ...Option) Pools {
	ps := &pools{
		opts:  opts,
		pools: make(map[string]Pool),
		ch:    make(chan Pool, 1024),
	}

	ps.daemon()
	return ps
}

type pools struct {
	mux   sync.RWMutex
	pools map[string]Pool
	opts  []Option
	ch    chan Pool
}

// GetConn returns a connection from the pool
func (p *pools) GetConn(ctx context.Context, target string) (Conn, error) {
	pp := p.GetPool(target)
	if pp != nil {
		return pp.Get(ctx)
	}

	return nil, errPoolNoAvaiableConn
}

// GetPool returns a pool by target
func (p *pools) GetPool(addr string) Pool {
	p.mux.RLock()
	res := p.pools[addr]
	p.mux.RUnlock()
	if res != nil {
		return res
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	res = p.pools[addr]
	if res != nil {
		return res
	}

	res, err := New(context.Background(), addr, p.opts...)
	if err != nil {
		log.Printf("[pool]new pool failed, addr = %s, err = %s", addr, err.Error())
	}

	if res != nil {
		p.pools[addr] = res
	}

	return res
}

// Remove removes the pool by target
func (p *pools) Remove(targets ...string) {
	if len(targets) == 0 {
		return
	}

	removed := make([]Pool, 0, len(targets))

	p.mux.Lock()
	for _, target := range targets {
		pp, ok := p.pools[target]
		if !ok {
			continue
		}
		delete(p.pools, target)
		removed = append(removed, pp)
	}
	p.mux.Unlock()

	for _, pp := range removed {
		pp.Close()
		if !pp.Closed() {
			select {
			// non block
			case p.ch <- pp:
			default:
			}
		}
	}
}

// Close closes all pools
func (p *pools) Close() {
	p.mux.Lock()
	pools := p.pools
	p.pools = make(map[string]Pool)
	p.mux.Unlock()
	for _, pp := range pools {
		pp.Close()
	}
	close(p.ch)
}

func (p *pools) GetStatus(ctx context.Context, target string) (string, bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if target == "all" {
		res := make(map[string]string)
		for t, pl := range p.pools {
			if pl != nil {
				res[t] = pl.Status()
			}
		}
		data, _ := jsoniter.Marshal(res)
		return string(data), true
	}

	pl, ok := p.pools[target]
	if !ok {
		return "", false
	}

	return pl.Status(), true
}

type element struct {
	p Pool
	t time.Time
}

func (p *pools) daemon() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("grpc pools daemon recovery, err = %v", r)
				gmetric.IncDefaultError("pools_daemon_panic", "")
				// restart daemon, but sleep 10s to avoid frequent restart
				time.Sleep(10 * time.Second)
				p.daemon()
			}
		}()

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		poolList := list.New()

		for {
			select {
			case pl, ok := <-p.ch:
				if !ok {
					break
				}
				v := &element{
					p: pl,
					t: time.Now(),
				}
				poolList.PushBack(v)
			case <-ticker.C:
				node := poolList.Front()
				l := poolList.Len()
				for i := 0; i < l; i++ {
					v := node.Value.(*element)
					next := node.Next()
					if v == nil {
						node = next
						continue
					}
					if v.p.Closed() {
						poolList.Remove(node)
						node = next
						continue
					}
					if v.t.Add(30 * time.Minute).Before(time.Now()) {
						gmetric.IncDefaultError("pool_leak", "")
						log.Printf("pool leak, status is %s", v.p.Status())
						v.p.ForceClose()
						poolList.Remove(node)
					}
					node = next
				}
			}
		}
	}()
}
