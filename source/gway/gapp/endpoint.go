package gapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	headerContentType = "Content-Type"

	mimeTextPlainCharsetUTF8 = "text/plain; charset=utf-8"
	mimeTextHTMLCharsetUTF8  = "text/html; charset=utf-8"
	mimeJsonCharsetUTF8      = "application/json; charset=utf-8"
)

// HealthFunc define health check callback
type HealthFunc func() (bool, interface{})

// alias http type
type (
	Params         = httprouter.Params
	Request        = http.Request
	ResponseWriter = http.ResponseWriter
)

// ParamsFromContext get path params from context
func ParamsFromContext(ctx context.Context) httprouter.Params {
	return httprouter.ParamsFromContext(ctx)
}

type Endpoint struct {
	// Method the http method, ALLCAPS
	Method  string
	Route   string
	Index   string
	Title   string
	Handler http.HandlerFunc
}

func newPrometheusEndpoint() Endpoint {
	return Endpoint{
		Route:   "/arch/metrics",
		Index:   "/arch/metrics",
		Title:   "prometheus endpoint",
		Handler: ToHandlerFunc(promhttp.Handler()),
	}
}

func newPprofEndpoint() Endpoint {
	return Endpoint{
		Route:   "/debug/pprof/*remain",
		Index:   "/debug/pprof/",
		Title:   "pprof tools",
		Handler: onPprof,
	}
}

func newHealthEndpoint(cb HealthFunc) Endpoint {
	return Endpoint{
		Method:  "POST,GET",
		Route:   "/arch/health",
		Index:   "/arch/health",
		Title:   "health check",
		Handler: ToHealthHandlerFunc(cb),
	}
}

func newAdminEndpoint(route string) Endpoint {
	if route == "" {
		route = "/admin"
	}
	return Endpoint{
		Method:  "POST,GET",
		Index:   route,
		Route:   route,
		Title:   "admin",
		Handler: onAdminHandler,
	}
}

func onPprof(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/debug/pprof/cmdline"):
		pprof.Cmdline(w, r)
	case strings.HasPrefix(path, "/debug/pprof/profile"):
		pprof.Profile(w, r)
	case strings.HasPrefix(path, "/debug/pprof/symbol"):
		pprof.Symbol(w, r)
	case strings.HasPrefix(path, "/debug/pprof/trace"):
		pprof.Trace(w, r)
	default:
		pprof.Index(w, r)
	}
}

var globalElbCode = int32(http.StatusOK)

func ToHealthHandlerFunc(cb HealthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			codeStr := r.FormValue("code")
			code, err := strconv.Atoi(codeStr)
			if err == nil {
				atomic.StoreInt32(&globalElbCode, int32(code))
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set(headerContentType, mimeTextPlainCharsetUTF8)
			elbCode := int(atomic.LoadInt32(&globalElbCode))
			body := fmt.Sprintf("current_code: %v", elbCode)
			_, _ = w.Write([]byte(body))
			return
		}

		elbCode := int(atomic.LoadInt32(&globalElbCode))
		health, data := cb()
		if health {
			w.WriteHeader(elbCode)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		var body []byte
		if data != nil {
			switch x := data.(type) {
			case []byte:
				body = x
			case string:
				body = []byte(x)
			default:
				res, _ := json.MarshalIndent(data, "", "  ")
				body = res
			}
		}

		if body != nil {
			w.Header().Set(headerContentType, mimeTextPlainCharsetUTF8)
			_, _ = w.Write(body)
		}
	}
}
