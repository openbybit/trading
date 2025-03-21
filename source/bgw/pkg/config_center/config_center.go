package config_center

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
)

//go:generate mockgen -source=config_center.go -destination=./config_center_mock.go -package=mock
type Configure interface {
	Listen(ctx context.Context, key string, listener observer.EventListener) error
	Get(ctx context.Context, key string) (string, error)
	GetChildren(ctx context.Context, key string) ([]string, []string, error)
	Put(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
}
