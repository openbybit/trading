package raft

import (
	"context"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
)

func init() {
	cluster.Register(constant.SelectorRaft, New())
}

type zone struct {
}

func New() cluster.Selector {
	return &zone{}
}

func (lb *zone) Select(_ context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	var cur registry.ServiceInstance
	for _, instance := range ins {
		im := instance.GetMetadata() // instance metadata

		if im.GetRole().IsLeader() {
			if cur == nil {
				cur = instance
			} else {
				if cur.GetMetadata().GetTerm() < im.GetTerm() {
					cur = instance
				}
			}

		}
	}
	if cur == nil {
		return nil, cluster.ErrServiceNotFound
	}

	return cur, nil
}
