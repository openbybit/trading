package gconfig

import "context"

var global Configure

// SetDefault 设置默认configure
func SetDefault(c Configure) {
	global = c
}

// Default 返回默认configure
func Default() Configure {
	return global
}

// Load 自动读取key对应的value,并监听value变化并自动反序列化到out结构体中,要求out是指针的指针,用于保证原子性替换整个指针
// 同时如果out结构体实现了OnInit和OnLoaded接口也会自动执行对应方法
// OnInit用于初始化默认值
// OnLoaded用于数据解析后做一些逻辑处理
func Load(ctx context.Context, conf Configure, key string, out interface{}, unmarshal UnmarshalFunc, logger Logger, opts ...Option) error {
	if conf == nil {
		conf = global
	}

	if conf == nil {
		return ErrInvalidInstance
	}

	return conf.Listen(ctx, key, ListenFunc(func(ev *Event) {
		err := Unmarshal([]byte(ev.Value), out, unmarshal)
		if logger != nil {
			if err != nil {
				logger.Errorf(ctx, "nacos load config fail, key: %v, error: %v", key, err)
			} else {
				logger.Infof(ctx, "nacos load config success, key: %s", key)
			}
		}
	}), WithForceGet(true))
}

// Get ...
func Get(ctx context.Context, key string, opts ...Option) (string, error) {
	if global != nil {
		return global.Get(ctx, key, opts...)
	}

	return "", ErrInvalidInstance
}

// Put ...
func Put(ctx context.Context, key string, value string, opts ...Option) error {
	if global != nil {
		return global.Put(ctx, key, value, opts...)
	}

	return ErrInvalidInstance
}

// Delete ...
func Delete(ctx context.Context, key string, opts ...Option) error {
	if global != nil {
		return global.Delete(ctx, key, opts...)
	}

	return ErrInvalidInstance
}

// Listen ...
func Listen(ctx context.Context, key string, listener Listener, opts ...Option) error {
	if global != nil {
		return global.Delete(ctx, key, opts...)
	}

	return ErrInvalidInstance
}
