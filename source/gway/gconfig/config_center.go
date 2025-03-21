package gconfig

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotFound 当没有数据时,会返回ErrNotFound需要调用者判断要不要报错
	ErrNotFound = errors.New("not found")
	// ErrInvalidInstance 没有初始化global configure
	ErrInvalidInstance = errors.New("invalid instance")
	// ErrInvalidType 传入的必须是结构体指针的指针
	ErrInvalidType = errors.New("invalid type, must be a pointer to a structure pointer")
	// ErrInvalidFormat 未知格式
	ErrInvalidFormat = errors.New("invalid format")
	// ErrInvalidParams 参数错误
	ErrInvalidParams = errors.New("invalid parameters")
	// ErrPutFailure 更新失败
	ErrPutFailure = errors.New("put failure")
	// ErrDelFailure 删除失败
	ErrDelFailure = errors.New("del failure")
	// ErrDuplicateListen 重复监听
	ErrDuplicateListen = errors.New("duplicate listen")
)

type EventType uint8

const (
	EventTypeCreate EventType = iota
	EventTypeUpdate
	EventTypeDelete
)

func (t EventType) String() string {
	switch t {
	case EventTypeCreate:
		return "create"
	case EventTypeUpdate:
		return "update"
	case EventTypeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

type Event struct {
	Type  EventType
	Key   string
	Value string
}

type Listener interface {
	OnEvent(ev *Event)
}

type ListenFunc func(ev *Event)

func (f ListenFunc) OnEvent(ev *Event) {
	f(ev)
}

type Logger interface {
	Infof(ctx context.Context, format string, args ...interface{})
	Errorf(ctx context.Context, format string, args ...interface{})
}

// Configure 配置管理中心
type Configure interface {
	Get(ctx context.Context, key string, opts ...Option) (string, error)
	Put(ctx context.Context, key string, value string, opts ...Option) error
	Delete(ctx context.Context, key string, opts ...Option) error
	Listen(ctx context.Context, key string, listener Listener, opts ...Option) error
}

// CreateFunc 创建Configure构造函数
type CreateFunc func(url string) (Configure, error)

var createMap = make(map[string]CreateFunc)

// Register 注册构造函数,默认为空,需要初始化时显示注册nacos或者etcd
func Register(name string, fn CreateFunc) {
	createMap[name] = fn
}

// New 通过url参数构造Configure,需要提前注册构造函数
func New(url string) (Configure, error) {
	idx := strings.Index(url, "://")
	if idx == -1 {
		return nil, fmt.Errorf("invalid url format: %s", url)
	}
	scheme := url[:idx]
	fn, ok := createMap[scheme]
	if !ok {
		return nil, fmt.Errorf("invalid scheme: %s", scheme)
	}

	return fn(url)
}
