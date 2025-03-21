package ws

import (
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	s := New()
	assert.False(t, s.isRunning())
	assert.False(t, s.Health(), "not health")
	assert.NotNil(t, s.State())

	getConfigMgr().LoadStaticConfig()

	_ = s.Start()
	// time.Sleep(time.Second)
	_ = s.Stop()
	// proc.Shutdown()
	_ = s.Stop()
}
