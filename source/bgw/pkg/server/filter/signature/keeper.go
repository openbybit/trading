package signature

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"gopkg.in/yaml.v3"

	"bgw/pkg/common/constant"
	"bgw/pkg/config_center/nacos"
)

const (
	file = "bgw_private_keys"

	bgwSignatureAlertTitle = "加签私钥配置更新"
)

var (
	errNoPrivateKey = errors.New("no private key")
)

var defaultKeyKeeper keyKeeper

type keyKeeper interface {
	GetAppName() string
	GetSignKey(context.Context, string) (string, error)
	buildListen(ctx context.Context) error
}

func newPrivateKeyKeeper() keyKeeper {
	return &privateKeyKeeper{
		appKey:  make(map[string]string),
		signKey: make(map[string]string),
	}
}

type signatureCfg struct {
	AppName string   `yaml:"app_name"`
	AppKeys []appKey `yaml:"app_key"`
}

type appKey struct {
	AppID string `yaml:"app_id"`
	Key   string `yaml:"key"`
}

type privateKeyKeeper struct {
	lock    sync.RWMutex
	appName string
	appKey  map[string]string // appKey contains private keys encrypted by sechub
	signKey map[string]string // signKey contains original private keys
}

// GetSignKey build sign key by app id
func (p *privateKeyKeeper) GetSignKey(ctx context.Context, appID string) (string, error) {
	p.lock.RLock()
	signKey, ok := p.signKey[appID]
	if ok {
		p.lock.RUnlock()
		return signKey, nil
	}

	appkey, ok := p.appKey[appID]
	p.lock.RUnlock()
	if !ok {
		return "", errNoPrivateKey
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	signKey, ok = p.signKey[appID]
	if ok {
		return signKey, nil
	}
	signKey, err := gsechub.Decrypt(appkey)
	if err != nil {
		glog.Error(ctx, "sechub get sign key failed", glog.String("err", err.Error()))
		return "", err
	}

	p.signKey[appID] = signKey

	return signKey, nil
}

// GetAppName get app name
func (p *privateKeyKeeper) GetAppName() string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.appName
}

// OnEvent nacos config callback
func (p *privateKeyKeeper) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}

	cfg := &signatureCfg{
		AppKeys: make([]appKey, 0),
	}

	if err := yaml.Unmarshal([]byte(e.Value), cfg); err != nil {
		msg := fmt.Sprintf("privateKey listener error, err = %s, EventKey = %s", err.Error(), e.Key)
		galert.Error(context.Background(), msg)
		return nil
	}
	if cfg.AppName == "" || len(cfg.AppKeys) == 0 {
		msg := fmt.Sprintf("privateKey config is invalid, EventKey = %s", e.Key)
		galert.Error(context.Background(), msg)
		return nil
	}

	p.build(cfg)
	msg := fmt.Sprintf("privateKey config build success, EventKey = %s", e.Key)
	galert.Info(context.Background(), msg)

	return nil
}

// GetEventType get event type
func (p *privateKeyKeeper) GetEventType() reflect.Type {
	return nil
}

// GetPriority get priority
func (p *privateKeyKeeper) GetPriority() int {
	return 0
}

func (p *privateKeyKeeper) build(cfg *signatureCfg) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.appName = cfg.AppName
	for _, app := range cfg.AppKeys {
		if _, ok := p.appKey[app.AppID]; ok {
			continue
		}
		p.appKey[app.AppID] = app.Key
		signKey, err := gsechub.Decrypt(app.Key)
		if err != nil {
			glog.Error(context.Background(), "sechub get sign key failed", glog.String("err", err.Error()))
			continue
		}

		p.signKey[app.AppID] = signKey
	}
}

func (p *privateKeyKeeper) buildListen(ctx context.Context) error {
	// build nacos config client
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	)
	if err != nil {
		glog.Error(ctx, "privatekey listen NewNacosConfigure error", glog.String("error", err.Error()))
		return err
	}

	// listen nacos config
	if err = nacosCfg.Listen(ctx, file, p); err != nil {
		galert.Error(ctx, fmt.Sprintf("privatekey listen error, err = %s, file = %s", err.Error(), file), galert.WithTitle(bgwSignatureAlertTitle))
		return err
	}

	return nil
}
