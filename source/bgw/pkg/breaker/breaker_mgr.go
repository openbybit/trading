package breaker

import (
	"fmt"
	"sync"

	"code.bydev.io/frameworks/byone/core/breaker"
)

type BreakerMgr interface {
	GetOrSet(service, target, method string) breaker.Breaker
	OnConfigUpdate(service string) error
	OnInstanceRemove(service string, ins []string) error
}

type breakerMgr struct {
	lock     sync.RWMutex
	services map[string]*serviceBreakerMgr // service => serviceBreakerMgr
}

func NewBreakerMgr() BreakerMgr {
	return &breakerMgr{
		services: make(map[string]*serviceBreakerMgr),
	}
}

func (b *breakerMgr) GetOrSet(service, target, method string) breaker.Breaker {
	b.lock.RLock()
	sbm, ok := b.services[service]
	b.lock.RUnlock()
	if ok {
		return sbm.getOrSet(target, method)
	}

	b.lock.Lock()
	sbm, ok = b.services[service]
	if ok {
		b.lock.Unlock()
		return sbm.getOrSet(target, method)
	}
	sbm = newServiceBreakerMgr(service)
	b.services[service] = sbm
	b.lock.Unlock()
	return sbm.getOrSet(target, method)
}

func (b *breakerMgr) OnConfigUpdate(service string) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.services, service)
	return nil
}

func (b *breakerMgr) OnInstanceRemove(service string, ins []string) error {
	b.lock.RLock()
	sbm, ok := b.services[service]
	b.lock.RUnlock()
	if !ok {
		return nil
	}
	return sbm.onInstanceRemove(ins)
}

// serviceBreakerMgr 为每个service独享,以减少service之间的相互影响
type serviceBreakerMgr struct {
	lock     sync.RWMutex
	service  string                     // service，对应服务的registry
	breakers map[string]breaker.Breaker // key为target+method，target为service或者ip:port
	methods  map[string]struct{}        // methods, 该service下开启熔断的methods
}

func newServiceBreakerMgr(service string) *serviceBreakerMgr {
	return &serviceBreakerMgr{
		service:  service,
		breakers: make(map[string]breaker.Breaker),
		methods:  make(map[string]struct{}, 0),
	}
}

func (s *serviceBreakerMgr) getOrSet(target, method string) breaker.Breaker {
	key := fmt.Sprintf("%s,%s", target, method)
	s.lock.RLock()
	bkr, ok := s.breakers[key]
	if ok {
		s.lock.RUnlock()
		return bkr
	}
	s.lock.RUnlock()

	s.lock.Lock()
	defer s.lock.Unlock()
	bkr, ok = s.breakers[key]
	if ok {
		return bkr
	}
	bkr = breaker.NewBreaker(breaker.WithName(key))
	s.breakers[key] = bkr
	s.methods[method] = struct{}{}
	return bkr
}

func (s *serviceBreakerMgr) onInstanceRemove(ins []string) error {
	if len(ins) == 0 {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.methods) == 0 {
		return nil
	}

	for m := range s.methods {
		for _, in := range ins {
			key := fmt.Sprintf("%s,%s", in, m)
			delete(s.breakers, key)
		}
	}
	return nil
}
