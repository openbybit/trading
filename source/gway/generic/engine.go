package generic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

// Engine generic engine
type Engine struct {
	sync.RWMutex
	// key: namespace, value: descriptorController
	controllers map[string]*controller
	// key: namespace, value: checksum
	cache map[string]string
}

// NewEngine create a new generic engine
func NewEngine(options ...Option) *Engine {
	e := new(Engine)

	// default values
	e.controllers = make(map[string]*controller)
	e.cache = make(map[string]string)

	for _, opt := range options {
		if opt != nil {
			opt(e)
		}
	}

	return e
}

// isCached check if the namespace is cached
func (e *Engine) isCached(key string, data []byte) bool {
	e.RLock()
	defer e.RUnlock()

	c, ok := e.cache[key]
	if !ok {
		return false
	}

	return c == checksum(data)
}

// Update descriptors and create new invoker
func (e *Engine) Update(key string, data []byte) (cached bool, err error) {
	if e.isCached(key, data) {
		return true, nil
	}

	c := newController()
	if err = c.init(data); err != nil {
		return
	}

	e.Lock()
	e.controllers[key] = c
	e.cache[key] = checksum(data)
	e.Unlock()

	return
}

// Invoke grpc invoke
func (e *Engine) Invoke(ctx context.Context, conn grpc.ClientConnInterface, request Request, result Result) (err error) {
	key := request.GetNamespace()

	e.RWMutex.RLock()
	c, ok := e.controllers[key]
	if !ok {
		e.RWMutex.RUnlock()
		return fmt.Errorf("controller not found of namespace: %s", key)
	}
	e.RWMutex.RUnlock()

	return c.invokeUnary(ctx, conn, request, result)
}

// ListServices list all services
func (e *Engine) ListServices() ([]string, error) {
	ss := make([]string, 0)

	e.RLock()
	defer e.RUnlock()

	for _, v := range e.controllers {
		if v == nil {
			continue
		}

		s, err := ListServices(v.descriptor)
		if err == nil {
			ss = append(ss, s...)
		}
	}

	return ss, nil
}

// ListMethods list all methods of service
func (e *Engine) ListMethods(svc string) ([]string, error) {
	ms := make([]string, 0)

	e.RLock()
	defer e.RUnlock()

	for _, v := range e.controllers {
		if v == nil {
			continue
		}
		m, err := ListMethods(v.descriptor, svc)
		if err == nil {
			ms = append(ms, m...)
		}
	}

	return ms, nil
}

// ServMethodModel service and method model
type ServMethodModel struct {
	PackageName     string
	ServiceName     string
	FullServiceName string
	MethodName      string
	FullMethodName  string
}

// ListServiceAndMethods list all services and methods
func (e *Engine) ListServiceAndMethods() (map[string][]ServMethodModel, error) {
	servList, err := e.ListServices()
	if err != nil {
		return nil, err
	}

	m := map[string][]ServMethodModel{}
	for _, svc := range servList {
		fullMethodList, err := e.ListMethods(svc)
		servMethodModelList := []ServMethodModel{}
		for _, method := range fullMethodList {
			cs := strings.Split(method, ".")
			if len(cs) < 3 {
				return nil, errors.New("method split failed")
			}

			dto := ServMethodModel{
				MethodName:      cs[len(cs)-1],
				ServiceName:     cs[len(cs)-2],
				PackageName:     strings.Join(cs[:len(cs)-2], "."),
				FullMethodName:  method,
				FullServiceName: svc,
			}
			servMethodModelList = append(servMethodModelList, dto)
		}

		if err != nil {
			return nil, err
		}

		m[svc] = servMethodModelList
	}

	return m, nil
}
