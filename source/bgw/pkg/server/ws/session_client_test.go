package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	uid := int64(123456)
	apikey := "key1"
	ip := "127.0.0.1"
	path := "/v5/private"
	brokerID := ""
	userAgent := ""
	referer := ""
	host := "localhost"
	xOriginFrom := "x-origin-from"
	conf := ClientConfig{
		IP:          ip,
		Path:        path,
		BrokerID:    brokerID,
		UserAgent:   userAgent,
		Referer:     referer,
		Host:        host,
		Params:      map[string]string{"source": "web", "version": "1"},
		XOriginFrom: xOriginFrom,
	}
	c := NewClient(&conf)
	c.SetMemberId(uid)
	c.SetAPIKey(apikey)
	assert.Equal(t, uid, c.GetMemberId())
	assert.Equal(t, apikey, c.GetAPIKey())
	assert.Equal(t, path, c.GetPath())
	assert.Equal(t, ip, c.GetIP())
	assert.EqualValues(t, 0, c.GetBrokerID())
	assert.Equal(t, userAgent, c.GetUserAgent())
	assert.Equal(t, referer, c.GetReferer())
	assert.Equal(t, 2, len(c.GetParams()))
	assert.Equalf(t, host, c.GetHost(), "host: %v", c.GetHost())
	assert.Equal(t, xOriginFrom, c.GetXOriginFrom())

	c.Subscribe([]string{"t1"})
	assert.True(t, c.HasTopic("t1"))
	assert.False(t, c.HasTopic("t3"))
	assert.ElementsMatch(t, []string{"t1"}, c.GetTopics().Values())
	c.Unsubscribe([]string{"t1", "t2"})
	assert.False(t, c.HasTopic("t1"))
	assert.Zero(t, c.GetTopics().Size())
	//
	c.Subscribe([]string{"t1"})
	c.Close()
	assert.Equalf(t, "123456-127.0.0.1-/v5/private-source:web&version:1", c.String(), "%s", c.String())

	// 其他特殊case
	c = &client{}
	assert.Empty(t, c.GetAPIKey())
	assert.NotNil(t, c.GetTopics())
}
