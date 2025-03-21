package zone_roundrobin

import (
	"context"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/dynconfig"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func init() {
	cluster.Register(constant.SelectorZoneRoundRobin, New())
}

type zone struct {
	idLoader *dynconfig.GrayIdsListLoader
}

func New() cluster.Selector {
	return &zone{
		idLoader: dynconfig.NewZoneMemberIDGrayListLoader(),
	}
}

func (lb *zone) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	md := metadata.MDFromContext(ctx)
	id := md.GetPartitionID()
	if id <= 0 {
		glog.Debug(ctx, "zone_round_robin uid is 0")
		return nil, cluster.ErrServiceNotFound
	}

	selector := cluster.GetSelector(ctx, constant.SelectorRoundRobin)

	vZone := lb.idLoader.CheckZone(id)
	if vZone != "" {
		glog.Debug(ctx, "zone_round_robin uid hit gray zone", glog.String("zone", vZone), glog.Int64("uid", id))
		instance, err := selector.Select(ctx, lb.selectByZoneName(vZone, ins))
		if err != nil {
			return nil, err
		}
		if instance != nil {
			return instance, nil
		}
	}

	var partition int
	zc, err := dynconfig.GetZoneConfig()
	if err != nil {
		return nil, err
	}
	if md.IsDemoUID {
		partition = int(id % zc.GetDemoZonePartition())
	} else {
		partition = int(id % zc.GetCommonZonePartition())
	}
	glog.Debug(ctx, "zone_round_robin",
		glog.Int64("zone_round_robin", int64(partition)),
		glog.Int64("demo_cap", zc.GetDemoZonePartition()), glog.Int64("cap", zc.GetCommonZonePartition()))

	var instances []registry.ServiceInstance
	for _, in := range ins {
		if in.GetMetadata().GetPartition() == partition {
			instances = append(instances, in)
		}
	}

	return selector.Select(ctx, instances)
}

func (lb *zone) selectByZoneName(zone string, ins []registry.ServiceInstance) []registry.ServiceInstance {
	var instances []registry.ServiceInstance
	for _, in := range ins {
		if name := in.GetMetadata().GetZoneName(); name == zone {
			instances = append(instances, in)
		}
	}
	return instances
}

func (lb *zone) Init(ctx context.Context) error {
	if err := dynconfig.InitZoneMemberIDGrayListLoader(ctx); err != nil {
		return err
	}
	return nil
}
