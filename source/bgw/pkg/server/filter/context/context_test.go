package context

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"bgw/pkg/config"

	gmd "google.golang.org/grpc/metadata"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
)

func TestContextFilter_Do(t *testing.T) {
	Convey("test new", t, func() {
		Init()
		c := &contextFilter{}
		c.flagWithRaw = true
		n := c.GetName()
		So(n, ShouldEqual, filter.ContextFilterKey)

		next := func(ctx *types.Ctx) error {
			return nil
		}

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Method = "POST"
		handler := c.Do(next)
		err := handler(ctx)
		So(err, ShouldBeNil)
	})
}

func TestContextFilter_parseFlags(t *testing.T) {
	Convey("test parse flags", t, func() {
		c := &contextFilter{}
		args := []string{"route", "--inboundHeader=[\"ABC\",\"BB\"]", "--outboundHeader=[\"ABC\",\"BB\"]", "--inboundCookie=[\"ABC\",\"_by_l_g_d\"]", "--withToken=true"}
		err := c.parseFlags(context.Background(), args)
		So(err, ShouldBeNil)

		args = []string{"route", "--wrongArgs=1234"}
		err = c.parseFlags(context.Background(), args)
		So(err, ShouldNotBeNil)

		args = []string{"route", "--inboundHeader={\"ABC\":\"BB\"}"}
		err = c.parseFlags(context.Background(), args)
		So(err, ShouldNotBeNil)

		args = []string{"route", "--outboundHeader={\"ABC\":\"BB\"}"}
		err = c.parseFlags(context.Background(), args)
		So(err, ShouldNotBeNil)

		args = []string{"route", "--inboundCookie={\"ABC\":\"BB\"}", "--withToken=true"}
		err = c.parseFlags(context.Background(), args)
		So(err, ShouldNotBeNil)

		err = c.Init(context.Background())
		err = c.Init(context.Background(), "route")
	})
}

type emptyMd struct{}

func (e *emptyMd) Metadata() gmd.MD {
	return map[string][]string{}
}

type md struct{}

func (m *md) Metadata() gmd.MD {
	return map[string][]string{
		nextToken: {"123"},
		weakToken: {"234"},
	}
}

func TestContextFilter_outbound(t *testing.T) {
	Convey("test outbound", t, func() {
		c := &contextFilter{}
		c.flagWithToken = true
		ctx := &types.Ctx{}

		ctx.SetUserValue(constant.CtxInvokeResult, &emptyMd{})
		c.outbound(ctx)
		So(string(ctx.Response.Header.Peek("traceID")), ShouldEqual, "")
		So(string(ctx.Response.Header.Peek("token")), ShouldEqual, "")
		So(ctx.Response.Header.PeekCookie(config.GetSecureTokenKey()), ShouldBeNil)
		So(string(ctx.Response.Header.ContentType()), ShouldEqual, "text/plain; charset=utf-8")

		m := metadata.MDFromContext(ctx)
		s := "secure_token"
		w := "weak_token"
		m.Intermediate.SecureToken = &s
		m.Intermediate.WeakToken = &w
		ctx.SetUserValue(constant.CtxInvokeResult, m)
		c.outbound(ctx)
		So(string(ctx.Response.Header.Peek("traceID")), ShouldEqual, "")
		So(string(ctx.Response.Header.Peek("token")), ShouldEqual, "")
		So(ctx.Response.Header.PeekCookie(config.GetSecureTokenKey()), ShouldBeNil)
		So(string(ctx.Response.Header.ContentType()), ShouldEqual, "text/plain; charset=utf-8")
	})
}

func TestContextFilter_outboundHeader(t *testing.T) {
	Convey("test outboundheader", t, func() {
		c := &contextFilter{}
		c.flagOutboundHeaders = []string{all}
		ctx := &types.Ctx{}
		rmd := map[string][]string{
			constant.BgwAPIResponseCodes: {},
			"345":                        {"123", "234"},
		}
		c.outboundHeader(ctx, rmd)

		c.flagOutboundHeaders = []string{"", "345", "789"}
		c.outboundHeader(ctx, rmd)
	})
}

func TestContextFilter_outboundCookie(t *testing.T) {
	Convey("test outboundCookie", t, func() {
		c := &contextFilter{}
		ctx := &types.Ctx{}
		rmd := map[string][]string{
			"set-cookie": {"123", "234"},
		}

		c.outboundCookie(ctx, rmd)
	})
}

func TestContextFilter_inboundHeader(t *testing.T) {
	Convey("test inboundHeader", t, func() {
		c := &contextFilter{}
		c.flagInboundHeaders = []string{all}
		ctx := &types.Ctx{}
		ctx.Request.Header.Set("345", "234")
		c.inboundHeader(ctx)

		c.flagInboundHeaders = []string{"", "234", "345"}
		c.inboundHeader(ctx)
	})
}

func TestContextFilter_inboundCookie(t *testing.T) {
	Convey("test inboundHeader", t, func() {
		c := &contextFilter{}
		c.flagInboundCookies = []string{"", "cookie"}
		ctx := &types.Ctx{}
		ctx.Request.Header.SetCookie("cookie", "234")
		c.inboundCookie(ctx)
	})
}

func TestGetRemoteIP(t *testing.T) {
	t.Log("test")
	a := assert.New(t)
	a.Equal(true, true)

	c := &types.Ctx{}
	c.Request.Header.Set("X-Forwarded-For", " 23.54.62.67, 34.26.9.23")
	ip := bhttp.GetRemoteIP(c)
	t.Log(ip)

	c.Request.Header.Set("X-Forwarded-For", " , 34.26.9.23")
	ip = bhttp.GetRemoteIP(c)
	t.Log(ip)

	c.Request.Header.Set("X-Forwarded-For", "  ")
	ip = bhttp.GetRemoteIP(c)
	t.Log(ip)
}

func TestDomain(t *testing.T) {
	a := assert.New(t)

	domain := "api2.bybit-test-1.bybit.com"
	root := domain
	strSplit := strings.Split(domain, ".")
	if len(strSplit) > 2 {
		root = strings.Join(strSplit[1:], ".")
	}
	t.Log(root)

	root1 := domain
	strSplit1 := bytes.Split([]byte(domain), []byte("."))
	if len(strSplit1) > 2 {
		root1 = string(bytes.Join(strSplit1[1:], []byte(".")))
	}
	t.Log(root1)
	a.Equal(root, root1)
}
