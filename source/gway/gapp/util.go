package gapp

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// ToHandlerFunc convert func to http.HandlerFunc
func ToHandlerFunc(in interface{}) http.HandlerFunc {
	switch h := in.(type) {
	case http.HandlerFunc:
		return h
	case http.Handler:
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					log.Printf("[gapp] invoke http.Handler panic, err=%v, stack=%s\n", e, string(debug.Stack()))
				}
			}()
			h.ServeHTTP(w, r)
		}
	case func(r *http.Request) (interface{}, error):
		type errorCode interface {
			Code() int
		}

		// 标准返回结果,如果有错误或者返回结果为空则使用此结构体返回,否则直接返回数据
		type response struct {
			Code    int         `json:"code,omitempty"`
			Message string      `json:"message,omitempty"`
			Data    interface{} `json:"data,omitempty"`
		}
		// 快捷方式, 返回interface{}和error,结果会序列化成json返回
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					log.Printf("[gapp] invoke handler panic, err=%v, stack=%s\n", e, string(debug.Stack()))
				}
			}()

			res, err := h(r)
			w.Header().Set(headerContentType, mimeJsonCharsetUTF8)
			var result interface{}
			if err != nil {
				rsp := &response{}
				if ec, ok := err.(errorCode); ok {
					rsp.Code = ec.Code()
				} else {
					rsp.Code = http.StatusInternalServerError
				}
				rsp.Message = err.Error()
				rsp.Data = res
				result = rsp
			} else if res == nil {
				result = &response{
					Code:    http.StatusOK,
					Message: http.StatusText(http.StatusOK),
				}
			} else {
				// 直接返回原始数据,不用response包装,避免多一层嵌套
				result = res
			}

			data, _ := json.MarshalIndent(result, "", "\t")
			if _, err = w.Write(data); err != nil {
				log.Printf("write http response fail,path=%v, err=%v\n", r.URL.Path, err)
			}
		}
	default:
		panic("invalid http.HandlerFunc")
	}
}
