package ghttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"unsafe"

	"google.golang.org/grpc/metadata"
)

var (
	invoker *Invoker
	once    sync.Once
)

// Invoker http->http invoker
type Invoker struct {
	client *http.Client
}

// GetInvoker new http invoker
func GetInvoker(opts ...Option) *Invoker {
	once.Do(func() {
		options := defaultOptions()

		for _, opt := range opts {
			if opt != nil {
				opt(options)
			}
		}

		invoker = &Invoker{
			client: getClient(options),
		}
	})

	return invoker
}

// getClient returns a http client with keep alive
func getClient(opts *options) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()

	t.ReadBufferSize = opts.readBufferSize
	t.MaxIdleConnsPerHost = opts.maxIdleConnsPerHost
	t.MaxIdleConns = opts.maxIdleConns
	t.MaxConnsPerHost = opts.maxConnsPerHost
	t.DisableKeepAlives = false

	return &http.Client{Transport: t}
}

// Invoke http invoker
func (i *Invoker) Invoke(ctx context.Context, addr string, request Request, result Result) error {
	path := request.GetService()
	url := i.getURL(addr, path)
	method := request.GetMethod()
	switch method {
	case http.MethodGet:
		url += unsafeGetString(request.QueryString())
	default:
		url += unsafeGetString(request.QueryString())
	}

	req, err := http.NewRequestWithContext(ctx, method, url, request.PayLoad())
	if err != nil {
		return fmt.Errorf("create requet error: %w, %s", ErrReqBuildFailed, err.Error())
	}

	md := request.GetMetadata()
	for key, vs := range md {
		if len(vs) > 0 {
			req.Header.Set(key, vs[0])
		}
	}

	// override host
	if hosts := md.Get("host"); len(hosts) > 0 {
		req.Host = hosts[0]
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()
	defer func() { result.SetStatus(resp.StatusCode) }()

	headers := metadata.New(nil)
	for key, vs := range resp.Header {
		if len(vs) > 0 {
			headers.Set(key, vs...)
		}
	}

	result.SetMetadata(headers)
	return result.SetData(resp.Body)
}

func (i *Invoker) getURL(addr, path string) string {
	url := strings.Builder{}
	url.WriteString("http://")
	url.WriteString(addr)
	url.WriteString(path)
	url.WriteString("?")
	return url.String()
}

// unsafeGetString byte convert to string
func unsafeGetString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
