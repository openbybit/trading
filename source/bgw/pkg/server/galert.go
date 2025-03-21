package server

import (
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"

	"bgw/pkg/config"
)

func initAlert() {
	webhook := config.Global.Alert.GetOptions("path", "")
	fields := []*galert.Field{
		galert.BasicField("env", fmt.Sprintf("%s:%s", env.EnvName(), env.ProjectEnvName())),
		galert.CurrentTimeField("utc", ""),
		galert.BasicField("ip", nets.GetLocalIP()),
		galert.BasicField("namespace", config.GetNamespace()+":"+config.GetGroup()),
	}

	x := galert.New(&galert.Config{
		Webhook: webhook,
		Fields:  fields,
	})
	galert.SetDefault(x)
}
