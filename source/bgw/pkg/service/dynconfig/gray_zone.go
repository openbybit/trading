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
	RouteStrategy  = "ROUTE_STRATEGY"
	GrayZoneConfig = "option-gray-zone"
)

var (
	grayIPWaiteListLoader         = &GrayIdsListLoader{}
	grayIPWaiteListLoaderOnceInit sync.Once
)

type idWhiteList struct {
	Zone     string  `json:"zone"`
	WhiteIDS []int64 `json:"white_ids"`
}

type GrayIdsListLoader struct {
	ctx       context.Context
	configure config_center.Configure
	grayIDs   sync.Map // gray, id --> zone
}

func NewZoneMemberIDGrayListLoader() *GrayIdsListLoader {
	return grayIPWaiteListLoader
}

func InitZoneMemberIDGrayListLoader(ctx context.Context) (err error) {
	grayIPWaiteListLoaderOnceInit.Do(func() {
		glog.Info(ctx, "InitZoneMemberIDGrayListLoader receive start signal")

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

		grayIPWaiteListLoader.configure = nc
		grayIPWaiteListLoader.ctx = ctx
		if e = nc.Listen(ctx, GrayZoneConfig, grayIPWaiteListLoader); e != nil {
			err = e
			glog.Error(ctx, "InitZoneMemberIDGrayListLoader watch error", glog.String("error", e.Error()))
			return
		}

		glog.Info(ctx, "InitZoneMemberIDGrayListLoader init success", glog.String("file", GrayZoneConfig))
	})
	if err != nil {
		galert.Error(ctx, "gray zone list init error, "+err.Error())
	}

	return
}

func (ql *GrayIdsListLoader) decode(key, content string) (qvs []idWhiteList) {
	err := util.JsonUnmarshalString(content, &qvs)
	if err != nil {
		glog.Info(ql.ctx, "GrayIdsListLoader JsonUnmarshalString error", glog.String("key", key), glog.String("content", content), glog.String("error", err.Error()))
		return nil
	}
	for _, ds := range qvs {
		if ds.Zone == "" {
			glog.Error(ql.ctx, "GrayIdsListLoader Zone is nil", glog.String("key", key), glog.String("content", content))
			return nil
		}
	}

	return
}

func (ql *GrayIdsListLoader) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	glog.Info(ql.ctx, "GrayIdsListLoader OnEvent", glog.String("key", re.Key), glog.String("content", re.Value))
	if re.Value == "" {
		return nil
	}

	qv := ql.decode(re.Key, re.Value)
	if qv == nil {
		return nil
	}

	// /BGW_data/option/zone_ids/gray/groups
	switch re.Action {
	case observer.EventTypeAdd:
		ql.setGrayIDs(qv)
	case observer.EventTypeUpdate:
		ql.setGrayIDs(qv)
	case observer.EventTypeDel:
		ql.delGrayIDs()
		glog.Warn(ql.ctx, "GrayIdsListLoader delGrayIDs", glog.String("key", re.Key))
	}

	return nil
}

func (ql *GrayIdsListLoader) setGrayIDs(qvs []idWhiteList) {
	hitMap := make(map[int64]int, 2)
	ql.grayIDs.Range(func(key, value interface{}) bool {
		hitMap[key.(int64)] = 0
		return true
	})

	for _, ds := range qvs {
		for i, id := range ds.WhiteIDS {
			if id <= 0 {
				glog.Error(ql.ctx, "GrayIdsListLoader setGrayIDs id <=0", glog.String("zone", ds.Zone), glog.Int64("index", int64(i)))
				continue
			}
			ql.grayIDs.Store(id, ds.Zone)
			hitMap[id] = 1
		}
	}

	for k, v := range hitMap {
		if v == 1 {
			continue
		}
		// delete expired gray ids
		ql.grayIDs.Delete(k)
	}
}

func (ql *GrayIdsListLoader) delGrayIDs() {
	m := make(map[int64]struct{})
	ql.grayIDs.Range(func(key, value interface{}) bool {
		m[key.(int64)] = struct{}{}
		return true
	})
	for k := range m {
		ql.grayIDs.Delete(k)
	}
}

// GetEventType remoting watch event
// nolint
func (ql *GrayIdsListLoader) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority
// nolint
func (ql *GrayIdsListLoader) GetPriority() int {
	return -1
}

func (ql *GrayIdsListLoader) CheckZone(uid int64) string {
	value, ok := ql.grayIDs.Load(uid)
	if !ok {
		return ""
	}
	zone, ok := value.(string)
	if !ok {
		glog.Error(ql.ctx, "GrayIdsListLoader CheckZone fail, not string", glog.Any("zone", value), glog.Int64("uid", uid))
		return ""
	}
	return zone
}
