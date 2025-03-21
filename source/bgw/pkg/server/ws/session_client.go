package ws

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
)

// Client is session info
type Client interface {
	GetMemberId() int64
	SetMemberId(int64)
	GetAPIKey() string
	SetAPIKey(apiKey string)

	GetPath() string
	GetIP() string
	GetBrokerID() int32
	GetUserAgent() string
	GetHost() string
	GetReferer() string
	GetXOriginFrom() string
	GetParams() map[string]string

	GetTopics() Topics
	// 返回新增的订阅topics
	Subscribe(topics []string) []string
	// 返回新增的取消订阅topics
	Unsubscribe(topics []string) []string
	HasTopic(topic string) bool
	Close()
	fmt.Stringer
}

var _ Client = (*client)(nil)

type client struct {
	uid       int64
	ip        string
	path      string
	host      string
	userAgent string
	referer   string
	brokerID  int32
	xof       string            // X-Origin-From
	params    map[string]string // 连接参数
	paramsStr string            // 仅用于日志输出
	apikey    atomic.Value      //
	topics    atomic.Value      // 高频读,低频写
}

type ClientConfig struct {
	IP          string
	Path        string
	Host        string
	UserAgent   string
	Referer     string
	BrokerID    string
	XOriginFrom string
	Params      map[string]string
}

// NewClient new client
func NewClient(conf *ClientConfig) Client {
	c := &client{}
	if conf != nil {
		c.ip = conf.IP
		c.path = conf.Path
		c.host = conf.Host
		c.userAgent = conf.UserAgent
		c.referer = conf.Referer
		c.xof = conf.XOriginFrom
		c.brokerID = cast.ToInt32(conf.BrokerID)
		c.params = conf.Params
		c.paramsStr = paramsToString(conf.Params)
	}
	c.topics.Store(newTopics())
	return c
}

func paramsToString(p map[string]string) string {
	if len(p) == 0 {
		return ""
	}

	var ks = make([]string, 0, len(p))
	for k := range p {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	i := 0
	b := bytes.NewBuffer(nil)
	for _, k := range ks {
		if i > 0 {
			b.WriteString("&")
		}
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(p[k])
		i++
	}

	return b.String()
}

// GetMemberId get member id
func (c *client) GetMemberId() int64 {
	return atomic.LoadInt64(&c.uid)
}

// SetMemberId set member id
func (c *client) SetMemberId(uid int64) {
	atomic.StoreInt64(&c.uid, uid)
}

// GetAPIKey get apikey
func (c *client) GetAPIKey() string {
	res, ok := c.apikey.Load().(string)
	if ok {
		return res
	}

	return ""
}

// SetAPIKey set apikey
func (c *client) SetAPIKey(apikey string) {
	c.apikey.Store(apikey)
}

// GetPath get path
func (c *client) GetPath() string {
	return c.path
}

// GetIP get ip
func (c *client) GetIP() string {
	return c.ip
}

// GetBrokerID get broker id
func (c *client) GetBrokerID() int32 {
	return c.brokerID
}

// GetUserAgent get useragent
func (c *client) GetUserAgent() string {
	return c.userAgent
}

// GetHost get host
func (c *client) GetHost() string {
	return c.host
}

// GetReferer get referer
func (c *client) GetReferer() string {
	return c.referer
}

// GetXOriginFrom get X-Origin-From
func (c *client) GetXOriginFrom() string {
	return c.xof
}

// GetParams get params
func (c *client) GetParams() map[string]string {
	return c.params
}

// GetTopics get topics
func (c *client) GetTopics() Topics {
	res, ok := c.topics.Load().(Topics)
	if ok {
		return res
	}

	return newTopics()
}

// Subscribe subscribe topics
func (c *client) Subscribe(topics []string) []string {
	newTopics := c.GetTopics().Clone()
	res := newTopics.Add(topics...)
	c.topics.Store(newTopics)
	for _, t := range res {
		WSGaugeInc("subscribe_topic", t)
	}
	return res
}

// Unsubscribe unsubscribe topics
func (c *client) Unsubscribe(topics []string) []string {
	newTopics := c.GetTopics().Clone()
	res := newTopics.Remove(topics...)
	c.topics.Store(newTopics)
	for _, t := range res {
		WSGaugeDec("subscribe_topic", t)
	}
	return res
}

// HasTopic has topic
func (c *client) HasTopic(topic string) bool {
	return c.GetTopics().Contains(topic)
}

// Close close
func (c *client) Close() {
	topics := c.GetTopics().Values()
	for _, t := range topics {
		WSGaugeDec("subscribe_topic", t)
	}
}

// String string
func (c *client) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprint(c.uid))
	builder.WriteString(`-`)
	builder.WriteString(c.ip)
	builder.WriteString("-")
	builder.WriteString(c.path)
	if len(c.paramsStr) > 0 {
		builder.WriteString("-")
		builder.WriteString(c.paramsStr)
	}
	return builder.String()
}
