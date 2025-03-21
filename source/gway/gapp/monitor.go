package gapp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type monitor struct {
	srv *http.Server
}

func (m *monitor) Start(addr string, endpoints []Endpoint) {
	if addr == "" {
		log.Println("gapp: address is empty, disable monitoring")
		return
	}

	router := httprouter.New()

	for _, ep := range endpoints {
		if ep.Method == "" {
			ep.Method = http.MethodGet
		}

		if ep.Index == "" {
			ep.Index = ep.Route
		}

		methods := strings.Split(ep.Method, ",")
		for _, m := range methods {
			m = strings.TrimSpace(m)
			router.HandlerFunc(m, ep.Route, ep.Handler)
		}
	}

	router.HandlerFunc("GET", "/", index(endpoints))

	s := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	m.srv = s

	go func() {
		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("gapp: start monitoring server fail: addr=%v, err=%v\n", addr, err)
		}
	}()
}

func (m *monitor) Stop() {
	if m.srv != nil {
		if err := m.srv.Shutdown(context.Background()); err != nil {
			log.Println("srv Shutdown error:", err)
		}
		m.srv = nil
	}
}
