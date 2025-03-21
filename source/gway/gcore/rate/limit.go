package rate

import (
	"sync"
	"time"
)

// Limiter simple memory rate limit
type Limiter interface {
	Allow() bool
}

func New(rate int64, internal time.Duration) Limiter {
	if internal <= 0 {
		internal = time.Second
	}

	return &limit{
		rate:     rate,
		internal: internal,
	}
}

type limit struct {
	mux          sync.Mutex
	rate         int64
	internal     time.Duration
	token        int64
	lastRestTime time.Time
}

func (l *limit) Allow() bool {
	if l.rate == 0 {
		return true
	}

	l.mux.Lock()
	defer l.mux.Unlock()

	now := time.Now()
	if now.Sub(l.lastRestTime) > l.internal {
		l.token = l.rate
		l.lastRestTime = now
	}

	if l.token <= 0 {
		return false
	} else {
		l.token--
		return true
	}
}
