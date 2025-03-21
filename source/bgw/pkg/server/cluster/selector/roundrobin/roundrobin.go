package roundrobin

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
)

func init() {
	cluster.Register(constant.SelectorRoundRobin, New())
}

const (
	// nolint
	COMPLETE = 0
	// nolint
	UPDATING = 1
)

var (
	instanceWeightMap sync.Map          // [string]instance
	state             = int32(COMPLETE) // update lock acquired ?
	recyclePeriod     = 60 * time.Second.Nanoseconds()
)

type roundRobinLoadBalance struct{}

// NewRoundRobinLoadBalance returns a round-robin load balance
//
// Use the weight's common advisory to determine round-robin ratio
func New() cluster.Selector {
	return &roundRobinLoadBalance{}
}

// Select gets invoker based on round-robin load balancing strategy
func (lb *roundRobinLoadBalance) Select(ctx context.Context, instances []registry.ServiceInstance) (registry.ServiceInstance, error) {
	count := len(instances)
	if count == 0 {
		return nil, cluster.ErrServiceNotFound
	}
	if count == 1 {
		return instances[0], nil
	}

	instances = cluster.LocalAware(instances)

	cache, _ := instanceWeightMap.LoadOrStore(instances[0].GetID(), &cachedInvokers{})
	cachedInvokers := cache.(*cachedInvokers)

	var (
		clean               = false
		totalWeight         = int64(0)
		maxCurrentWeight    = int64(math.MinInt64)
		now                 = time.Now()
		selectedInvoker     registry.ServiceInstance
		selectedWeightRobin *weightedRoundRobin
	)

	for _, instance := range instances {
		weight := instance.GetWeight()
		if weight < 0 {
			weight = 0
		}

		loaded, found := cachedInvokers.LoadOrStore(instance.GetID(), &weightedRoundRobin{weight: weight})
		weightRobin := loaded.(*weightedRoundRobin)
		if !found {
			clean = true
		}

		if weightRobin.Weight() != weight {
			weightRobin.setWeight(weight)
		}

		currentWeight := weightRobin.increaseCurrent()
		weightRobin.lastUpdate = &now

		if currentWeight > maxCurrentWeight {
			maxCurrentWeight = currentWeight
			selectedInvoker = instance
			selectedWeightRobin = weightRobin
		}
		totalWeight += weight
	}

	cleanIfRequired(clean, cachedInvokers, &now)

	if selectedWeightRobin != nil {
		selectedWeightRobin.Current(totalWeight)
		return selectedInvoker, nil
	}

	// should never happen
	return instances[0], nil
}

func cleanIfRequired(clean bool, invokers *cachedInvokers, now *time.Time) {
	if clean && atomic.CompareAndSwapInt32(&state, COMPLETE, UPDATING) {
		defer atomic.CompareAndSwapInt32(&state, UPDATING, COMPLETE)
		invokers.Range(func(identify, robin interface{}) bool {
			weightedRoundRobin := robin.(*weightedRoundRobin)
			elapsed := now.Sub(*weightedRoundRobin.lastUpdate).Nanoseconds()
			if elapsed > recyclePeriod {
				invokers.Delete(identify)
			}
			return true
		})
	}
}

// Record the weight of the invoker
type weightedRoundRobin struct {
	weight     int64
	current    int64
	lastUpdate *time.Time
}

func (robin *weightedRoundRobin) Weight() int64 {
	return atomic.LoadInt64(&robin.weight)
}

func (robin *weightedRoundRobin) setWeight(weight int64) {
	robin.weight = weight
	robin.current = 0
}

func (robin *weightedRoundRobin) increaseCurrent() int64 {
	return atomic.AddInt64(&robin.current, robin.weight)
}

func (robin *weightedRoundRobin) Current(delta int64) {
	atomic.AddInt64(&robin.current, -1*delta)
}

type cachedInvokers struct {
	sync.Map /*[string]weightedRoundRobin*/
}
