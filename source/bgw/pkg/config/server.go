package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
)

// ServerConfig is a server config
type ServerConfig struct {
	Addr               int             `json:",default=8080"`
	ReadTimeout        int             `json:",optional" ` // 秒
	ReadHeaderTimeout  int             `json:",optional" ` // 秒
	WriteTimeout       int             `json:",optional" ` // 秒
	IdleTimeout        int             `json:",optional" ` // 秒
	MaxHeaderBytes     int             `json:",optional" `
	ReadBufferSize     int             `json:",optional" `
	WriteBufferSize    int             `json:",optional" `
	MaxRequestBodySize int             `json:",optional"`
	ListenType         string          `json:",optional"` // unix is default
	ListenUnixAddr     string          `json:",optional" `
	ListenTCPAddr      int             `json:",optional"`
	EnableController   bool            `json:",optional"`
	ServiceRegistry    ServiceRegistry `json:",optional"`

	Options map[string]interface{} `json:",optional"`
}

type ServiceRegistry struct {
	Enable      bool   `json:",default=true"`
	ServiceName string `json:",optional"`
}

// GetHTTPServerConfig is a http server config
func GetHTTPServerConfig() *ServerConfig {
	return &Global.Server.Http
}

// GetWSServerConfig is a websocket server config
func GetWSServerConfig() *ServerConfig {
	return &Global.Server.Http
}

// GetAddr is an addr
func (s *ServerConfig) GetAddr() string {
	return fmt.Sprintf(":%d", s.Addr)
}

// GetPort is a port
func (s *ServerConfig) GetPort() int {
	return s.Addr
}

// GetReadTimeout is a read timeout
func (s *ServerConfig) GetReadTimeout() time.Duration {
	if s.ReadTimeout <= 1 {
		return time.Second
	}
	return time.Duration(s.ReadTimeout) * time.Second
}

// GetWriteTimeout is written timeout
func (s *ServerConfig) GetWriteTimeout() time.Duration {
	if s.WriteTimeout <= 1 {
		return time.Second
	}
	return time.Duration(s.WriteTimeout) * time.Second
}

func (s *ServerConfig) GetIdleTimeout() time.Duration {
	if s.IdleTimeout <= 10 {
		return 10 * time.Second
	}
	return time.Duration(s.IdleTimeout) * time.Second
}

// GetReadBufferSize is a read buffer size
func (s *ServerConfig) GetReadBufferSize() int {
	if s.ReadBufferSize <= 8192 {
		return 8192
	}
	return s.ReadBufferSize
}

// GetWriteBufferSize is a write buffer size
func (s *ServerConfig) GetWriteBufferSize() int {
	if s.WriteBufferSize <= 8192 {
		return 8192
	}
	return s.WriteBufferSize
}

// GetMaxRequestBodySize is a max request body size, default size is 4M
func (s *ServerConfig) GetMaxRequestBodySize() int {
	if s.MaxRequestBodySize <= 4 {
		return 4 * 1024 * 1024
	}
	return s.MaxRequestBodySize * 1024 * 1024
}

// GetEnableController is enable controller
func (s *ServerConfig) GetEnableController() bool {
	if e := os.Getenv(constant.EnableController); e == "true" {
		return true
	}
	return s.EnableController
}

// GetStringOption will return the value of the key. If not found, def will be return;
// def => default value
func (s *ServerConfig) GetStringOption(key string, def string) string {
	value, ok := s.Options[key]
	if !ok {
		return def
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprint(value)
}

func (s *ServerConfig) GetIntOption(key string, def int) int {
	value, ok := s.Options[key]
	if !ok {
		return def
	}

	if v, ok := value.(int); ok {
		return v
	}

	str := fmt.Sprint(value)
	res, err := strconv.Atoi(str)
	if err != nil {
		glog.Error(context.Background(), "server_config GetIntOption fail", glog.String("key", key))
	}
	return res
}

func (s *ServerConfig) GetStringListOption(key string) []string {
	value, ok := s.Options[key]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		res := make([]string, 0, len(v))
		for _, x := range v {
			res = append(res, fmt.Sprint(x))
		}

		return res
	default:
		return []string{fmt.Sprint(value)}
	}
}
