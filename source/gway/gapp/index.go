package gapp

import (
	"html/template"
	"net/http"
	"os"
	"time"
)

func index(endpoints []Endpoint) http.HandlerFunc {
	started := time.Now()
	pid := os.Getpid()
	wd, _ := os.Getwd()
	hostname, _ := os.Hostname()
	exec := os.Args[0]

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentType, mimeTextHTMLCharsetUTF8)

		err := tpl.Execute(w, map[string]interface{}{
			"Uptime":    time.Since(started).Truncate(time.Second),
			"Endpoints": endpoints,
			"Hostname":  hostname,
			"Pid":       pid,
			"Cwd":       wd,
			"Exec":      exec,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

var (
	tpl *template.Template = template.Must(template.New("index").Parse(`<!doctype html>
<html>
<head>
<meta charset="UTF-8">
<title>BAT Framework</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/water.css@2/out/light.css">
</head>
<body style="max-width: 92vw">
<h1>
<svg style="vertical-align: middle" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64" viewBox="-0.5 -0.5 482 482"><rect width="480" height="480" rx="72" ry="72" fill="#ffb11a" pointer-events="all"/><image x="29.5" y="29.5" width="420" height="420" xlink:href="data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiA/PjxzdmcgaWQ9IkxheWVyXzEiIHN0eWxlPSJlbmFibGUtYmFja2dyb3VuZDpuZXcgMCAwIDI0IDI0OyIgdmVyc2lvbj0iMS4xIiB2aWV3Qm94PSIwIDAgMjQgMjQiIHhtbDpzcGFjZT0icHJlc2VydmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiPjxnIGlkPSJYTUxJRF8xMzUzXyI+PHBhdGggZD0iICAgTTQuMTgsMTkuODE5bDAuMjQyLTAuMzJjMC44MDgtMS4wNjgsMS4zNjgtMi4zMzYsMS40MzItMy42NzRjMC4wMTEtMC4yMzgsMC4wMS0wLjQ3Ni0wLjAwMy0wLjcxMyAgIGMtMC4wOS0xLjYzNy0wLjc1My0zLjI0OC0yLjAwMy00LjQ5OGMtMC4xMjktMC4xMjktMC4yNjUtMC4yNDYtMC40MDItMC4zNjNDNC4yMDEsNy40NTYsMy41NTEsNC4zNjksMS41LDIuMDg5ICAgYzIuNTQtMS4xMjMsNS42MTYtMC42NDYsNy42OTEsMS40NDZjMC4wODgsMC4zODUtMS42ODUsMy40MDYsMS41NzUsNi45MzNjMC4wNDEsMC4wMDIsMC4wNjQsMC4wMDIsMC4xMDQsMC4wMDRsMS43MzktMS43NDN2MS4xNTYgICBjMCwwLjgzMiwwLjY3NCwxLjUwNiwxLjUwNiwxLjUwNmgxLjE1NmwtMS43NzUsMS43NzVjMCwwLDAuMjA5LDAuMjI3LDAuMzMyLDAuMzMyYzMuMzk4LDIuODk5LDYuMjYyLDEuMjI3LDYuNjM2LDEuMzEyICAgYzIuMDkxLDIuMDc1LDIuNTY5LDUuMTUxLDEuNDQ2LDcuNjkxYy0yLjI4LTIuMDUxLTUuMzY4LTIuNzAxLTguMTYyLTEuOTQ2Yy0wLjExNy0wLjEzNi0wLjIzNC0wLjI3My0wLjM2My0wLjQwMiAgIGMtMS4yNS0xLjI1LTIuODYyLTEuOTEzLTQuNDk4LTIuMDAzYy0wLjIzNy0wLjAxMy0wLjQ3NS0wLjAxNC0wLjcxMy0wLjAwM2MtMS4zMzgsMC4wNjQtMi42MDUsMC42MjQtMy42NzQsMS40MzJMNC4xOCwxOS44MTl6IiBpZD0iWE1MSURfNDJfIiBzdHlsZT0iZmlsbDojMDAwMDAwO3N0cm9rZTojMDAwMDAwO3N0cm9rZS1saW5lY2FwOnJvdW5kO3N0cm9rZS1saW5lam9pbjpyb3VuZDtzdHJva2UtbWl0ZXJsaW1pdDoxMDsiLz48L2c+PC9zdmc+"/></svg>
&nbsp;
BAT Framework
</h1>
<em>THIS SHOULD NEVER BE EXPOSED</em>

<hr>
<table style="max-width: 800px;" border="1">
<tr>
<th>hostname</th><td>{{.Hostname}}</td>
</tr><tr>
<th>pid</th><td>{{.Pid}}</td>
</tr><tr>
<th>cwd</th><td>{{.Cwd}}</td>
</tr><tr>
<th>exec</th><td>{{.Exec}}</td>
</tr><tr>
<th>uptime</th><td>{{.Uptime}}</td>
</tr>
</table>

<section>
<h3>Available endpoints:</h3>
<dl>
{{range $ep := .Endpoints}}
<dt><strong>{{$ep.Title}}</strong></dt>
<dd>
<a href="{{$ep.Index}}">{{$ep.Index}}</a>
</dd>
{{end}}
</dl>
</table>
</section>

</body>
</html>`))
)
