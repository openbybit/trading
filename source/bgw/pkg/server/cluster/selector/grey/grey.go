package grey

import (
	"context"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
)

// Init 此为部署的灰度策略
func Init() {
	cluster.Register(constant.SelectorGrey, New())
}

// lb *greyLoadBalance bgw/pkg/cluster/selector.Selector
type greyLoadBalance struct {
}

func (lb *greyLoadBalance) Select(context.Context, []registry.ServiceInstance) (registry.ServiceInstance, error) {
	panic("not implememt!")

}

func New() cluster.Selector {
	return &greyLoadBalance{}
}
