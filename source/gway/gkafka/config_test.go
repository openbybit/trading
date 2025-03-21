package gkafka

import (
	"testing"
)

func Test_newConfig(t *testing.T) {
	t.Run("new config", func(t *testing.T) {
		cfg := newBaseCfg()
		if cfg == nil {
			t.Error("should be not nil")
		}
	})

	t.Run("to byone config", func(t *testing.T) {
		// patch := gomonkey.ApplyFunc(env.ServiceName, func() string { return "bgw" })
		// defer patch.Reset()
		conf := &Config{
			Topic:    "trading_result.USDT.0_10",
			Brokers:  []string{"127.0.0.1"},
			Username: "username",
			Password: "password",
			Config:   newBaseCfg(),
		}
		_, err := toByoneConfig(conf)
		if err != nil {
			t.Errorf("toByoneConfig has some error, %v", err)
		}
	})

	t.Run("test byone err", func(t *testing.T) {
		_, err := toByoneConfig(&Config{Config: newBaseCfg()})
		if err == nil {
			t.Errorf("toByoneConfig should return err")
		}
	})
}
