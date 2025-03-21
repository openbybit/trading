package nacos

import (
	"errors"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
)

// GetNacosConfig will return the nacos config
func GetNacosConfig(namespace string) (*gnacos.Config, error) {
	rc := &config.Global.NacosCfg
	if rc == nil {
		return nil, fmt.Errorf("get nacos config error")
	}

	params := map[string]string{
		gnacos.TIMEOUT_KEY:   rc.Timeout,
		gnacos.LOG_DIR_KEY:   rc.GetOptions(gnacos.LOG_DIR_KEY, ""),
		gnacos.LOG_LEVEL_KEY: rc.GetOptions(gnacos.LOG_LEVEL_KEY, "info"),
		gnacos.CACHE_DIR_KEY: rc.GetOptions(gnacos.CACHE_DIR_KEY, ""),
		gnacos.APP_NAME_KEY:  constant.GWSource,
	}
	for k, vs := range rc.Options {
		if len(vs) > 0 {
			params[k] = vs[0]
		}
	}
	password, err := gsechub.Decrypt(rc.Password)
	if err != nil {
		// local environment, donot need sechub
		if !(errors.Is(err, gsechub.ErrNilClient) && !env.IsProduction()) {
			return nil, fmt.Errorf("nacos config sechub.Decrypt error: %w", err)
		}
	}

	cfg, err := gnacos.NewConfig(rc.Address, rc.Username, password, namespace, params)
	if err != nil {
		return nil, fmt.Errorf("nacos.NewConfig error: %w", err)
	}
	return cfg, nil
}
