package server

import (
	"code.bydev.io/fbu/gateway/gway.git/gsechub"

	"bgw/pkg/config"
)

func initSechub() {
	sechubCfg := config.Global.SechubCfg
	clientCfg := &gsechub.Config{
		Host:       sechubCfg.Address,
		AppName:    sechubCfg.GetOptions("app_name", ""),
		AppSignKey: sechubCfg.GetOptions("app_sign_key", ""),
		TLSCert:    sechubCfg.GetOptions("tls_cert", ""),
	}
	gsechub.Init(clientCfg)
}
