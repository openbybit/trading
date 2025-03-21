package gray

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"gopkg.in/yaml.v2"

	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

const (
	grayNamespace = "bgw"
	grayGroup     = "BGW_GROUP"
	grayFile      = "bgw_gray_config_file"

	bgwGrayAlertTitle = "灰度配置更新"

	timeLayOut = "2006-01-02 15:04:05"

	clusterAll = "all"
)

var (
	globalCfg = newGrayCfg()
)

func newGrayCfg() *grayCfg {
	return &grayCfg{
		Tags:    make(map[string]*tagCfg),
		grayers: make([]*grayer, 0),
	}
}

type grayCfg struct {
	observer.EmptyListener
	sync.RWMutex
	once     sync.Once
	Tags     map[string]*tagCfg      // tag => tag Strags
	grayers  []*grayer               // grayers listen config update
	nacosCli config_center.Configure // nacos client to listen remote config
}

type tagCfg struct {
	Version  int64                  // Version show if tagCfg updated
	Deadline string                 // Deadline is absolute useful time for current tag config
	Clusters map[string]*Strategies // cluster => Strategies
	deadline int64                  // deadline is unixnano int64 of Deadline
}

func (g *grayCfg) RegisterGrayer(gr *grayer) {
	g.Lock()
	defer g.Unlock()

	g.updateGrayer(gr)
	g.grayers = append(g.grayers, gr)
	glog.Info(context.Background(), "grayer register success", glog.String("tag", gr.tag))
}

func (g *grayCfg) initOnce() {
	g.once.Do(func() {
		_ = g.doInit()
	})
}

func (g *grayCfg) doInit() error {
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(grayGroup),
		nacos.WithNameSpace(grayNamespace),
	)
	if err != nil {
		msg := fmt.Sprintf("gray config new nacos cli error, err = %s, file = %s", err.Error(), grayFile)
		galert.Error(context.Background(), msg, galert.WithTitle(bgwGrayAlertTitle))
		return err
	}
	g.nacosCli = nacosCfg
	if err = nacosCfg.Listen(context.Background(), grayFile, g); err != nil {
		msg := fmt.Sprintf("gray config listen error, err = %s, file = %s", err.Error(), grayFile)
		galert.Error(context.Background(), msg, galert.WithTitle(bgwGrayAlertTitle))
		return err
	}
	return nil
}

func (g *grayCfg) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}
	// do not log event Value because gray config maybe big. use admin to get gray config instead.
	glog.Info(context.TODO(), "gray config OnEvent", glog.String("key", e.Key))

	data := make(map[string]*tagCfg)
	if err := yaml.Unmarshal([]byte(e.Value), &data); err != nil {
		msg := fmt.Sprintf("gray config unmarshsl failed, err = %s, EventKey = %s", err.Error(), e.Key)
		galert.Error(context.Background(), msg, galert.WithTitle(bgwGrayAlertTitle))
		return nil
	}

	cfgs := make(map[string]*tagCfg)
	for tag, cfg := range data {
		newCfg := &tagCfg{
			Deadline: cfg.Deadline,
			Clusters: make(map[string]*Strategies),
			Version:  cfg.Version,
		}
		for cluster, val := range cfg.Clusters {
			cs := strings.Split(cluster, ",")
			for _, c := range cs {
				newCfg.Clusters[c] = val
			}
		}

		d, err := time.Parse(timeLayOut, cfg.Deadline)
		if err != nil {
			d = time.Now().Add(48 * time.Hour)
		}

		newCfg.deadline = d.UnixNano()
		cfgs[tag] = newCfg
	}

	glog.Debug(context.Background(), "gray config update", glog.Any("cfg", cfgs))

	g.Lock()
	defer g.Unlock()
	g.Tags = cfgs
	for _, gr := range g.grayers {
		g.updateGrayer(gr)
	}
	return nil
}

// updateGrayer must be protected by mutex
func (g *grayCfg) updateGrayer(gr *grayer) {
	tCfg, ok := g.Tags[gr.tag]
	if !ok {
		return
	}
	strags := tCfg.Clusters[gr.cluster]
	// 如果没有配置具体集群的配置，则尝试获取all的配置
	if strags == nil || len(*strags) == 0 {
		strags = tCfg.Clusters[clusterAll]
	}
	gr.OnCfgChange(strags, tCfg.deadline, tCfg.Version)
}
