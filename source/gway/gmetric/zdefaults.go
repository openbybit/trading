package gmetric

import (
	"time"
)

type Labels map[string]string

var (
	// default metrics
	defaultErrorCounter CounterVec
	defaultBasicCounter CounterVec
	defaultGauge        GaugeVec
	defaultLatency      HistogramVec
	// http metrics
	httpErrorCounter CounterVec
	httpLatency      HistogramVec
)

// Init 初始化默认埋点
func Init(namespace string) {
	if namespace == "" {
		panic("empty namespace")
	}

	subsystem := "default"

	defaultErrorCounter = NewCounterVec(&CounterVecOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "error",
		Help:      "default error counter",
		Labels:    []string{"type", "label"},
	})

	defaultBasicCounter = NewCounterVec(&CounterVecOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "counter",
		Help:      "default basic counter",
		Labels:    []string{"type", "label"},
	})

	defaultGauge = NewGaugeVec(&GaugeVecOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "gauge",
		Help:      "default gauge",
		Labels:    []string{"type", "label"},
	})

	defaultLatency = NewHistogramVec(&HistogramVecOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "latency",
		Help:      "default latency, milliseconds",
		Labels:    []string{"type", "label"},
		Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 1.2, 1.4, 1.6, 1.8, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 80, 100, 500, 1000, 3000},
	})

	const httpSubsystem = "http"

	httpErrorCounter = NewCounterVec(&CounterVecOpts{
		Namespace: namespace,
		Subsystem: httpSubsystem,
		Name:      "error",
		Help:      "default error counter",
		Labels:    []string{"type", "error", "path", "method", "service"},
	})

	httpLatency = NewHistogramVec(&HistogramVecOpts{
		Namespace: namespace,
		Subsystem: httpSubsystem,
		Name:      "latency",
		Help:      "default latency, milliseconds",
		Labels:    []string{"type", "path", "method", "service", "callOrigin"},
		Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 1.2, 1.4, 1.6, 1.8, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 80, 100, 500, 1000, 3000},
	})
}

// IncDefaultCounter standard counter
func IncDefaultCounter(typ, label string) {
	defaultBasicCounter.Inc(typ, label)
}

// AddDefaultCounter standard counter
func AddDefaultCounter(value int, typ, label string) {
	defaultBasicCounter.Add(float64(value), typ, label)
}

// IncDefaultError standard error counter
func IncDefaultError(typ string, label string) {
	defaultErrorCounter.Inc(typ, label)
}

// AddDefaultError standard error counter
func AddDefaultError(value int, typ, label string) {
	defaultErrorCounter.Add(float64(value), typ, label)
}

// SetDefaultGauge standard gauge
func SetDefaultGauge(value float64, typ, label string) {
	defaultGauge.Set(value, typ, label)
}

// ObserveDefaultLatency standard latency histogram
func ObserveDefaultLatency(d time.Duration, typ string, label string) {
	if d != 0 {
		defaultLatency.Observe(toMilliseconds(d), typ, label)
	}
}

// ObserveDefaultLatencySince standard latency histogram
func ObserveDefaultLatencySince(t time.Time, typ, label string) {
	if !t.IsZero() {
		defaultLatency.Observe(toMilliseconds(time.Since(t)), typ, label)
	}
}

// IncHTTPCounter http counter
func IncHTTPCounter(typ, errorCode, path, method, service string) {
	httpErrorCounter.Inc(typ, errorCode, path, method, service)
}

// ObserveHTTPLatency standard http latency histogram
func ObserveHTTPLatency(d time.Duration, typ, path, method, service, callOrigin string) {
	if d != 0 {
		httpLatency.Observe(toMilliseconds(d), typ, path, method, service, callOrigin)
	}
}

// toMilliseconds returns the duration as a floating point number of milliseconds.
func toMilliseconds(d time.Duration) float64 {
	sec := d / time.Millisecond
	nsec := d % time.Millisecond
	return float64(sec) + float64(nsec)/1e6
}
