package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

var (
	ErrNotSupport         = errors.New("httpc: not support")
	ErrInvalidPointerType = errors.New("httpc: invalid type, must be pointer")
)

const (
	DefaultTimeout = time.Second * 60
)

type Config struct {
	Client           *http.Client
	Timeout          time.Duration
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration
	Proxy            *url.URL
	BaseUrl          string
}

func NewClient(c *Config) *Client {
	if c == nil {
		c = &Config{}
	}

	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = DefaultTimeout
	}

	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = DefaultTimeout
	}

	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: c.DialTimeout,
		}).DialContext,
		TLSHandshakeTimeout: c.HandshakeTimeout,
	}

	if c.Proxy != nil {
		t.Proxy = http.ProxyURL(c.Proxy)
	}

	client := &http.Client{
		Timeout:   c.Timeout,
		Transport: t,
	}

	return &Client{client: client, baseUrl: c.BaseUrl}
}

type Client struct {
	client  *http.Client
	baseUrl string
}

func (c *Client) Get(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodGet, url, nil, toCallOptions(opts))
}

func (c *Client) Post(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodPost, url, reqBody, toCallOptions(opts))
}

func (c *Client) Put(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodPut, url, reqBody, toCallOptions(opts))
}

func (c *Client) Patch(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodPatch, url, reqBody, toCallOptions(opts))
}

func (c *Client) Delete(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodDelete, url, nil, toCallOptions(opts))
}

func (c *Client) Head(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodHead, url, url, toCallOptions(opts))
}

func (c *Client) Connect(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodConnect, url, nil, toCallOptions(opts))
}

func (c *Client) Options(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodOptions, url, nil, toCallOptions(opts))
}

func (c *Client) Trace(ctx context.Context, url string, opts ...CallOption) Response {
	return c.doRequest(ctx, http.MethodTrace, url, nil, toCallOptions(opts))
}

func (c *Client) doRequest(ctx context.Context, method string, url string, reqBody interface{}, o *callOptions) Response {
	if ctx == nil {
		ctx = context.Background()
	}

	var body []byte
	if reqBody != nil {
		contentType := parseContentType(o.header.Get(HeaderContentType))
		if x, err := encode(contentType, reqBody); err != nil {
			return Response{err: err}
		} else {
			body = x
		}
	}

	url = replacePathParams(url, o.params)
	realUrl := toUrl(c.baseUrl, url)
	req, err := http.NewRequestWithContext(ctx, method, realUrl, nil)
	if err != nil {
		return Response{err: err}
	}

	// set header
	req.Header = o.header

	// build query
	if len(o.query) > 0 {
		query := req.URL.Query()
		merge(query, o.query)
		req.URL.RawQuery = query.Encode()
	}

	// build cookies
	if len(o.cookies) > 0 {
		if req.Header == nil {
			req.Header = http.Header{}
		}
		for _, c := range o.cookies {
			req.AddCookie(c)
		}
	}

	var rsp *http.Response
	for i := 0; ; i++ {
		if body != nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

		if o.timeout != 0 {
			rsp, err = doRequestWithTimeout(c.client, req, ctx, o.timeout)
		} else {
			rsp, err = c.client.Do(req)
		}

		if err == nil {
			if rsp.StatusCode != http.StatusOK {
				err = &StatusErr{Code: rsp.StatusCode, Info: rsp.Status}
			}

			return Response{Response: rsp, err: err}
		}

		if i >= o.retry {
			break
		}
		delay := o.backoff.Next()
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	return Response{err: err}
}

func doRequestWithTimeout(client *http.Client, req *http.Request, parent context.Context, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	req = req.Clone(ctx)
	return client.Do(req)
}
