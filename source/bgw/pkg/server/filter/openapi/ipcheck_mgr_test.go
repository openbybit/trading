package openapi

import (
	"strings"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"github.com/stretchr/testify/assert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIpCheck(t *testing.T) {
	data := `
enable: true
uids: [123000000,456]
`
	data = strings.TrimSpace(data)
	_ = getIpCheckMgr().build([]byte(data))
	t.Log(getIpCheckMgr().GetConfig())
	t.Log(getIpCheckMgr().CanSkipIpCheck(456))
	t.Log(getIpCheckMgr().CanSkipIpCheck(789))
}

func TestIpCheckDisable(t *testing.T) {
	data := `
enable: false
uids: [123000000,456]
`
	data = strings.TrimSpace(data)
	_ = getIpCheckMgr().build([]byte(data))
	t.Log(getIpCheckMgr().GetConfig())
	t.Log(getIpCheckMgr().CanSkipIpCheck(456))
	t.Log(getIpCheckMgr().CanSkipIpCheck(789))
}

func TestIpCheckNoConfig(t *testing.T) {
	conf := getIpCheckMgr().GetConfig()
	t.Log("config should be nil", conf)
	t.Log(getIpCheckMgr().CanSkipIpCheck(456))
	t.Log(getIpCheckMgr().CanSkipIpCheck(789))
}

func TestIpCheckMgr_OnEvent(t *testing.T) {
	ipm := getIpCheckMgr()
	err := ipm.OnEvent(&observer.BaseEvent{})
	assert.NoError(t, err)

	err = ipm.OnEvent(&observer.DefaultEvent{
		Value: "asas",
	})
	assert.NoError(t, err)

	data := `
enable: false
uids: [123000000,456999]
`
	err = ipm.OnEvent(&observer.DefaultEvent{
		Value: data,
	})
	assert.NoError(t, err)
	cfg := ipm.config.Load().(*IpCheckWhitelistConfig)
	assert.Equal(t, false, cfg.Enable)
	assert.Equal(t, int64(123000000), cfg.UidList[0])
	assert.Equal(t, int64(456999), cfg.UidList[1])
}

func TestIpCheckMgr_doinit(t *testing.T) {
	Convey("tes do init", t, func() {
		im := &ipCheckMgr{}
		im.doInit()
		im.Init()

		_ = im.GetPriority()
		_ = im.GetEventType()
	})
}
