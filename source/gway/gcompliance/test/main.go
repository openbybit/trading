package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
)

func main() {
	h := &handler{
		newWall(),
	}

	r := request{}
	d, _ := json.Marshal(r)
	log.Println(string(d))

	err := http.ListenAndServe("127.0.0.1:9999", h)
	if err != nil {
		log.Printf("err = %s", err.Error())
	}
}

type handler struct {
	gcompliance.Wall
}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r := &request{}
	err := json.Unmarshal(body, r)
	if err != nil {
		_, _ = res.Write([]byte(err.Error()))
		return
	}

	resp, _, err := h.CheckStrategy(context.Background(), r.Broker, r.Site, r.Scene, r.Uid, r.Country, r.SubV, r.Source, r.UserSiteID)
	if err != nil {
		_, _ = res.Write([]byte(err.Error()))
		return
	}
	_, _ = res.Write([]byte(resp.GetEndPointExec()))
	_, _ = res.Write([]byte("\n"))
	_, cfg, err := h.GetSiteConfig(context.Background(), r.Broker, r.Uid, r.Site, r.Product)
	if err != nil {
		_, _ = res.Write([]byte(err.Error()))
		return
	}

	data, _ := json.Marshal(cfg)
	_, _ = res.Write(data)
}

func newWall() gcompliance.Wall {
	addr := "dns:///compliance-wall-cht-test-1.test.efficiency.ww5sawfyut0k.bitsvc.io:9090"
	wall, err := gcompliance.NewWall(gcompliance.NewAddrCfg(addr), true)
	if err != nil {
		return nil
	}
	return wall
}

type request struct {
	Broker     int32
	Site       string
	Scene      string
	Uid        int64
	Country    string
	SubV       string
	Source     string
	UserSiteID string
	Product    string
}
