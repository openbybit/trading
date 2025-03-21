package http

import (
	"context"
	"net/http"
	"testing"
)

func TestGet(t *testing.T) {
	var text string
	rsp := Get(context.Background(), "http://www.baidu.com")
	if err := rsp.Decode(&text); err != nil {
		t.Error(err)
	}
	t.Log(text)
}

func TestGetWithCookie(t *testing.T) {
	var text string
	rsp := Get(nil, "http://www.baidu.com", WithCookies(&http.Cookie{Name: "aa", Value: "aa"}))
	if err := rsp.Decode(&text); err != nil {
		t.Error(rsp.Error())
	}

	t.Log(text)
}
