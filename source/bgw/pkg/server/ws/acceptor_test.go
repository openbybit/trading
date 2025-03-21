package ws

import (
	"testing"
	"time"

	"bgw/pkg/server/ws/mock"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"github.com/stretchr/testify/assert"
)

func TestAcceptor(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	id := "id"
	appid := "appid"
	topics := []string{"t1"}
	addr := "@"

	initExchange()

	t.Run("basic", func(t *testing.T) {
		acc := newAcceptor(mock.NewGrpcServerStream(), id, appid, topics, &acceptorOptions{Address: addr, PublicTopics: []string{"public"}, Extensions: map[string]string{"version": "v1"}})
		assert.Equal(t, id, acc.ID())
		assert.Equal(t, appid, acc.AppID())
		assert.Equal(t, topics, acc.Topics())
		assert.Equal(t, 0, acc.UserShardIndex())
		assert.Equal(t, 0, acc.UserShardTotal())
		assert.Equal(t, uint64(0), acc.FocusEvents())
		assert.Equal(t, addr, acc.Address())
		assert.Equal(t, 0.0, acc.SendChannelRate())
		assert.Equal(t, int64(0), acc.LastWriteFailTime())
		assert.NotZero(t, acc.CreateTime())
		assert.NotNil(t, acc.Extensions())
		assert.ElementsMatch(t, []string{"public"}, acc.PublicTopics())
	})

	t.Run("admin", func(t *testing.T) {
		s := mock.NewGrpcServerStream()
		acc := newAcceptor(s, id, appid, topics, &acceptorOptions{Address: addr})
		_, err := acc.SendAdmin(nil)
		assert.NotNil(t, err)

		_, err = acc.SendAdmin(&envelopev1.SubscribeResponse{})
		assert.NotNil(t, err)

		acc.Start()

		req := newAdminReq(envelopev1.Admin_TYPE_STATUS, "")
		_, err = acc.SendAdmin(req)
		time.Sleep(time.Millisecond * 11)
		acc.Close()
		assert.Nil(t, err)
	})
}
