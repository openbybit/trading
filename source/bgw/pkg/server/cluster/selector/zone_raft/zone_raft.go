package zone_raft

import (
	"context"
	"sync/atomic"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/dynconfig"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func init() {
	cluster.Register(constant.SelectorZoneRaft, New())
}

type zone struct {
	index    uint32
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
		glog.Debug(ctx, "zone_raft uid is 0, will round robin an index")
		id = int64(atomic.AddUint32(&lb.index, 1))
	} else {
		vZone := lb.idLoader.CheckZone(id)
		if vZone != "" {
			glog.Debug(ctx, "zone_raft uid hit gray zone", glog.String("zone", vZone), glog.Int64("uid", id),
				glog.Int64("len", int64(len(ins))))
			return lb.selectZoneInstance(vZone, ins)
		}
	}

	zc, err := dynconfig.GetZoneConfig()
	if err != nil {
		return nil, err
	}
	partition := int(id % zc.GetCommonZonePartition())
	glog.Debug(ctx, "zone_raft",
		glog.Int64("zone_raft", int64(partition)),
		glog.Int64("cap", zc.GetCommonZonePartition()), glog.Int64("len", int64(len(ins))))

	var cur registry.ServiceInstance
	for _, instance := range ins {
		im := instance.GetMetadata() // instance metadata

		if im.GetRole().IsLeader() && partition == im.GetPartition() {
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

func (lb *zone) selectZoneInstance(zone string, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	var cur registry.ServiceInstance
	for _, instance := range ins {
		im := instance.GetMetadata() // instance metadata

		if im.GetRole().IsLeader() && zone == im.GetZoneName() {
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

func (lb *zone) Init(ctx context.Context) error {
	if err := dynconfig.InitZoneMemberIDGrayListLoader(ctx); err != nil {
		return err
	}
	return nil
}
