package bhttp

import (
	"strings"

	"bgw/pkg/common/types"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var (
	ContentTypePostForm = []byte("application/x-www-form-urlencoded")
)

const (
	// HTTPMethodAny do not support HTTP_METHOD_ANY
	HTTPMethodAny = "HTTP_METHOD_ANY"
)

var HttpMethodAnyConfig = []string{
	"HTTP_METHOD_GET",
	"HTTP_METHOD_POST",
	"HTTP_METHOD_PUT",
	"HTTP_METHOD_DELETE",
}

// GetRemoteIP get user client ip
func GetRemoteIP(c *types.Ctx) (ip string) {
	defer func() {
		if ip != "" {
			ip = strings.TrimSpace(ip)
		} else {
			ip = c.RemoteIP().String()
		}
	}()

	xff := strings.TrimSpace(cast.ToString(c.Request.Header.Peek("X-Bybit-Forwarded-For")))
	glog.Debug(c, "bxff ips", glog.String("bxff-ips", xff))
	if xff == "" {
		xff = strings.TrimSpace(cast.ToString(c.Request.Header.Peek("X-Forwarded-For")))
		glog.Debug(c, "xff ips", glog.String("xff-ips", xff))
	}
	if xff == "" {
		return c.RemoteIP().String()
	}

	ips := strings.Split(xff, ",")
	for _, s := range ips {
		if s != "" {
			return s
		}
	}
	return
}
