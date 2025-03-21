package ws

import (
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	t.Run("Parse", func(t *testing.T) {
		var s sdkConf
		err := s.Parse("abc")
		assert.NotNil(t, err)
	})

	t.Run("build", func(t *testing.T) {
		var d dynamicLogConf
		d.build()
	})

	t.Run("dynamic conf", func(t *testing.T) {
		d := dynamicConf{}
		d.Verify()
	})

	t.Run("convert level fail", func(t *testing.T) {
		l := LogConf{
			Level: "invalid level",
		}
		gl := l.convert(bgwsLogName)
		assert.Equal(t, glog.InfoLevel, gl.Level)
	})

	t.Run("log build mainnet", func(t *testing.T) {
		es := newEnvStore()
		defer es.Recovery()
		es.SetMainnet()
		env.SetProjectEnvName("ls-asset")
		l := LogConf{}
		l.convert(bgwsLogName)
		assert.Equal(t, glog.TypeLumberjack, l.Type)
		assert.Equal(t, "info", l.Level)
	})
}
