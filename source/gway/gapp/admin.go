package gapp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
)

type AdminFunc func(args AdminArgs) (interface{}, error)

type adminInfo struct {
	name    string
	help    string
	handler AdminFunc
}

var adminMap = make(map[string]*adminInfo)

func init() {
	RegisterAdmin("help", "help", onHelp)
}

func RegisterAdmin(cmd string, help string, fn AdminFunc) {
	adminMap[cmd] = &adminInfo{name: cmd, help: help, handler: fn}
}

// GET: cmd=xxx&params=p1,p2,p3&key1=val1&key2=val2
// POST:
func onAdminHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("[gapp] invoke admin panic, err=%v, stack=%s\n", e, string(debug.Stack()))
		}
	}()

	args := AdminArgs{Options: map[string]string{}}
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Write([]byte("invalid body"))
			return
		}

		bodyStr := strings.TrimSpace(string(body))
		if strings.HasPrefix(bodyStr, "{") {
			type Request struct {
				Args string `json:"args"`
			}
			req := &Request{}
			if err := json.Unmarshal(body, req); err != nil {
				w.Write([]byte("unmarshal fail"))
				return
			}
			bodyStr = req.Args
		}

		// 要求admin格式为: cmd params options(k=v)
		tokens := strings.Fields(strings.TrimSpace(bodyStr))
		if len(tokens) == 0 {
			w.Write([]byte("empty params"))
			return
		}
		args.Cmd = tokens[0]
		// parse params and options
		for i := 1; i < len(tokens); i++ {
			t := tokens[i]
			kv := strings.SplitN(t, "=", 2)
			if len(kv) == 0 {
				continue
			}
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				key = strings.TrimPrefix(key, "--")
				args.Options[key] = strings.TrimSpace(kv[1])
			} else if len(kv) == 1 {
				args.Params = append(args.Params, strings.TrimSpace(kv[0]))
			}
		}
	} else {
		// 格式: cmd=xxx&params=xxx,xxx,xxx&{k1}={v1}&{k2}={v2}
		// 其中cmd和params是固定名字不能重复
		q := r.URL.Query()
		args.Cmd = q.Get("cmd")
		params := q.Get("params")
		if params != "" {
			args.Params = strings.Split(params, ",")
		}
		for k, v := range q {
			if k == "cmd" || k == "params" {
				continue
			}
			if len(v) == 0 {
				continue
			}
			args.Options[k] = v[0]
		}
	}

	data, err := invokeAdmin(args)
	w.Header().Add(headerContentType, mimeJsonCharsetUTF8)
	w.WriteHeader(200)
	_, _ = w.Write(data)
	logf("[gapp][admin] req=%v, err=%v", args, err)
}

func invokeAdmin(args AdminArgs) ([]byte, error) {
	var result interface{}
	var err error
	args.Cmd = strings.TrimSpace(args.Cmd)
	cmd := adminMap[args.Cmd]
	if cmd != nil {
		result, err = cmd.handler(args)
	} else {
		err = fmt.Errorf("not found admin handler, cmd=%s", args.Cmd)
	}

	if err != nil {
		type errResponse struct {
			Error string `json:"error"`
		}
		result = &errResponse{Error: err.Error()}
	}

	if result != nil {
		data, _ := json.MarshalIndent(result, "", "  ")
		return data, err
	}

	return []byte("{}"), nil
}

func onHelp(args AdminArgs) (interface{}, error) {
	var message []string
	for k, v := range adminMap {
		if k == "help" {
			continue
		}
		m := fmt.Sprintf("%s: %s", k, v.help)
		message = append(message, m)
	}

	sort.Strings(message)
	return message, nil
}

// xxx {params} --opt_key=opt_value
type AdminArgs struct {
	Cmd     string
	Params  []string
	Options map[string]string
}

func (a *AdminArgs) ParamSize() int {
	return len(a.Params)
}

func (a *AdminArgs) GetIntAt(index int) int {
	if index < len(a.Params) {
		r, _ := strconv.Atoi(a.Params[index])
		return r
	}

	return 0
}

func (a *AdminArgs) GetInt64At(index int) int64 {
	if index < len(a.Params) {
		v := strings.TrimSpace(a.Params[index])
		r, _ := strconv.ParseInt(v, 10, 64)
		return r
	}

	return 0
}

func (a *AdminArgs) GetStringAt(index int) string {
	if index < len(a.Params) {
		return a.Params[index]
	}

	return ""
}

func (a *AdminArgs) GetIntBy(key string) int {
	v := strings.TrimSpace(a.Options[key])
	if v != "" {
		r, _ := strconv.Atoi(v)
		return r
	}

	return 0
}

func (a *AdminArgs) GetInt64By(key string) int64 {
	v := strings.TrimSpace(a.Options[key])
	if v != "" {
		r, _ := strconv.ParseInt(v, 10, 64)
		return r
	}

	return 0
}

func (a *AdminArgs) GetStringBy(key string) string {
	return a.Options[key]
}
