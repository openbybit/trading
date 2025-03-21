package http

import (
	"net/http"
	"net/url"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/backoff"
)

type callOptions struct {
	timeout time.Duration     // invoke timeout
	params  map[string]string // path中参数,例如:http://www.baidu.com/im/v1/chats/:chat_id
	query   url.Values        // query参数,例如：http://www.baidu.com?aa=xxx&bb=xxx
	header  http.Header       // 消息头
	cookies []*http.Cookie    //
	retry   int               // 重试次数
	backoff backoff.BackOff   // 重试时间
	proxy   bool              // 是否使用代理,仅global才有用
}

func toCallOptions(opts []CallOption) *callOptions {
	o := &callOptions{}
	for _, fn := range opts {
		fn(o)
	}

	return o
}

func (c *callOptions) getHeader() http.Header {
	if c.header == nil {
		c.header = http.Header{}
	}

	return c.header
}

type CallOption func(o *callOptions)

func WithTimeout(d time.Duration) CallOption {
	return func(o *callOptions) {
		o.timeout = d
	}
}

func WithQuery(q url.Values) CallOption {
	return func(o *callOptions) {
		o.query = merge(o.query, q)
	}
}

func WithHeader(h http.Header) CallOption {
	return func(o *callOptions) {
		o.header = http.Header(merge(url.Values(o.header), url.Values(h)))
	}
}

func WithParams(p map[string]string) CallOption {
	return func(o *callOptions) {
		if o.params == nil {
			o.params = p
		} else {
			for k, v := range p {
				o.params[k] = v
			}
		}
	}
}

func WithContentType(v string) CallOption {
	return func(o *callOptions) {
		o.getHeader().Set(HeaderContentType, v)
	}
}

func WithCookies(c ...*http.Cookie) CallOption {
	return func(o *callOptions) {
		o.cookies = append(o.cookies, c...)
	}
}

func WithRetry(retry int, bc backoff.BackOff) CallOption {
	return func(o *callOptions) {
		if bc == nil {
			bc = backoff.NewExponential()
		}
		o.retry = retry
		o.backoff = bc
	}
}

func WithProxy() CallOption {
	return func(o *callOptions) {
		o.proxy = true
	}
}
