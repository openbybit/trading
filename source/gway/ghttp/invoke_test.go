package ghttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestInvoke(t *testing.T) {
	invoker := GetInvoker(
		WithReadBufferSize(defaultReadBufferSize),
		WithMaxConnsPerHost(defaultMaxConnsPerHost),
		WithMaxIdleConns(defaultMaxIdleConns),
		WithMaxConnsPerHost(defaultMaxConnsPerHost),
	)

	_ = invoker.Invoke(nil, "aaa", &mockRequest{method: "!"}, &mockResult{})
	_ = invoker.Invoke(context.Background(), "aaa", &mockRequest{method: "GET"}, &mockResult{})

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("test", "test")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("body"))
	}))
	defer func() { testServer.Close() }()

	addr := strings.TrimPrefix(testServer.URL, "http://") + "/"

	_ = invoker.Invoke(context.Background(), addr, &mockRequest{method: "GET", md: metadata.MD{"host": []string{"host"}}}, &mockResult{})
	_ = invoker.Invoke(context.Background(), addr, &mockRequest{method: "POST"}, &mockResult{})
}

type mockRequest struct {
	method string
	md     metadata.MD
}

func (r *mockRequest) GetService() string { return "mock" }

func (r *mockRequest) GetMethod() string { return r.method }

func (r *mockRequest) QueryString() []byte { return nil }

func (r *mockRequest) PayLoad() io.Reader { return nil }

func (r *mockRequest) GetMetadata() metadata.MD { return r.md }

type mockResult struct{}

func (r *mockResult) SetStatus(int)           {}
func (r *mockResult) SetMetadata(metadata.MD) {}
func (r *mockResult) SetData(io.Reader) error { return nil }
