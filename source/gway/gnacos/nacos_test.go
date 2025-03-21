package gnacos

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestNacos(t *testing.T) {
	convey.Convey("NewConfig", t, func() {
		addr := "nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848"
		username := "nacos"
		password := "nacos"
		cfg, err := NewConfig("", username, password, "", nil)
		convey.ShouldNotBeNil(err)
		convey.ShouldBeNil(cfg)

		cfg, err = NewConfig(addr, username, password, DEFAULT_NAMESPACE, nil)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(cfg)

		rawURL := "nacos://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?cache=data%2Fcache%2Fnacos&log=data%2Flogs%2Fnacos&loglevel=error&namespace=bgw-aaron&timeout=30s"
		cfg, err = NewConfigByURL(rawURL)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(cfg)

		cfg, err = NewConfigByURL("")
		convey.ShouldNotBeNil(err)
		convey.ShouldBeNil(cfg)

		cfg, err = NewConfigByURL(nil)
		convey.ShouldNotBeNil(err)
		convey.ShouldBeNil(cfg)

		cfg, err = NewConfigByURL(123)
		convey.ShouldNotBeNil(err)
		convey.ShouldBeNil(cfg)

		params := map[string]string{REGISTRY_TIMEOUT_KEY: "5g"}
		cfg, err = NewConfig(addr, username, password, "", params)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(cfg)

		params = map[string]string{
			UPDATE_THREAD_NUM_KEY: "50",
			BEAT_INTERVAL_KEY:     "6000",
			SHARE_KEY:             "true",
		}
		cfg, err = NewConfig(addr, username, password, "", params)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(cfg)

		params = map[string]string{SHARE_KEY: "false"}
		noshareCfg, err := NewConfig(addr, username, password, "", params)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(noshareCfg)
		noshareClient, err := NewNamingClient(noshareCfg)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(noshareClient)

		cfg, err = NewConfig(addr, username, password, "", nil)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(cfg)

		client, err := NewNamingClient(cfg)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(client)

		client, err = NewNamingClient(cfg)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(client)
		client.Valid()
		client.Close()

		client1, err := NewConfigClient(noshareCfg)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(client1)
		client1, err = NewConfigClient(cfg)
		convey.ShouldBeNil(err)
		convey.ShouldNotBeNil(client1)
		client1.Valid()
		client1.Close()
	})
}
