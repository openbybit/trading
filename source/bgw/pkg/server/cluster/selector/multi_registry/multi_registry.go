package multi_registry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/dynconfig"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func init() {
	cluster.Register(constant.SelectorMultiRegistry, New())
}

var (
	errServiceListNotEnough = fmt.Errorf("service list is not enough")
)

type discoveryFunc = func(*common.URL) []registry.ServiceInstance

type selector struct {
	vipID   *dynconfig.VIPIdsListLoader
	grayID  *dynconfig.GrayIdsListLoader
	index   uint32
	dicover discoveryFunc
	urls    sync.Map // registry -> url
}

func New() cluster.Selector {
	return &selector{
		vipID:  dynconfig.NewZoneMemberIDVIPListLoader(),
		grayID: dynconfig.NewZoneMemberIDGrayListLoader(),
	}
}

func (s *selector) SetDiscovery(f discoveryFunc) {
	s.dicover = f
}

func (s *selector) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	md := metadata.MDFromContext(ctx)
	id := md.GetPartitionID()
	if id > 0 {
		// vip raft
		vZone := s.vipID.CheckZone(id)
		if vZone != "" {
			glog.Debug(ctx, "multi-service uid hit vip zone", glog.String("zone", vZone), glog.Int64("uid", id))
			url := s.urlFromContext(ctx)
			if url == nil {
				return nil, errServiceListNotEnough
			}
			md.InvokeService = url.Addr
			glog.Debug(ctx, "multi-service url", glog.Any("registry", url))

			ins = s.dicover(url)

			return selectZoneInstance(vZone, ins)
		}

		// zone raft
		gZone := s.grayID.CheckZone(id)
		if gZone != "" {
			glog.Debug(ctx, "multi-service uid hit gray zone", glog.String("zone", gZone), glog.Int64("uid", id),
				glog.Int64("len", int64(len(ins))))
			return selectZoneInstance(gZone, ins)
		}
	} else {
		glog.Debug(ctx, "multi-service uid is 0, will round robin an index")
		id = int64(atomic.AddUint32(&s.index, 1))
	}

	zc, err := dynconfig.GetZoneConfig()
	if err != nil {
		return nil, err
	}
	partition := int(id % zc.GetCommonZonePartition())
	glog.Debug(ctx, "multi-service",
		glog.Int64("partition", int64(partition)),
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

type registryMeta struct {
	DuplicateRegistry struct {
		Registry string `json:"registry"`
	} `json:"duplicate_registry,omitempty"`
}

func (s *selector) Extract(meta *cluster.ExtractConf) (interface{}, error) {
	registries := &registryMeta{}
	err := util.JsonUnmarshal([]byte(meta.LoadBalanceMeta), registries)
	if err != nil {
		return nil, err
	}
	if registries.DuplicateRegistry.Registry == "" {
		return nil, fmt.Errorf("duplicate_registry service name is empty")
	}
	ser := &service{
		Registry:  registries.DuplicateRegistry.Registry,
		Group:     meta.Group,
		Namespace: meta.Namespace,
	}
	return ser, nil
}

func (s *selector) Inject(ctx context.Context, metas interface{}) (context.Context, error) {
	if metas == nil {
		return ctx, fmt.Errorf("nothing to inject")
	}
	ser, ok := metas.(*service)
	if !ok || ser.Registry == "" {
		return nil, fmt.Errorf("duplicate_registry Registry is empty")
	}

	return contextWithSelectMetas(ctx, ser), nil
}

type service struct {
	Registry  string `json:"registry"`
	Group     string `json:"group"`
	Namespace string `json:"namespace"`
}

const multiServiceMetaKey = "multi-service-meta-key"

type multiServiceMeat struct{}

func (s *selector) urlFromContext(ctx context.Context) *common.URL {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(multiServiceMetaKey)
	} else {
		v = ctx.Value(multiServiceMeat{})
	}

	if v == nil {
		return nil
	}

	ser, ok := v.(*service)
	if !ok {
		return nil
	}

	value, ok := s.urls.Load(ser.Registry)
	if !ok {
		url, err := common.NewURL(ser.Registry, common.WithProtocol(constant.NacosProtocol),
			common.WithGroup(ser.Group), common.WithNamespace(ser.Namespace))
		if err != nil {
			glog.Debug(ctx, "multi-service get vip url failed", glog.String("err", err.Error()))
			return nil
		}
		s.urls.Store(ser.Registry, url)
		return url
	}
	url, ok := value.(*common.URL)
	if ok {
		return url
	}
	return nil
}

func contextWithSelectMetas(ctx context.Context, origin *service) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(multiServiceMetaKey, origin)
	} else {
		return context.WithValue(ctx, multiServiceMeat{}, origin)
	}
	return nil
}

func selectZoneInstance(zone string, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
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
