package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

const (
	serviceName = "bgws"
)

const (
	urlParamKeyPlatformUnderline = "_platform" // platform, web, app
	urlParamKeyPlatform          = "platform"  // platform
	urlParamKeySource            = "source"    // web source
	urlParamKeyVersion           = "version"   // app version
	urlParamKeyVersionShort      = "v"         //
)

const (
	// listenTypeUnix = "unix"
	listenTypeTcp = "tcp"
	listenTypeAll = "all"
)

const (
	defaultErrorCode = 10000
)

type (
	Upgrader       = websocket.FastHTTPUpgrader // Upgrader is a websocket.Upgrader with some default values
	WSConn         = websocket.Conn             // WSConn is a websocket connection
	RequestHandler = fasthttp.RequestHandler
)

type rateLimit struct {
	mux          sync.Mutex
	rate         int64
	internal     time.Duration
	token        int64
	lastRestTime time.Time
}

func newRateLimit(rate int64, internal time.Duration) *rateLimit {
	limit := &rateLimit{}
	limit.Set(rate, internal)
	return limit
}

func (l *rateLimit) Set(rate int64, internal time.Duration) {
	if internal <= 0 {
		internal = time.Second
	}
	l.rate = rate
	l.internal = internal
}

func (l *rateLimit) Allow() bool {
	if l.rate == 0 {
		return true
	}

	l.mux.Lock()
	defer l.mux.Unlock()

	now := time.Now()
	if now.Sub(l.lastRestTime) > l.internal {
		l.token = l.rate
		l.lastRestTime = now
	}

	if l.token <= 0 {
		return false
	} else {
		l.token--
		return true
	}
}

// Status returns the status of the service
type Status struct {
	Health       bool
	Core         json.RawMessage `json:"Core,omitempty"`
	UserCount    int
	SessionCount int
	Users        map[int64]string
	Sessions     []string
}

// CodeError is an error with a code.
type CodeError interface {
	error
	Code() int
}

func toCodeErr(err error) CodeError {
	if err == nil {
		return nil
	}
	if ce, ok := err.(CodeError); ok {
		return ce
	}

	return newCodeErr(defaultErrorCode, err.Error())
}

func isError(from CodeError, target CodeError) bool {
	return from.Code() == target.Code()
}

func newCodeErr(code int, format string, args ...interface{}) CodeError {
	msg := fmt.Sprintf(format, args...)
	e := &codeErr{
		code: code,
		msg:  msg,
		str:  msg,
	}

	return e
}

// newCodeErrFrom 提供一些debug信息,但正式环境并不发送给用户
func newCodeErrFrom(parent CodeError, format string, args ...interface{}) CodeError {
	e := &codeErr{
		code: parent.Code(),
		msg:  parent.Error(),
		str:  parent.Error() + ":" + fmt.Sprintf(format, args...),
	}

	return e
}

type codeErr struct {
	code int
	msg  string
	str  string
}

func (e *codeErr) Code() int {
	return e.code
}

func (e *codeErr) Error() string {
	if env.IsProduction() {
		return e.msg
	} else {
		return e.str
	}
}

type logWriter struct {
	mux   sync.RWMutex
	buf   bytes.Buffer
	count int32
}

func (w *logWriter) Write(p []byte) (int, error) {
	glog.Error(context.TODO(), cast.UnsafeBytesToString(p))

	w.mux.Lock()
	if w.count < 50 {
		w.count++
		w.buf.Write(p)
		w.buf.WriteByte('\n')
	}
	w.mux.Unlock()

	return 0, nil
}

func (w *logWriter) String() string {
	w.mux.RLock()
	res := w.buf.String()
	w.mux.RUnlock()
	return res
}

type Int64Set map[int64]struct{}

func (s *Int64Set) UnmarshalJSON(data []byte) error {
	*s = make(map[int64]struct{})
	if bytes.HasPrefix(data, []byte("[")) {
		items := make([]int64, 0)
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, v := range items {
			(*s)[v] = struct{}{}
		}
	} else {
		items := make(map[int64]bool)
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for k, v := range items {
			if v {
				(*s)[k] = struct{}{}
			}
		}
	}

	return nil
}

func (s *Int64Set) UnmarshalYAML(unmarshal func(interface{}) error) error {
	items := make([]int64, 0)
	if err := unmarshal(&items); err == nil {
		*s = make(map[int64]struct{})
		for _, v := range items {
			(*s)[v] = struct{}{}
		}

		return nil
	}

	items1 := make(map[int64]bool)
	if err := unmarshal(&items1); err != nil {
		return err
	}

	*s = make(map[int64]struct{})
	for k, v := range items1 {
		if v {
			(*s)[k] = struct{}{}
		}
	}

	return nil
}

type StringSet map[string]struct{}

func (s *StringSet) Add(key string) {
	(*s)[key] = struct{}{}
}

func (s StringSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s *StringSet) UnmarshalJSON(data []byte) error {
	*s = make(map[string]struct{})
	if bytes.HasPrefix(data, []byte("[")) {
		items := make([]string, 0)
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, v := range items {
			v = strings.TrimSpace(v)
			if v != "" {
				(*s)[v] = struct{}{}
			}
		}
	} else {
		items := make(map[string]bool)
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for k, v := range items {
			k = strings.TrimSpace(k)
			if v && k != "" {
				(*s)[k] = struct{}{}
			}
		}
	}

	return nil
}

func (s *StringSet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	items := make([]string, 0)
	if err := unmarshal(&items); err == nil {
		*s = make(map[string]struct{})
		for _, v := range items {
			v = strings.TrimSpace(v)
			if v != "" {
				(*s)[v] = struct{}{}
			}
		}

		return nil
	}

	items1 := make(map[string]bool)
	if err := unmarshal(&items1); err != nil {
		return err
	}

	*s = make(map[string]struct{})
	for k, v := range items1 {
		k = strings.TrimSpace(k)
		if v && k != "" {
			(*s)[k] = struct{}{}
		}
	}

	return nil
}
