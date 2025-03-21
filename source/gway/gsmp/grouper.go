package gsmp

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/willf/bloom"

	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/gsmp/imp"

	"go.uber.org/atomic"
)

const (
	defaultInterval = 120
)

type Grouper interface {
	Init(ctx context.Context)
	GetGroup(ctx context.Context, uid int64) (group int32, err error)
	HandleMsg(msg []byte) error
}

type Discovery = func(ctx context.Context, registry, namespace, group string) (addrs []string)

type grouper struct {
	registry  string
	group     string
	namespace string
	discovery Discovery
	index     *atomic.Int32
	connPool  pool.Pools

	synced *atomic.Int32 // synced indicates sync status: 1 processing\ 2 success \3 failed

	sync.RWMutex
	groups map[int64]int32    // uid -> group_id
	uids   map[string][]int64 // inst_id -> uids

	bLock sync.RWMutex
	bl    *bloom.BloomFilter

	interval int // (s), for ut
}

type Config struct {
	Registry  string
	Group     string
	Namespace string
	Discovery Discovery
}

func New(cfg *Config) (Grouper, error) {
	if cfg.Discovery == nil || cfg.Registry == "" || cfg.Group == "" || cfg.Namespace == "" {
		return nil, errors.New("bad config")
	}

	return &grouper{
		registry:  cfg.Registry,
		group:     cfg.Group,
		namespace: cfg.Namespace,
		discovery: cfg.Discovery,
		connPool:  pool.NewPools(),
		groups:    make(map[int64]int32, 0),
		uids:      make(map[string][]int64, 0),
		index:     atomic.NewInt32(0),
		synced:    atomic.NewInt32(0),
		bl:        bloom.New(1<<30, 7),
		interval:  defaultInterval,
	}, nil
}

func (g *grouper) Init(ctx context.Context) {
	// asynchronously get full data
	go g.init(ctx)
}

func (g *grouper) init(ctx context.Context) {
	var (
		n     int
		timer = time.NewTimer(time.Second)
	)
	defer timer.Stop()

	for g.synced.Load() != 1 {
		data, uids, err := g.GetImpGroup(ctx, 0)
		if err == nil {
			g.Lock()
			g.groups = data
			g.uids = uids
			g.Unlock()
			g.synced.Store(1)
			log.Printf("init success, %d", len(data))
			return
		}
		log.Printf("init err, %d, %s", n, err.Error())
		n += 1
		if n > 10 {
			g.synced.Store(2)
			log.Printf("smp grouper get full data failed, err = %s", err.Error())
			return
		}
		d := n * 20
		if d > g.interval {
			d = g.interval
		}
		timer.Reset(time.Duration(d) * time.Second)
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

func (g *grouper) GetGroup(ctx context.Context, uid int64) (int32, error) {
	g.RLock()
	group, ok := g.groups[uid]
	g.RUnlock()
	if ok {
		return group, nil
	}

	if g.synced.Load() != 1 {
		return g.GetUidGroup(ctx, uid)
	}
	return 0, nil
}

func (g *grouper) GetImpGroup(ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
	conn, err := g.GetImpConn(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	// no uid, full data request
	req := &imp.SmpGroupQueryReq{
		Uid: uid,
	}
	res, err := imp.NewImpSmpServiceClient(conn.Client()).SmpGroupQuery(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if res.RetCode != "0" {
		return nil, nil, errors.New(res.RetMsg)
	}

	return convert(res.SmpGroups...), getInstUids(res.SmpGroups...), nil
}

func (g *grouper) GetUidGroup(ctx context.Context, uid int64) (int32, error) {

	log.Printf("uid is %d", uid)

	d := make([]byte, 8)
	binary.BigEndian.PutUint64(d, uint64(uid))
	g.bLock.RLock()
	if g.bl.Test(d) {
		g.bLock.RUnlock()
		return 0, nil
	}
	g.bLock.RUnlock()
	data, uids, err := g.GetImpGroup(ctx, uid)
	if err != nil {
		return 0, err
	}

	g.bLock.Lock()
	g.bl.Add(d)
	g.bLock.Unlock()

	// group id maybe not exist
	group, _ := data[uid]
	// update groups
	g.Lock()
	for id, gp := range data {
		g.groups[id] = gp
	}

	for inst, ids := range uids {
		g.uids[inst] = ids
	}
	g.Unlock()

	// if GetUidGroup succeed and Init failed, try Init again.
	if g.synced.CompareAndSwap(2, 0) {
		g.Init(ctx)
	}
	return group, nil
}

func (g *grouper) GetImpConn(ctx context.Context) (pool.Conn, error) {
	addr, err := g.getServiceRoundRobin(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := g.connPool.GetConn(ctx, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (g *grouper) getServiceRoundRobin(ctx context.Context) (addr string, err error) {
	addrs := g.discovery(ctx, g.registry, g.namespace, g.group)
	if len(addrs) == 0 {
		return "", errors.New("[Discovery]no instances found")
	}

	cur := int(g.index.Add(1))
	cur = cur % len(addrs)

	return addrs[cur], nil
}

func (g *grouper) HandleMsg(value []byte) error {
	msg := &imp.SmpGroupConfigItem{}
	err := json.Unmarshal(value, msg)
	if err != nil {
		return err
	}

	groups := convert(msg)
	uids := getInstUids(msg)
	// uid\groups of the inst are removed
	if len(groups) == 0 {
		g.Lock()
		uids := g.uids[msg.InstId]
		delete(g.uids, msg.InstId)
		for _, uid := range uids {
			delete(g.groups, uid)
		}
		g.Unlock()
		return nil
	}

	g.Lock()
	// get current uids of the inst
	curUids := make(map[int64]struct{}, 0)
	for inst, _ := range uids {
		uids := g.uids[inst]
		for _, uid := range uids {
			curUids[uid] = struct{}{}
		}
	}

	for id, gp := range groups {
		g.groups[id] = gp
		delete(curUids, id)
	}
	// delete empty group uids
	if len(curUids) > 0 {
		for uid, _ := range curUids {
			delete(g.groups, uid)
		}
	}
	for inst, ids := range uids {
		g.uids[inst] = ids
	}
	g.Unlock()
	return nil
}

func convert(insts ...*imp.SmpGroupConfigItem) map[int64]int32 {
	res := make(map[int64]int32, 0)
	for _, inst := range insts {
		for _, g := range inst.InstSmpGroup {
			for _, uid := range g.Uids {
				res[uid] = g.SmpGroup
			}
		}
	}

	return res
}

func getInstUids(insts ...*imp.SmpGroupConfigItem) map[string][]int64 {
	res := make(map[string][]int64, 0)
	for _, inst := range insts {
		uids := make([]int64, 0)
		for _, g := range inst.InstSmpGroup {
			for _, uid := range g.Uids {
				uids = append(uids, uid)
			}
		}
		res[inst.InstId] = uids
	}

	return res
}
