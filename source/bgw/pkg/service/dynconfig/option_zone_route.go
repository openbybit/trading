package dynconfig

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center/nacos"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

const (
	ZoneConfig     = "option-zone.yaml"
	ZoneDemoConfig = "option-zone-demo.yaml"
)

var zoneInit sync.Once
var zoneConfig *OptionZoneConfig

type OptionZoneConfig struct {
	zoneNumber     int64
	demoZoneNumber int64
}

func newOptionZoneConfig() *OptionZoneConfig {
	return &OptionZoneConfig{
		zoneNumber:     10,
		demoZoneNumber: 1,
	}
}

// GetCommonZonePartition get common zone partition
func (o *OptionZoneConfig) GetCommonZonePartition() int64 {
	return o.zoneNumber
}

// GetDemoZonePartition get demo zone partition
func (o *OptionZoneConfig) GetDemoZonePartition() int64 {
	return o.demoZoneNumber
}

// GetZoneConfig get zone config
func GetZoneConfig() (*OptionZoneConfig, error) {
	var err error
	zoneInit.Do(func() {
		namespace := config.GetNamespace()
		if env.IsProduction() {
			namespace = constant.DEFAULT_NAMESPACE
		}
		nc, e := nacos.NewNacosConfigure(
			context.TODO(),
			nacos.WithGroup(RouteStrategy), // specified group
			nacos.WithNameSpace(namespace), // namespace isolation
		)
		if e != nil {
			err = fmt.Errorf("OptionZoneConfig NewNacosConfigure error: %w", e)
			return
		}

		zoneConfig = newOptionZoneConfig()

		if e := nc.Listen(context.Background(), ZoneConfig, zoneConfig); e != nil {
			err = fmt.Errorf("OptionZoneConfig watch %s error: %w", ZoneConfig, e)
			return
		}
		if e := nc.Listen(context.Background(), ZoneDemoConfig, zoneConfig); e != nil {
			err = fmt.Errorf("OptionZoneConfig watch %s error: %w", ZoneDemoConfig, e)
			return
		}

		glog.Info(context.TODO(), fmt.Sprintf("OptionZoneConfig nacos listener init success: %s %s %s %s", namespace, RouteStrategy, ZoneConfig, ZoneDemoConfig))
	})
	if err != nil {
		galert.Error(context.TODO(), "GetZoneConfig error", galert.WithField("err", err))
		glog.Error(context.TODO(), "GetZoneConfig error", glog.NamedError("err", err))
		return nil, err
	}

	return zoneConfig, nil
}

type zoneInfo struct {
	Business      string `yaml:"business"`
	ShardStrategy string `yaml:"shardStrategy"`
	Number        int64  `yaml:"number"`
}

// OnEvent on event
func (o *OptionZoneConfig) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	glog.Info(context.TODO(), "OptionZoneConfig OnEvent", glog.String("key", re.Key), glog.String("content", re.Value))
	if re.Value == "" {
		return nil
	}

	cfg := zoneInfo{}
	if err := util.YamlUnmarshalString(re.Value, &cfg); err != nil {
		glog.Error(context.TODO(), "OptionZoneConfig YamlUnmarshalString error", glog.String("file", re.Key), glog.NamedError("err", err))
		galert.Error(context.TODO(), "OptionZoneConfig YamlUnmarshalString error", galert.WithField("file", re.Key), galert.WithField("err", err))
		return nil
	}

	if cfg.Number <= 0 {
		glog.Error(context.TODO(), "OptionZoneConfig invalid zone number", glog.String("file", re.Key))
		galert.Error(context.TODO(), "OptionZoneConfig invalid zone number", galert.WithField("file", re.Key))
		return fmt.Errorf("invalid zone number: %s", re.Key)
	}

	if re.Key == ZoneDemoConfig {
		atomic.StoreInt64(&o.demoZoneNumber, cfg.Number)
		galert.Info(context.TODO(), "OptionZoneConfig update success", galert.WithField("file", re.Key), galert.WithField("demoZoneNumber", o.demoZoneNumber))
		return nil
	}

	atomic.StoreInt64(&o.zoneNumber, cfg.Number)
	galert.Info(context.TODO(), "OptionZoneConfig update success", galert.WithField("file", re.Key), galert.WithField("zoneNumber", o.zoneNumber))
	return nil
}

// GetEventType remoting watch event
func (o *OptionZoneConfig) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority get priority
func (o *OptionZoneConfig) GetPriority() int {
	return -1
}
