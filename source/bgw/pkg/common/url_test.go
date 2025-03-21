package common

import (
	"testing"

	"bgw/pkg/common/constant"

	"github.com/tj/assert"
)

func TestURL(t *testing.T) {
	a := assert.New(t)

	u1, err := NewURL("nacos", WithGroup("default"), WithProtocol("nacos"))
	a.Nil(err)
	a.Equal(u1.Protocol, "nacos", "mismatch protocol")
	a.Equal(u1.Addr, "nacos", "mismatch addr")
	t.Log(u1)

	u2, err := NewURL("dns://foo.com")
	a.Nil(err)
	a.Equal(u2.Protocol, "dns", "error protocol")
	a.Equal(u2.Addr, "foo.com", "error addr")

	u1, err = NewURL("abcdfg", WithProtocol(constant.NacosProtocol), WithGroup("group:jjj"), WithNamespace("name:space"))
	a.Nil(err)
	t.Log(u1.String())
	t.Log(u1.GetParam(constant.NAMESPACE_KEY, constant.DEFAULT_NAMESPACE))
	t.Log(u1.GetParam(constant.GROUP_KEY, constant.DEFAULT_GROUP))
}
