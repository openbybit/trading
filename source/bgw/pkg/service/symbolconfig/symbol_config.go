package symbolconfig

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"bgw/pkg/config"

	"code.bydev.io/fbu/future/sdk.git/pkg/scmeta"
	"code.bydev.io/fbu/future/sdk.git/pkg/scmeta/scsrc"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/lib/bat/core.git/pkg/bat"
	"code.bydev.io/lib/bat/core.git/pkg/blabel"
	"code.bydev.io/lib/bat/core.git/pkg/bstd"
	"code.bydev.io/lib/bat/solutions.git/pkg/bkfk"
)

const (
	ServerName         = "bgw"
	ResultTopicName    = "symbol_config_result"
	ResultAckTopicName = "symbol_config_result_ack"
)

var (
	defaultSymbolModule scmeta.Module
	once                sync.Once
)

// GetSymbolModule get symbol config module
func GetSymbolModule() (scmeta.Module, error) {
	if defaultSymbolModule == nil {
		if err := InitSymbolConfig(); err != nil {
			return nil, fmt.Errorf("InitSymbolConfig error: %w", err)
		}
	}
	return defaultSymbolModule, nil
}

// InitSymbolConfig init symbol config module
func InitSymbolConfig() (err error) {
	var sc scmeta.Module
	once.Do(func() {
		kfs := &config.Global.Kafka
		if kfs == nil || kfs.Address == "" {
			err = fmt.Errorf("remote config of kafka addr is nil, for symbol config")
			return
		}
		ads := strings.Split(kfs.Address, ",")
		if len(ads) == 0 || ads[0] == "" {
			err = fmt.Errorf("remote config of kafka error, for symbol config")
			return
		}
		enableLogResult := cast.StringToBool(kfs.GetOptions("enable_symbol_config_log", ""))

		sc, err = buildSymbolConfig(ads, true, enableLogResult)
		if err != nil {
			return
		}
		defaultSymbolModule = sc
	})
	if err != nil || defaultSymbolModule == nil {
		return fmt.Errorf("SymbolModule is nil: %w", err)
	}

	return err
}

func buildSymbolConfig(addr []string, allBrokerSymbols, resultLog bool) (scmeta.Module, error) {
	ml := bstd.ModuleLifecycle{Context: context.Background(), Blocker: &sync.WaitGroup{}}
	kfk, err := bkfk.New(ml, bkfk.KfkConfig{
		BrokerAddrs: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("bkfk.New error, %s", err.Error())
	}

	scSrcCfg := scsrc.Config{
		ServiceName:        ServerName,
		ResultTopicName:    ResultTopicName,
		ResultAckTopicName: ResultAckTopicName,
		EnableLogResult:    resultLog,
	}
	scSrcModule, err := scSrcCfg.New(ml, kfk, New(nil))
	if err != nil {
		return nil, fmt.Errorf("scSrcCfg.New error, %s", err.Error())
	}

	scMetaCfg := scmeta.Config{
		EnableAllBrokerSymbols: allBrokerSymbols,
		Broker:                 scmeta.BrokerID_BYBIT,
	}
	scCfgModule, err := scMetaCfg.New(ml, scSrcModule)
	if err != nil {
		return nil, fmt.Errorf("scMetaCfg.New error, %s", err.Error())
	}
	return scCfgModule, nil
}

// CrasherModule crash
type CrasherModule = *crasherModule

type crasherModule struct {
	lifecycle bat.Lifecycle
}

func New(lifecycle bat.Lifecycle) CrasherModule {
	m := &crasherModule{
		lifecycle: lifecycle,
	}
	return m
}

func (m *crasherModule) Panic(ctx context.Context, msg string, details ...blabel.Label) error {
	// glog.Error(ctx, "crasher: "+msg, details...) // print to basic logger again in case of logger closed / flushed

	if m == nil {
		panic("crasher: panic on nil crasher")
	}
	time.Sleep(time.Second * 10)
	panic(msg)
}
