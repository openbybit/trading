package http

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

func IsContentType(source, target string) bool {
	return strings.HasPrefix(source, target)
}

func parseContentType(s string) string {
	if s == "" {
		return s
	}

	index := strings.LastIndexByte(s, ';')
	if index != -1 {
		return strings.TrimSpace(s[:index])
	}

	return strings.TrimSpace(s)
}

func encode(contentType string, data interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	switch d := data.(type) {
	case string:
		return []byte(d), nil
	case []byte:
		return d, nil
	}

	switch contentType {
	case MIMEApplicationJSON:
		return json.Marshal(data)
	case MIMEApplicationXML:
		return xml.Marshal(data)
	case MIMEApplicationForm:
		uv, err := toUrlValue(data)
		if err != nil {
			return nil, err
		}
		r := uv.Encode()
		return []byte(r), nil
	default:
		return nil, ErrNotSupport
	}
}

func decode(contentType string, data []byte, result interface{}) error {
	if result == nil {
		return nil
	}

	if len(data) == 0 {
		return nil
	}

	v := reflect.ValueOf(result)
	t := v.Type()
	if t.Kind() != reflect.Ptr {
		return ErrInvalidPointerType
	}

	switch v := result.(type) {
	case *string:
		*v = string(data)
		return nil
	case *[]byte:
		*v = data
		return nil
	}

	switch contentType {
	case MIMEApplicationJSON:
		return json.Unmarshal(data, result)
	case MIMEApplicationXML:
		return xml.Unmarshal(data, result)
	case MIMEApplicationForm:
		values, err := url.ParseQuery(string(data))
		if err != nil {
			return err
		}

		return parseUrlValue(values, result)
	default:
		return ErrNotSupport
	}
}

func parseUrlValue(values url.Values, result interface{}) error {
	switch r := result.(type) {
	case *url.Values:
		*r = values
	case url.Values:
		for k, v := range values {
			r[k] = v
		}
	case *map[string]string:
		m := make(map[string]string)
		for k, v := range values {
			m[k] = v[0]
		}
		*r = m
	case *map[string]interface{}:
		m := make(map[string]interface{})
		for k, v := range values {
			m[k] = v[0]
		}
		*r = m
	case map[string]string:
		for k, v := range values {
			r[k] = v[0]
		}
	case map[string]interface{}:
		for k, v := range values {
			r[k] = v[0]
		}
	default:
		return ErrNotSupport
	}

	return nil
}

func toUrlValue(data interface{}) (url.Values, error) {
	switch m := data.(type) {
	case url.Values:
		return m, nil
	case map[string]string:
		r := make(url.Values)
		for k, v := range m {
			r.Add(k, v)
		}
		return r, nil
	case map[string]interface{}:
		r := url.Values{}
		for k, v := range m {
			kind := reflect.TypeOf(v).Kind()
			if kind <= reflect.Float64 {
				r.Add(k, fmt.Sprintf("%+v", v))
			} else if kind == reflect.Slice {
				vv := reflect.ValueOf(v)
				for i := 0; i < vv.Len(); i++ {
					f := vv.Field(i)
					r.Add(k, fmt.Sprintf("%+v", f.Interface()))
				}
			} else {
				return nil, ErrNotSupport
			}
		}
		return r, nil
	default:
		return nil, ErrNotSupport
	}
}

func toUrl(baseUrl string, pathUrl string) string {
	if baseUrl == "" {
		return pathUrl
	}

	if len(pathUrl) > 8 {
		if strings.Index(pathUrl, "://") != -1 {
			return pathUrl
		}
	}

	baseUrl = strings.TrimSuffix(baseUrl, "/")
	if strings.HasPrefix(pathUrl, "/") {
		return fmt.Sprintf("%s%s", baseUrl, pathUrl)
	} else {
		return fmt.Sprintf("%s/%s", baseUrl, pathUrl)
	}
}

var namedReg = regexp.MustCompile(":\\w+")

// replacePathParams 替换path中参数,格式为:named
func replacePathParams(url string, params map[string]string) string {
	if len(params) == 0 {
		return url
	}

	url = namedReg.ReplaceAllStringFunc(url, func(s string) string {
		if x, ok := params[s[1:]]; ok {
			return x
		}

		return s
	})

	return url
}

func merge(old, added url.Values) url.Values {
	if old == nil {
		return added
	}

	for k, v := range added {
		for _, vv := range v {
			old.Add(k, vv)
		}
	}

	return old
}
