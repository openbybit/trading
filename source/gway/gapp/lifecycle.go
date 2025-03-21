package gapp

import (
	"context"
	"fmt"
)

type LifecycleEvent int

// String returns name of the event.
func (le LifecycleEvent) String() string {
	switch le {
	case LifecycleStart:
		return "start"
	case LifecycleStop:
		return "stopped"
	case LifecycleShutdown:
		return "shutdown"
	case LifecycleDestroy:
		return "destroy"
	default:
		return fmt.Sprintf("?UNKNOWN LIFECYCLE(%d)?", le)
	}
}

const (
	// LifecycleStart occurs after app initialization and before main function.
	LifecycleStart = LifecycleEvent(iota + 1)
	// LifecycleStop occurs after main function. Application should stop receiving
	// new request and try to interupt running request to shut down gracefully.
	LifecycleStop
	// LifecycleShutdown occurs after stop and shutdown waitgroup, all modules
	// should stop working immediately.
	LifecycleShutdown
	// LifecycleDestroy occurs after shutdown and destroy waitgroup. It's for finalize
	// works such as flush local log file.
	LifecycleDestroy
)

type Lifecycle interface {
	OnLifecycle(ctx context.Context, event LifecycleEvent) error
}

type LifecycleFunc func(ctx context.Context, event LifecycleEvent) error

func (lf LifecycleFunc) OnLifecycle(ctx context.Context, event LifecycleEvent) error {
	return lf(ctx, event)
}
