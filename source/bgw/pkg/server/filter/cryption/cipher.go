package cryption

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"code.bydev.io/public-lib/sec/sec-sign.git/cipher"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"

	"bgw/pkg/common/constant"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

const (
	group         = "security"
	namespace     = ""
	cipherCfgFile = "sec_sign"
	grayCfgFile   = "sec_sign_gray"
)

var (
	globalCipher *Cipher
	once         sync.Once
)

func getCipher(ctx context.Context) *Cipher {
	_ = initCipher(ctx)
	return globalCipher
}

func initCipher(ctx context.Context) error {
	if globalCipher != nil {
		return nil
	}

	var err error
	once.Do(func() {
		globalCipher, err = newCipher(ctx)
		if err != nil {
			galert.Error(ctx, fmt.Sprintf("ciper init failed, err = %s", err.Error()))
		}
	})

	return err
}

func newCipher(ctx context.Context) (*Cipher, error) {
	c, _ := cipher.NewCipher(&cipher.Config{})
	C := &Cipher{
		Cipher: c,
		grayer: &grayer{},
	}
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(group),
		nacos.WithNameSpace(namespace),
	)

	if err != nil {
		glog.Error(ctx, "cipher nacos configure error", glog.String("err", err.Error()))
		return C, err
	}

	if err = nacosCfg.Listen(ctx, cipherCfgFile, C); err != nil {
		glog.Error(ctx, "cipher cfg listen error", glog.String("err", err.Error()))
		return C, err
	}

	nacosCfg1, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.DEFAULT_GROUP),
		nacos.WithNameSpace(constant.BGWConfigNamespace),
	)

	if err != nil {
		glog.Error(ctx, "cipher nacos configure error", glog.String("err", err.Error()))
		return C, err
	}

	if err = nacosCfg1.Listen(ctx, grayCfgFile, C); err != nil {
		glog.Error(ctx, "cipher gray cfg listen error", glog.String("err", err.Error()))
		return C, err
	}

	C.nacosCli = nacosCfg
	C.nacosCli1 = nacosCfg1
	return C, nil
}

type Cipher struct {
	*cipher.Cipher
	observer.EmptyListener
	nacosCli  config_center.Configure
	nacosCli1 config_center.Configure
	*grayer
}

func (c *Cipher) OnEvent(event observer.Event) error {
	glog.Debug(context.Background(), "in cipher on event")
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		glog.Debug(context.Background(), "cipher event type no ok")
		return nil
	}
	if e.Value == "" {
		glog.Debug(context.Background(), "cipher empty value", glog.String("v", e.Value))
		return nil
	}

	if e.Key == cipherCfgFile {
		return c.updateCipherCfg(e.Value)
	}

	return c.updateGrayCfg(e.Value)
}

func (c *Cipher) updateCipherCfg(value string) error {
	cfg := cipher.Config{}
	if err := jsoniter.Unmarshal([]byte(value), &cfg); err != nil {
		msg := fmt.Sprintf("crypter update config failed, err = %s", err.Error())
		galert.Error(context.Background(), msg)
		return nil
	}
	glog.Debug(context.Background(), "cryption cfgs", glog.Any("key", cfg))

	for i, _ := range cfg.SignKey {
		k, err := gsechub.Decrypt(cfg.SignKey[i].XORKey)
		if err == nil {
			cfg.SignKey[i].XORKey = k
		} else {
			galert.Error(context.Background(), fmt.Sprintf("crypter XORKey decrypt failed, err = %s", err.Error()))
		}
		k, err = gsechub.Decrypt(cfg.SignKey[i].RSAPrivateKey)
		if err == nil {
			cfg.SignKey[i].RSAPrivateKey = k
		} else {
			galert.Error(context.Background(), fmt.Sprintf("crypter RSAPrivateKey decrypt failed, err = %s", err.Error()))
		}
	}

	err := c.Refresh(&cfg)
	if err != nil {
		msg := fmt.Sprintf("crypter s, EventKey = %s", cipherCfgFile)
		galert.Error(context.Background(), msg)
	}

	glog.Debug(context.Background(), "crypter key", glog.Any("key", cfg))
	glog.Info(context.Background(), "crypter config update")
	return nil
}

func (c *Cipher) updateGrayCfg(value string) error {
	cfg := grayCfg{}
	if err := yaml.Unmarshal([]byte(value), &cfg); err != nil {
		msg := fmt.Sprintf("crypter update gray config failed, err = %s", err.Error())
		galert.Error(context.Background(), msg)
		return nil
	}

	glog.Info(context.Background(), "cryption gray cfgs", glog.Any("cfg", cfg))
	c.set(cfg.Status, cfg.FullOn, cfg.Tails)
	return nil
}

// 临时的灰度配置，只支持按用户尾号灰度，灰度的粒度通过位数控制，如1，111，111111
type grayer struct {
	status atomic.Int32 // 灰度配置总开关，0表示不生效， 1表示灰度生效
	fullOn atomic.Bool  // 增加全量灰度的字段，如果为true，则表示所有用户都在灰度中

	sync.RWMutex
	tails []int64 // 灰度的尾号
}

type grayCfg struct {
	Status int32
	FullOn bool `yaml:"full_on"`
	Tails  []int64
}

func (g *grayer) set(status int32, fullOn bool, tails []int64) {
	g.Lock()
	g.tails = tails
	g.status.Store(status)
	g.fullOn.Store(fullOn)
	g.Unlock()
}

func (g *grayer) check(uid int64) bool {
	if g.status.Load() == 0 {
		return false
	}

	if g.fullOn.Load() {
		return true
	}

	g.RLock()
	defer g.RUnlock()
	for _, tail := range g.tails {
		if tailMatch(uid, tail) {
			return true
		}
	}
	return false
}

func tailMatch(uid, tail int64) bool {
	if tail > uid {
		return false
	}

	if tail == 0 {
		re := uid % 10
		if re == 0 {
			return true
		}
	}

	for tail > 0 {
		t1 := tail % 10
		t2 := uid % 10
		if t1 != t2 {
			return false
		}
		tail = tail / 10
		uid = uid / 10
	}

	return true
}
