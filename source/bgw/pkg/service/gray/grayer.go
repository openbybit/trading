package gray

import (
	"context"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

// Grayer is a gray config which check gray status.
// bgw灰度功能:
type Grayer interface {
	GrayStatus(ctx context.Context) (gray bool, err error)
	Tag() string
}

func NewGrayer(tag, cluster string) Grayer {
	g := &grayer{
		tag:     tag,
		cluster: cluster,
	}
	globalCfg.RegisterGrayer(g)

	globalCfg.initOnce()
	return g
}

type grayer struct {
	tag     string
	cluster string

	sync.RWMutex
	version  int64
	deadline int64
	sgs      *Strategies
}

func (g *grayer) GrayStatus(ctx context.Context) (bool, error) {
	g.RLock()
	if time.Now().UnixNano() >= g.deadline {
		g.RUnlock()
		return false, nil
	}
	strags := g.sgs
	g.RUnlock()

	return strags.grayCheck(ctx)
}

func (g *grayer) Tag() string {
	return g.tag
}

func (g *grayer) OnCfgChange(cfg *Strategies, deadline int64, v int64) {
	g.Lock()
	defer g.Unlock()
	if v <= g.version {
		return
	}
	g.version = v
	g.sgs = cfg
	g.deadline = deadline
	glog.Debug(context.Background(), "gray on cfg change success", glog.String("tag", g.tag), glog.Int64("version", v))
}
