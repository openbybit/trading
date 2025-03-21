package ws

import (
	"sync"
	"testing"

	"bgw/pkg/common/types"
	"bgw/pkg/server/ws/mock"
	"bgw/pkg/service/masque"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

func TestUpgrade(t *testing.T) {
	s := &wsServer{}
	h := s.upgrade(version2)
	ctx := &types.Ctx{}
	uri := fasthttp.AcquireURI()
	uri.SetPath("/private")
	uri.SetQueryString(userToken + "=abcdef&a=b")
	ctx.Request.SetURI(uri)
	ctx.Request.Header.Set("Connection", "Upgrade")
	ctx.Request.Header.Set("Upgrade", "Websocket")
	ctx.Request.Header.Set("Sec-Websocket-Version", "13")
	ctx.Request.Header.Set("Sec-Websocket-Key", "213")
	getAppConf().ReplaceStreamPlatform = true

	t.Run("nil srv", func(t *testing.T) {
		h(ctx)
	})

	t.Run("verifyToken err", func(t *testing.T) {
		uri = fasthttp.AcquireURI()
		uri.SetPath("/private")
		uri.SetQueryString(userToken + "=&a=b")
		ctx.Request.SetURI(uri)
		masque.SetMasqueService(nil)
		s.upgrader = &websocket.FastHTTPUpgrader{
			WriteBufferPool: &sync.Pool{},
			CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
				return true
			},
		}
		s.srv = &fasthttp.Server{
			ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
				t.Log(err)
			},
		}
		h(ctx)

		uri.SetQueryString(userToken + "=abcdef&a=b")
		ctx.Request.SetURI(uri)
		h(ctx)
	})

	t.Run("normal", func(t *testing.T) {
		masque.SetMasqueService(&mock.Masq{
			Uid: 12345,
		})
		h(ctx)
	})
}

func TestDoUpgrade(t *testing.T) {
	ctx := &types.Ctx{}
	uri := fasthttp.AcquireURI()
	uri.SetPath("/private")
	uri.SetQueryString(userToken + "=abcdef&a=b")
	ctx.Request.SetURI(uri)
	getAppConf().ReplaceStreamPlatform = true

	s := &wsServer{}
	s.doUpgrade(ctx, &WSConn{}, version2, 12345)

	t.Run("AddSession err", func(t *testing.T) {
		sconf := getDynamicConf()
		sconf.MaxSessions = 0
		s.doUpgrade(ctx, &WSConn{}, version2, 12345)
		sconf.MaxSessions = 10000
	})

	t.Run("bindUser err", func(t *testing.T) {
		sconf := getDynamicConf()
		sconf.MaxSessionsPerUser = 0
		s.doUpgrade(ctx, &WSConn{}, version2, 12345)
		sconf.MaxSessionsPerUser = 10000
	})
}

func TestUnregister(t *testing.T) {
	s := &wsServer{}
	s.Unregister()
	s.Stop()
}
