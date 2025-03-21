package ws

import (
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/stretchr/testify/assert"
)

func TestAcceptorMgr(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	topics1 := []string{"private.order", "private.position", "private.wallet", "private.execution", "private.notice"}
	topics2 := []string{"legalTenderAsset"}
	m := newAcceptorMgr()
	acc1 := newAcceptor(nil, "linear_0", "linear", topics1, &acceptorOptions{ShardIndex: 0, ShardTotal: 10, Address: "127.0.0.1"})
	acc2 := newAcceptor(nil, "linear_1", "linear", topics1, &acceptorOptions{ShardIndex: 1, ShardTotal: 10, Address: "127.0.0.1"})
	acc3 := newAcceptor(nil, "fiat", "fiat", topics2, &acceptorOptions{ShardIndex: -1, ShardTotal: 0, Address: "@"})

	m.RefreshAppIDGauge("linear")
	assert.Nil(t, m.GetByIndex(0))

	// 添加
	assert.Nil(t, m.Add(acc1))
	assert.Nil(t, m.Add(acc2))
	assert.Nil(t, m.Add(acc3))
	// 重复添加
	assert.NotNil(t, m.Add(acc1))

	// 查询
	assert.Equal(t, 3, m.Size())
	assert.NotNil(t, m.Get(acc1.ID()))
	assert.Equal(t, 3, len(m.GetAll()))
	assert.Equal(t, 2, len(m.GetByAppID("linear")))
	assert.Equal(t, acc1.ID(), m.GetByIndex(0).ID())
	assert.Equal(t, 2, len(m.GetByTopics([]string{"private.order", "private.position"})))
	assert.Nilf(t, m.GetByTopics(nil), "no topics")

	m.Remove(acc1.ID())
	assert.Equal(t, 2, m.Size())

	m.Remove("not_exist_id")

	m.Close()
}
