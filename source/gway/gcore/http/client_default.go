package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	global *Client
	proxy  *Client
)

func init() {
	global = NewClient(nil)

	squid := strings.TrimSpace(os.Getenv("SQUID_PROXY"))
	if squid != "" {
		if !strings.Contains(squid, "://") {
			squid = fmt.Sprintf("http://%s", squid)
		}

		p, err := url.Parse(squid)
		if err != nil {
			log.Printf("init squid proxy fail, err=%v, url=%v\n", err, squid)
		} else {
			proxy = NewClient(&Config{Proxy: p})
		}
	}

	if proxy == nil {
		proxy = global
	}
}

func Set(client *Client, useProxy bool) {
	if useProxy {
		proxy = client
	} else {
		global = client
	}
}

func getClient(userProxy bool) *Client {
	if userProxy {
		return proxy
	}

	return global
}

func Get(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodGet, url, nil, o)
}

func Post(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodPost, url, reqBody, o)
}

func Put(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodPut, url, reqBody, o)
}

func Patch(ctx context.Context, url string, reqBody interface{}, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodPatch, url, reqBody, o)
}

func Delete(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodDelete, url, nil, o)
}

func Head(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodHead, url, nil, o)
}

func Connect(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodConnect, url, nil, o)
}

func Options(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodOptions, url, nil, o)
}

func Trace(ctx context.Context, url string, opts ...CallOption) Response {
	o := toCallOptions(opts)
	return getClient(o.proxy).doRequest(ctx, http.MethodTrace, url, nil, o)
}
