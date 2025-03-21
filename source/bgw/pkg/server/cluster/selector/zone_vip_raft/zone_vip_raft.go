package zone_vip_raft

import (
	"context"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/dynconfig"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func init() {
	cluster.Register(constant.SelectorZoneVIPRaft, New())
}

type zone struct {
	idLoader *dynconfig.VIPIdsListLoader
}

func New() cluster.Selector {
	return &zone{
		idLoader: dynconfig.NewZoneMemberIDVIPListLoader(),
	}
}

func (lb *zone) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	md := metadata.MDFromContext(ctx)
	id := md.GetPartitionID()
	if id <= 0 {
		glog.Debug(ctx, "zone_vip_raft uid is 0")
		return nil, cluster.ErrServiceNotFound
	}

	vZone := lb.idLoader.CheckZone(id)
	if vZone == "" {
		glog.Error(ctx, "zone_vip_raft uid not hit vip zone", glog.Int64("uid", id))
		return nil, berror.ErrZoneVIPSelectorInvalidUID
	}
	glog.Debug(ctx, "zone_vip_raft uid hit vip zone", glog.String("zone", vZone), glog.Int64("uid", id))

	var cur registry.ServiceInstance
	for _, instance := range ins {
		im := instance.GetMetadata() // instance metadata

		if im.GetRole().IsLeader() && vZone == im.GetZoneName() {
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
	if err := dynconfig.InitZoneMemberIDVIPListLoader(ctx); err != nil {
		return err
	}
	return nil
}
