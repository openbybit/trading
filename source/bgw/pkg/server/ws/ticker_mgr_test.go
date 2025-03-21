package ws

import (
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func TestTick(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)
	gTickerMgr.Start()
	time.Sleep(time.Millisecond)
	gTickerMgr.Stop()
}
