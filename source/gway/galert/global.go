package galert

import (
	"context"
	"sync"
)

var (
	global     Alerter
	globalOnce sync.Once
)

func SetDefault(alerter Alerter) {
	global = alerter
}

func getGlobal() Alerter {
	globalOnce.Do(func() {
		if global != nil {
			return
		}
		global = New(nil)
	})

	return global
}

func Alert(ctx context.Context, level Level, message string, opts ...Option) {
	getGlobal().Alert(ctx, level, message, opts...)
}

func Info(ctx context.Context, message string, opts ...Option) {
	getGlobal().Alert(ctx, LevelInfo, message, opts...)
}

func Warn(ctx context.Context, message string, opts ...Option) {
	getGlobal().Alert(ctx, LevelWarn, message, opts...)
}

func Error(ctx context.Context, message string, opts ...Option) {
	getGlobal().Alert(ctx, LevelError, message, opts...)
}
