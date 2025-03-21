package whitelist

import (
	"context"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
)

func init() {
	cluster.Register(constant.SelectorWhitelist, New())
}

type whitelist struct{}

func New() cluster.Selector {
	return &whitelist{}
}

func (lb *whitelist) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	return nil, nil
}
