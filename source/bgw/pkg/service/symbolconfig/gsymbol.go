package symbolconfig

import (
	"context"
	"net/url"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"code.bydev.io/fbu/gateway/gway.git/gsymbol"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
)

var gonce sync.Once

func initGsymbol() {
	gonce.Do(func() {
		cfg := &gsymbol.Config{}
		cfg.NacosAddress = nacosAddress()
		err := gsymbol.Start(cfg)
		if err != nil {
			galert.Error(context.Background(), "gsymbol init err, "+err.Error())
		}
	})
}

func GetFutureManager() *gsymbol.FutureManager {
	initGsymbol()
	return gsymbol.GetFutureManager()
}

func GetOptionManager() *gsymbol.OptionManager {
	initGsymbol()
	return gsymbol.GetOptionManager()
}

func GetSpotManager() *gsymbol.SpotManager {
	initGsymbol()
	return gsymbol.GetSpotManager()
}

func nacosAddress() string {
	cfg := &config.Global.NacosCfg
	u := url.URL{Host: cfg.Address}
	username := cfg.Username
	password, _ := gsechub.Decrypt(cfg.Password)
	if username != "" || password != "" {
		u.User = url.UserPassword(username, password)
	}
	values := u.Query()

	namespace := config.GetNamespace()
	if env.IsProduction() {
		namespace = constant.DEFAULT_NAMESPACE
	}
	values.Set("namespace", namespace)
	u.RawQuery = values.Encode()
	return "nacos:" + u.String()
}
