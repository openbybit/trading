package ws

import (
	"context"
	"fmt"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/deadlock"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var gAcceptorMgr = newAcceptorMgr()

func GetAcceptorMgr() AcceptorMgr {
	return gAcceptorMgr
}

func newAcceptorMgr() *acceptorMgr {
	m := &acceptorMgr{
		acceptorMap: make(map[string]Acceptor),
		topics:      make(map[string][]Acceptor),
	}
	return m
}

type AcceptorMgr interface {
	Size() int
	Get(id string) Acceptor
	GetByIndex(index int) Acceptor
	GetByAppID(id string) []Acceptor
	GetAll() []Acceptor
	GetByTopics(topics []string) []Acceptor
}

type acceptorMgr struct {
	acceptors   []Acceptor            // add or remove will copy a new list
	acceptorMap map[string]Acceptor   // id -> acceptor
	topics      map[string][]Acceptor // topic -> acceptor list
	mux         deadlock.RWMutex
}

func (m *acceptorMgr) Size() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.acceptors)
}

func (m *acceptorMgr) Get(id string) Acceptor {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.acceptorMap[id]
}

func (m *acceptorMgr) GetByIndex(index int) Acceptor {
	m.mux.RLock()
	defer m.mux.RUnlock()
	if len(m.acceptors) > 0 {
		index = index % len(m.acceptors)
		return m.acceptors[index]
	}

	return nil
}

func (m *acceptorMgr) GetByAppID(appId string) []Acceptor {
	m.mux.RLock()
	res := make([]Acceptor, 0, len(m.acceptorMap))
	for _, a := range m.acceptors {
		if a.AppID() == appId {
			res = append(res, a)
		}
	}
	m.mux.RUnlock()
	return res
}

func (m *acceptorMgr) GetAll() []Acceptor {
	m.mux.RLock()
	res := m.acceptors
	m.mux.RUnlock()
	return res
}

func (m *acceptorMgr) GetByTopics(topics []string) []Acceptor {
	m.mux.RLock()
	defer m.mux.RUnlock()

	switch len(topics) {
	case 0:
		return nil
	case 1:
		return m.topics[topics[0]]
	default:
		// remove duplicate acceptor
		ta := make(map[string]Acceptor, 0)
		for _, t := range topics {
			if list, ok := m.topics[t]; ok {
				for _, ac := range list {
					ta[ac.ID()] = ac
				}
			}
		}

		cs := make([]Acceptor, 0, len(ta))
		for _, a := range ta {
			cs = append(cs, a)
		}
		return cs

	}
}

func (m *acceptorMgr) Close() {
	glog.Info(context.Background(), "start to stop acceptor_mgr")
	m.mux.Lock()
	acceptors := make([]Acceptor, 0, len(m.acceptors))
	acceptors = append(acceptors, m.acceptors...)
	m.mux.Unlock()

	for _, acc := range acceptors {
		acc.Close()
	}

	sconf := getDynamicConf()
	if sconf.EnableGracefulClose {
		glog.Info(context.Background(), "acceptor_mgr wait for graceful exit")
		// 等待优雅退出
		start := time.Now()
		for {
			if m.Size() == 0 {
				break
			}

			if time.Since(start) > defaultQuitWaitTime {
				break
			}

			time.Sleep(time.Second)
		}
	} else {
		glog.Info(context.Background(), "acceptor_mgr ignore graceful close")
	}

	glog.Info(context.Background(), "stop acceptor_mgr finish")
}

func (m *acceptorMgr) Add(a *acceptor) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, ok := m.acceptorMap[a.ID()]; ok {
		return fmt.Errorf("duplicate acceptor id: %v", a.ID())
	}

	acceptors := make([]Acceptor, len(m.acceptors)+1)
	copy(acceptors, m.acceptors)
	acceptors[len(m.acceptors)] = a

	m.acceptors = acceptors
	m.acceptorMap[a.ID()] = a

	for _, t := range a.topics {
		m.topics[t] = append(m.topics[t], a)
	}

	return nil
}

func (m *acceptorMgr) Remove(id string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	_, ok := m.acceptorMap[id]
	if !ok {
		return
	}

	delete(m.acceptorMap, id)
	glog.Infof(context.Background(), "acceptor_mgr remove acceptor, id: %v, left: %v", id, len(m.acceptorMap))

	acceptors := make([]Acceptor, 0, len(m.acceptors)-1)
	for _, a := range m.acceptors {
		if a.ID() != id {
			acceptors = append(acceptors, a)
		}
	}
	m.acceptors = acceptors

	for key, list := range m.topics {
		for idx, ac := range list {
			if ac.ID() == id {
				x := append(list[:idx], list[idx+1:]...)
				m.topics[key] = x
				break
			}
		}
	}
}

func (m *acceptorMgr) RefreshAppIDGauge(appID string) {
	count := len(m.GetByAppID(appID))
	WSGauge(float64(count), "acceptor_manager", appID)
}
