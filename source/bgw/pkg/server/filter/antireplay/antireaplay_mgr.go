package antireplay

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"git.bybit.com/codesec/sechub-sdk-go/api"
	"git.bybit.com/codesec/sechub-sdk-go/client"
	"github.com/hashicorp/go-version"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

const (
	XBTimestamp = "X-B-Timestamp"
	XBAccessKey = "X-B-AccessKeyId"
	XBSignature = "X-B-Signature"

	bgwAntiReplayAlertTitle = "防重放配置更新"

	antiReplayFile = "bgwh_anti_replay.yaml"
)

type AntiReplay interface {
	Init(ctx context.Context) error
	GetAntiReplayDiffTime() (int64, int64)
	EnableSign() bool
	EnableAntiReplay() bool
	VerifyAccessKey(ctx context.Context, appName, platform, accessKey string, version *version.Version) (string, error)
}

type antiReplayMgr struct {
	once             sync.Once
	listener         config_center.Configure
	secHubCli        *api.Sechub
	mux              sync.RWMutex
	enableAntiReplay bool
	enableSign       bool
	antiReplayDiff   int64 // ms
	fastThanServer   int64 // ms
	secretCache      map[string]secretWithConstraints
}

type secretWithConstraints struct {
	constraints version.Constraints
	secret      string
}

func newAntiReplayMgr() AntiReplay {
	return &antiReplayMgr{secretCache: make(map[string]secretWithConstraints), enableSign: true, enableAntiReplay: true,
		antiReplayDiff: 30 * time.Second.Milliseconds(),
		fastThanServer: 5 * time.Second.Milliseconds(),
	}
}

// Init antiReplay manager init
func (a *antiReplayMgr) Init(ctx context.Context) (err error) {
	a.once.Do(func() {
		sechubCfg := &config.Global.SechubCfg
		clientCfg := &client.Config{
			Host:       sechubCfg.Address,
			AppName:    sechubCfg.GetOptions("app_name", ""),
			AppSignKey: sechubCfg.GetOptions("app_sign_key", ""),
			TLSCert:    sechubCfg.GetOptions("tls_cert", ""),
		}
		a.secHubCli = api.NewClient(clientCfg)

		nc, e := nacos.NewNacosConfigure(
			context.Background(),
			nacos.WithGroup(constant.BGW_GROUP),              // specified group
			nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
		)
		if e != nil {
			err = e
			return
		}

		a.listener = nc

		if e := nc.Listen(ctx, antiReplayFile, a); e != nil {
			err = e
			return
		}
		glog.Info(ctx, "antiReplayMgr init ok")
	})
	if err != nil {
		msg := fmt.Sprintf("antiReplayMgr init error, err = %s", err.Error())
		galert.Error(ctx, msg)
	}
	return
}

type antiReplayConfig struct {
	AntiReplayDiff   int          `json:"anti_replay_diff" yaml:"anti_replay_diff"`
	FastThanServer   int          `json:"fast_than_server" yaml:"fast_than_server"`
	EnableAntiReplay bool         `json:"enable_anti_replay" yaml:"enable_anti_replay"`
	EnableSign       bool         `json:"enable_sign" yaml:"enable_sign"`
	SecretDatas      []SecretData `json:"secret_datas" yaml:"secret_datas"`
}

type SecretData struct {
	AppName   string `json:"app_name" yaml:"app_name"`
	AccessKey string `json:"access_key" yaml:"access_key"`
	Secret    string `json:"secret" yaml:"secret"`
	Platform  string `json:"platform" yaml:"platform"`
	Version   string `json:"version" yaml:"version"`
}

func (a *antiReplayMgr) buildSecretInfo(data string) error {
	var cfg antiReplayConfig
	if err := util.YamlUnmarshalString(data, &cfg); err != nil {
		glog.Error(context.TODO(), "antiReplayMgr YamlUnmarshalString error", glog.String("error", err.Error()))
		return err
	}

	if cfg.AntiReplayDiff < 5 {
		return fmt.Errorf("AntiReplayDiff too low, must >= 5s, current:%d", cfg.AntiReplayDiff)
	}
	if cfg.FastThanServer < 1 {
		cfg.FastThanServer = 1 // 1s
	}

	secret := make(map[string]secretWithConstraints)
	for _, secretData := range cfg.SecretDatas {
		if secretData.Secret == "" || secretData.AccessKey == "" || secretData.AppName == "" || secretData.Platform == "" || secretData.Version == "" {
			return fmt.Errorf("invalid AntiReplay config")
		}

		constraints, err := version.NewConstraint(secretData.Version)
		if err != nil {
			return fmt.Errorf("invalid AntiReplay version:%s, %s", secretData.Version, err.Error())
		}

		key := a.getKey(secretData.AppName, secretData.Platform, secretData.AccessKey)
		accessSecret, err := a.secHubCli.DecryptData(secretData.Secret)
		if err != nil {
			return fmt.Errorf("AntiReplay config secHubCli DecryptData error, %s -> %s", key, err.Error())
		}
		secret[key] = secretWithConstraints{
			constraints: constraints,
			secret:      accessSecret,
		}
	}

	a.mux.Lock()
	defer a.mux.Unlock()
	a.enableSign = cfg.EnableSign
	a.enableAntiReplay = cfg.EnableAntiReplay
	a.antiReplayDiff = int64(cfg.AntiReplayDiff) * time.Second.Milliseconds()
	a.fastThanServer = int64(cfg.FastThanServer) * time.Second.Milliseconds()
	a.secretCache = secret
	glog.Info(context.TODO(), "antiReplayMgr config", glog.Bool("enableSign", a.enableSign), glog.Bool("anti-reply", a.enableAntiReplay),
		glog.Int64("antiReplayDiff", a.antiReplayDiff), glog.Int64("fastThanServer", a.fastThanServer))

	return nil
}

func (a *antiReplayMgr) getKey(appName, platform, accessKey string) string {
	return fmt.Sprintf("%s:%s:%s", appName, platform, accessKey)
}

// VerifyAccessKey verify access key
func (a *antiReplayMgr) VerifyAccessKey(ctx context.Context, appName, platform, accessKey string, version *version.Version) (string, error) {

	key := a.getKey(appName, platform, accessKey)

	a.mux.RLock()
	defer a.mux.RUnlock()

	data, ok := a.secretCache[key]
	if !ok {
		glog.Debug(ctx, "antiReplayMgr secretCache config is empty", glog.String("key", key))
		return "", fmt.Errorf("antiReplayMgr config is empty, %s", key)
	}

	check := data.constraints.Check(version)
	if !check {
		glog.Debug(ctx, "antiReplayMgr secretCache version check error", glog.String("key", key), glog.String("version", version.String()))
		return "", fmt.Errorf("antiReplayMgr version check error, %s", key)
	}
	return data.secret, nil
}

// GetAntiReplayDiffTime get anti replay diff time
func (a *antiReplayMgr) GetAntiReplayDiffTime() (int64, int64) {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.antiReplayDiff, a.fastThanServer
}

// EnableSign enable verify
func (a *antiReplayMgr) EnableSign() bool {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.enableSign
}

// EnableAntiReplay enable anti replay
func (a *antiReplayMgr) EnableAntiReplay() bool {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.enableAntiReplay
}

// OnEvent config listen event handle
func (a *antiReplayMgr) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}

	if err := a.buildSecretInfo(e.Value); err != nil {
		msg := fmt.Sprintf("anti replay config update error, err = %s, EventKey = %s", err.Error(), antiReplayFile)
		galert.Error(context.TODO(), msg, galert.WithTitle(bgwAntiReplayAlertTitle))
		return err
	}
	msg := fmt.Sprintf("anti replay config update success, EventKey = %s", antiReplayFile)
	galert.Info(context.TODO(), msg, galert.WithTitle(bgwAntiReplayAlertTitle))

	return nil
}

// GetEventType get event type
func (a *antiReplayMgr) GetEventType() reflect.Type {
	return nil
}

// GetPriority get priority
func (a *antiReplayMgr) GetPriority() int {
	return 0
}
