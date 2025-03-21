package random

import (
	"context"
	"math/rand"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
)

func init() {
	cluster.Register(constant.SelectorRandom, New())
}

type randomLoadBalance struct{}

// NewRandomLoadBalance returns a random load balance instance.
// Set random probabilities by weight, and the request will be sent to provider randomly.
func New() cluster.Selector {
	return &randomLoadBalance{}
}

func (lb *randomLoadBalance) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	ins = cluster.LocalAware(ins)

	length := len(ins)
	if length == 0 {
		return nil, cluster.ErrServiceNotFound
	}
	if length == 1 {
		return ins[0], nil
	}
	sameWeight := true
	weights := make([]int64, length)

	firstWeight := ins[0].GetWeight()
	totalWeight := firstWeight
	weights[0] = firstWeight

	for i := 1; i < length; i++ {
		weight := ins[i].GetWeight()
		weights[i] = weight

		totalWeight += weight
		if sameWeight && weight != firstWeight {
			sameWeight = false
		}
	}

	if totalWeight > 0 && !sameWeight {
		// If (not every invoker has the same weight & at least one invoker's weight>0),
		// select randomly based on totalWeight.
		offset := rand.Int63n(totalWeight)

		for i := 0; i < length; i++ {
			offset -= weights[i]
			if offset < 0 {
				return ins[i], nil
			}
		}
	}
	// If all invokers have the same weight value or totalWeight=0, return evenly.
	idx := rand.Intn(length)
	// go build -gcflags="-d=ssa/check_bce/debug=1"  random.go
	// optimize: bounds check elimination
	if len(ins) > idx && idx >= 0 {
		return ins[idx], nil
	}
	return nil, cluster.ErrServiceNotFound
}
