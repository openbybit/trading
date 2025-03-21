package dynconfig

import (
	"context"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

const (
	VipZoneConfig = "option-vip-zone"
)

var (
	vipIPWaiteListLoader         = &VIPIdsListLoader{}
	vipIPWaiteListLoaderOnceInit sync.Once
)

type VIPIdsListLoader struct {
	ctx       context.Context
	configure config_center.Configure
	vipIDs    sync.Map // vip, id --> zone
}

func NewZoneMemberIDVIPListLoader() *VIPIdsListLoader {
	return vipIPWaiteListLoader
}

func InitZoneMemberIDVIPListLoader(ctx context.Context) (err error) {
	vipIPWaiteListLoaderOnceInit.Do(func() {
		glog.Info(ctx, "InitZoneMemberIDVIPListLoader receive start signal")

		namespace := config.GetNamespace()
		if env.IsProduction() {
			namespace = constant.DEFAULT_NAMESPACE
		}
		nc, e := nacos.NewNacosConfigure(
			ctx,
			nacos.WithGroup(RouteStrategy), // specified group
			nacos.WithNameSpace(namespace), // namespace isolation
		)
		if e != nil {
			err = e
			return
		}

		vipIPWaiteListLoader.configure = nc
		vipIPWaiteListLoader.ctx = ctx
		if e = nc.Listen(ctx, VipZoneConfig, vipIPWaiteListLoader); e != nil {
			err = e
			glog.Error(ctx, "InitZoneMemberIDVIPListLoader watch error", glog.String("error", e.Error()))
			return
		}

		glog.Info(ctx, "InitZoneMemberIDVIPListLoader init success", glog.String("file", VipZoneConfig))
	})
	if err != nil {
		galert.Error(ctx, "vip zone list init error, "+err.Error())
	}

	return
}

func (ql *VIPIdsListLoader) decode(key, content string) (qvs []idWhiteList) {
	err := util.JsonUnmarshalString(content, &qvs)
	if err != nil {
		glog.Info(ql.ctx, "VIPIdsListLoader JsonUnmarshalString error", glog.String("key", key), glog.String("content", content), glog.String("error", err.Error()))
		return nil
	}
	for _, ds := range qvs {
		if ds.Zone == "" {
			glog.Error(ql.ctx, "VIPIdsListLoader Zone is nil", glog.String("key", key), glog.String("content", content))
			return nil
		}
	}

	return
}

func (ql *VIPIdsListLoader) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	glog.Info(ql.ctx, "VIPIdsListLoader OnEvent", glog.String("key", re.Key), glog.String("content", re.Value))
	if re.Value == "" {
		return nil
	}

	qv := ql.decode(re.Key, re.Value)
	if qv == nil {
		return nil
	}

	// /BGW_data/option/zone_ids/vip/groups
	switch re.Action {
	case observer.EventTypeAdd:
		ql.setVipIDs(qv)
	case observer.EventTypeUpdate:
		ql.setVipIDs(qv)
	case observer.EventTypeDel:
		ql.delVipIDs()
		glog.Warn(ql.ctx, "VIPIdsListLoader delVipIDs", glog.String("key", re.Key))
	}

	return nil
}

func (ql *VIPIdsListLoader) setVipIDs(qvs []idWhiteList) {
	hitMap := make(map[int64]int, 2)
	ql.vipIDs.Range(func(key, value interface{}) bool {
		hitMap[key.(int64)] = 0
		return true
	})

	for _, ds := range qvs {
		for i, id := range ds.WhiteIDS {
			if id <= 0 {
				glog.Error(ql.ctx, "VIPIdsListLoader setVipIDs id <=0", glog.String("zone", ds.Zone), glog.Int64("index", int64(i)))
				continue
			}
			ql.vipIDs.Store(id, ds.Zone)
			hitMap[id] = 1
		}
	}

	for k, v := range hitMap {
		if v == 1 {
			continue
		}
		// delete expired vip ids
		ql.vipIDs.Delete(k)
	}
}

func (ql *VIPIdsListLoader) delVipIDs() {
	m := make(map[int64]struct{})
	ql.vipIDs.Range(func(key, value interface{}) bool {
		m[key.(int64)] = struct{}{}
		return true
	})
	for k := range m {
		ql.vipIDs.Delete(k)
	}
}

// GetEventType remoting watch event
func (ql *VIPIdsListLoader) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority get Priority
func (ql *VIPIdsListLoader) GetPriority() int {
	return -1
}

func (ql *VIPIdsListLoader) CheckZone(uid int64) string {
	value, ok := ql.vipIDs.Load(uid)
	if !ok {
		return ""
	}
	zone, ok := value.(string)
	if !ok {
		glog.Error(ql.ctx, "VIPIdsListLoader CheckZone fail, not string", glog.Any("zone", value), glog.Int64("uid", uid))
		return ""
	}
	return zone
}
