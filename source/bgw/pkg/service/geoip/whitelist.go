package geoip

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"gopkg.in/yaml.v3"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common/constant"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

var (
	wl     Whitelist
	wlOnce sync.Once
)

// Whitelist check if ip in the ip white list.
type Whitelist interface {
	Check(ctx context.Context, ip string) bool
	Query(ctx context.Context) []string
}

type whitelist struct {
	mutex sync.RWMutex
	ips   map[string]struct{}

	nacos config_center.Configure
}

func CheckIPWhitelist(ctx context.Context, ip string) bool {
	err := InitIPWhitelist(ctx)
	if err != nil || wl == nil {
		return false
	}

	res := wl.Check(ctx, ip)
	if res {
		gmetric.IncDefaultCounter("ip_whitelist", ip)
	}
	return res
}

func InitIPWhitelist(ctx context.Context) error {
	if wl != nil {
		return nil
	}

	var err error
	wlOnce.Do(func() {
		wl, err = newIPWhitelist(ctx)
		if err != nil {
			msg := fmt.Sprintf("ip whitelist init error, err = %s", err.Error())
			galert.Error(ctx, msg, galert.WithTitle("ip whitelist"))
			return
		}
		registerWLAdmin()
	})

	return err
}

func newIPWhitelist(ctx context.Context) (Whitelist, error) {
	wl := &whitelist{
		ips: make(map[string]struct{}),
	}
	err := wl.buildListen(ctx)
	if err != nil {
		return nil, err
	}
	return wl, nil
}

func (w *whitelist) Check(ctx context.Context, ip string) bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	_, ok := w.ips[ip]
	return ok
}

func (w *whitelist) Query(ctx context.Context) []string {
	res := make([]string, 0)
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for ip := range w.ips {
		res = append(res, ip)
	}

	return res
}

const file = "bgw-ip-whitelist"

func (w *whitelist) buildListen(ctx context.Context) error {
	// build nacos config client
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	)
	if err != nil {
		glog.Error(ctx, "ip whitelist new nacos configure error", glog.String("error", err.Error()))
		return err
	}
	w.nacos = nacosCfg

	// listen nacos config
	if err = nacosCfg.Listen(ctx, file, w); err != nil {
		glog.Error(ctx, "ip whitelist nacos listen error", glog.String("error", err.Error()))
		return err
	}

	return nil
}

func (w *whitelist) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}

	list := make([]string, 0)
	if err := yaml.Unmarshal([]byte(e.Value), &list); err != nil {
		msg := fmt.Sprintf("ip whitelist on event error, err = %s, EventKey = %s", err.Error(), e.Key)
		galert.Error(context.Background(), msg)
		return nil
	}

	ips := make(map[string]struct{})
	for _, ip := range list {
		ips[ip] = struct{}{}
	}

	glog.Info(context.Background(), "ip white list update success", glog.Int("length", len(ips)))

	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.ips = ips
	return nil
}

// GetEventType get event type
func (w *whitelist) GetEventType() reflect.Type {
	return nil
}

// GetPriority get priority
func (w *whitelist) GetPriority() int {
	return 0
}

func registerWLAdmin() {
	if wl == nil {
		return
	}
	// curl 'http://localhost:6480/admin?cmd=queryIPWhiteList'
	gapp.RegisterAdmin("queryIPWhiteList", "query ip white list", OnQueryIPWhitelist)
}

func OnQueryIPWhitelist(args gapp.AdminArgs) (interface{}, error) {
	return wl.Query(context.Background()), nil
}
