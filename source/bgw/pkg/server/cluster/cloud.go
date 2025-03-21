package cluster

import (
	"context"
	"fmt"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"gopkg.in/yaml.v3"

	"bgw/pkg/config_center/nacos"
)

const (
	cloudCfgNamespace = "public"
	cloudCfgGroup     = "common-cloud-traffic-setting"

	DegradeCloudNamesKeys = "allow-downgrade-cloud"
)

var (
	cloudCfg *CloudDegrade

	globalCloudCfgDataID = "global-cloud-traffic-setting.yml"
	appCloudCfgDataID    = fmt.Sprintf("app-cloud-traffic-setting-%s.yml", env.ServiceName())
)

func GetDegradeCloudMap() (map[string]struct{}, bool) {
	return cloudCfg.GetDegradeMap()
}

func InitCloudCfg() {
	go doInitCloudCfg()
}

func doInitCloudCfg() {
	ctx := context.Background()
	cloudCfg = &CloudDegrade{
		globalCfg: make(map[string]map[string][]string),
		appCfg:    make(map[string]map[string][]string),
	}
	// build nacos config client
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(cloudCfgGroup),         // specified group
		nacos.WithNameSpace(cloudCfgNamespace), // namespace isolation
	)
	if err != nil {
		msg := fmt.Sprintf("cloud cfg listen NewNacosConfigure error, %s", err.Error())
		galert.Error(ctx, msg)
		return
	}

	// listen nacos config
	if err = nacosCfg.Listen(ctx, globalCloudCfgDataID, cloudCfg); err != nil {
		msg := fmt.Sprintf("cloud cfg listen error, err = %s, file = %s", err.Error(), globalCloudCfgDataID)
		galert.Error(ctx, msg)
		return
	}

	// listen nacos config
	if err = nacosCfg.Listen(ctx, appCloudCfgDataID, cloudCfg); err != nil {
		msg := fmt.Sprintf("cloud cfg listen error, err = %s, file = %s", err.Error(), appCloudCfgDataID)
		galert.Error(ctx, msg)
		return
	}
}

type CloudDegrade struct {
	observer.EmptyListener
	sync.RWMutex
	globalCfg map[string]map[string][]string
	appCfg    map[string]map[string][]string
}

// for ut without mock
var cloud = env.CloudProvider()

func (c *CloudDegrade) GetDegradeMap() (map[string]struct{}, bool) {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	res, ok := c.getAppDegrade(cloud)
	if ok {
		return res, true
	}

	res, ok = c.getGlobalDegrade(cloud)
	return res, ok
}

func (c *CloudDegrade) getAppDegrade(cloud string) (map[string]struct{}, bool) {
	res := make(map[string]struct{})

	ac, ok := c.appCfg[cloud]
	if !ok || ac == nil {
		return res, false
	}

	targets, ok := ac[DegradeCloudNamesKeys]
	if !ok || len(targets) == 0 {
		return res, false
	}

	for _, t := range targets {
		res[t] = struct{}{}
	}

	return res, true
}

func (c *CloudDegrade) getGlobalDegrade(cloud string) (map[string]struct{}, bool) {
	res := make(map[string]struct{})

	ac, ok := c.globalCfg[cloud]
	if !ok || ac == nil {
		return res, false
	}

	targets, ok := ac[DegradeCloudNamesKeys]
	if !ok || len(targets) == 0 {
		return res, false
	}

	for _, t := range targets {
		res[t] = struct{}{}
	}

	return res, true
}

type CloudDegradeRule struct {
	// Cloud is the rule key
	Cloud map[string]map[string][]string
}

// OnEvent nacos config callback
func (c *CloudDegrade) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok || e.Value == "" {
		return nil
	}

	cfg := &CloudDegradeRule{}
	if err := yaml.Unmarshal([]byte(e.Value), cfg); err != nil {
		msg := fmt.Sprintf("cloud degrade listen error, err = %s, EventKey = %s", err.Error(), e.Key)
		galert.Error(context.Background(), msg)
		return err
	}

	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	if e.Key == globalCloudCfgDataID {
		c.globalCfg = cfg.Cloud
	}
	if e.Key == appCloudCfgDataID {
		c.appCfg = cfg.Cloud
	}

	glog.Info(context.Background(), "cloud cfg update", glog.String("key", e.Key), glog.Any("cfg", cfg))

	return nil
}
