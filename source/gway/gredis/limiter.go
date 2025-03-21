package gredis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.bydev.io/frameworks/byone/core/stores/redis"
)

const (
	redisPrefix            = "BGW" + ":rate_limit:"
	redisPrefixWithCounter = "BGW" + ":rate_limit:counter:"

	defaultRateType = "default"
	counterRateType = "counter"
)

type Limit struct {
	Rate   int `json:"rate"`
	Burst  int `json:"burst"`
	Period time.Duration
}

func (l Limit) String() string {
	return fmt.Sprintf("%d req/%s (burst %d)", l.Rate, fmtDur(l.Period), l.Burst)
}

func (l Limit) IsZero() bool {
	return l == Limit{}
}

func fmtDur(d time.Duration) string {
	switch d {
	case time.Second:
		return "s"
	case time.Minute:
		return "m"
	case time.Hour:
		return "h"
	}
	return d.String()
}

func PerSecond(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Second,
		Burst:  rate,
	}
}

func PerMinute(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Minute,
		Burst:  rate,
	}
}

func PerHour(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Hour,
		Burst:  rate,
	}
}

// ------------------------------------------------------------------------------

// Limiter controls how frequently events are allowed to happen.
type Limiter struct {
	rdb *redis.Redis
}

// NewLimiter returns a new Limiter.
func NewLimiter(rdb *redis.Redis) *Limiter {
	return &Limiter{
		rdb: rdb,
	}
}

// Allow is a shortcut for AllowN(ctx, key, limit, 1).
func (l Limiter) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	return l.AllowN(ctx, key, limit, 1)
}

// AllowN reports whether n events may happen at time now.
func (l Limiter) AllowN(ctx context.Context, key string, limit Limit, n int) (*Result, error) {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	values := []interface{}{limit.Burst, limit.Rate, limit.Period.Seconds(), n}

	v, err := EvalSha(ctx, l.rdb, allowN, []string{redisPrefix + key}, values...)
	if err != nil {
		return nil, err
	}

	values, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("redis AllowN: unexpected result type %T", v)
	}

	retryAfter, err := strconv.ParseFloat(values[2].(string), 64)
	if err != nil {
		return nil, err
	}

	resetAfter, err := strconv.ParseFloat(values[3].(string), 64)
	if err != nil {
		return nil, err
	}

	res := &Result{
		Limit:      limit,
		Allowed:    int(values[0].(int64)),
		Remaining:  int(values[1].(int64)),
		RetryAfter: dur(retryAfter),
		ResetAfter: dur(resetAfter),
	}
	return res, nil
}

// AllowM reports whether n events may happen at time now.
// this is a different implementation used for bgwg.
func (l Limiter) AllowM(ctx context.Context, key string, limit Limit, n int) (*Result, error) {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	values := []interface{}{n, int32(limit.Period.Seconds()), limit.Rate}

	key = redisPrefixWithCounter + key

	v, err := EvalSha(ctx, l.rdb, incrbyEX, []string{key}, values...)
	if err != nil {
		return nil, err
	}

	vals, ok := v.([]interface{})
	if !ok || len(vals) != 2 {
		return nil, fmt.Errorf("redis AllowM: unexpected result type %T, len: %d", v, len(vals))
	}

	res := &Result{
		Limit:      limit,
		Allowed:    n,
		Remaining:  limit.Rate - int(vals[0].(int64)), // 1->1->9, 10->10->0, 11->11->-1
		ResetAfter: time.Duration(vals[1].(int64)) * time.Millisecond,
	}
	if res.Remaining < 0 {
		res.Remaining = 0
		res.Allowed = 0
	}
	return res, nil
}

// Reset gets a key and reset all limitations and previous usages
func (l *Limiter) Reset(ctx context.Context, key string, rateType string) error {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	switch rateType {
	case defaultRateType:
		key = redisPrefix + key
	case counterRateType:
		key = redisPrefixWithCounter + key
	default:
		return fmt.Errorf("unknown rate type: %s", rateType)
	}

	_, err := l.rdb.DelCtx(ctx, key)
	return err
}

func dur(f float64) time.Duration {
	if f == -1 {
		return -1
	}
	return time.Duration(f * float64(time.Second))
}

type Result struct {
	// Limit is the limit that was used to obtain this result.
	Limit Limit

	// Allowed is the number of events that may happen at time now.
	Allowed int

	// Remaining is the maximum number of requests that could be
	// permitted instantaneously for this key given the current
	// state. For example, if a rate limiter allows 10 requests per
	// second and has already received 6 requests for this key this
	// second, Remaining would be 4.
	Remaining int

	// RetryAfter is the time until the next request will be permitted.
	// It should be -1 unless the rate limit has been exceeded.
	RetryAfter time.Duration

	// ResetAfter is the time until the RateLimiter returns to its
	// initial state for a given key. For example, if a rate limiter
	// manages requests per second and received one request 200ms ago,
	// Reset would return 800ms. You can also think of this as the time
	// until Limit and Remaining will be equal.
	ResetAfter time.Duration
}

func (r Result) String() string {
	return fmt.Sprintf("%s %d %d %s %s", r.Limit.String(), r.Allowed, r.Remaining, r.RetryAfter.String(), r.ResetAfter.String())
}
