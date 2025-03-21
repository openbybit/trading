package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

// StatusErr 当返回码非http.StatusOk时，返回此错误
type StatusErr struct {
	Code int    `json:"code"`
	Info string `json:"info"`
}

func (e *StatusErr) Error() string {
	return fmt.Sprintf("invalid http status,code=%+v, info=%+v", e.Code, e.Info)
}

// Response wrapper http.Response
type Response struct {
	*http.Response
	err error
}

func (r Response) Error() error {
	return r.err
}

func (r Response) Decode(out interface{}) error {
	if r.err != nil {
		return r.err
	}

	contentType := parseContentType(r.Header.Get(HeaderContentType))
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return decode(contentType, body, out)
}
