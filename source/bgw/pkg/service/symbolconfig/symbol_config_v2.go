package symbolconfig

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gsymbol/future"

	"bgw/pkg/config"
)

var (
	scOnce sync.Once
	sc     *future.Scmeta
)

// GetSymbolConfig return symbol config model without bat
func GetSymbolConfig() (*future.Scmeta, error) {
	var err error
	scOnce.Do(func() {
		kfs := &config.Global.Kafka
		if kfs == nil || kfs.Address == "" {
			err = fmt.Errorf("remote config of kafka addr is nil, for symbol config")
			return
		}
		addrs := strings.Split(kfs.Address, ",")
		if len(addrs) == 0 || addrs[0] == "" {
			err = fmt.Errorf("remote config of kafka error, for symbol config")
			return
		}
		enableLogResult := cast.StringToBool(kfs.GetOptions("enable_symbol_config_log", ""))
		cfg := &future.Config{
			Server:           ServerName,
			ResultTopic:      ResultTopicName,
			ResultAckTopic:   ResultAckTopicName,
			Addr:             addrs,
			LogResult:        enableLogResult,
			AllBrokerSymbols: true,
		}
		sc, err = future.New(context.Background(), cfg)
	})
	return sc, err
}
