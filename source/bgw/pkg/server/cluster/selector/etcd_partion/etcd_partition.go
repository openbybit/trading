package etcd_partition

import (
	"context"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func init() {
	cluster.Register(constant.SelectorEtcdPartition, New(1))
}

type etcdPartition struct {
	cap int64
}

func New(cap int64) cluster.Selector {
	return &etcdPartition{
		cap: cap,
	}
}

func (ep *etcdPartition) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	md := metadata.MDFromContext(ctx)
	id := md.UID
	if id == 0 {
		return nil, nil
	}

	partition := int(id % ep.cap)
	glog.Debug(ctx, "etcd_partition", glog.Int64("partition", int64(partition)), glog.Int64("cap", ep.cap))

	var instances []registry.ServiceInstance
	for _, instance := range ins {
		id := instance.GetID()
		glog.Debug(ctx, "etcd_partition", glog.String("id", id))
		instances = append(instances, instance)
	}

	selector := cluster.GetSelector(ctx, constant.SelectorRoundRobin)

	return selector.Select(ctx, instances)
}
