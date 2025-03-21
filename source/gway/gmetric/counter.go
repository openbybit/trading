package gmetric

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// A CounterVecOpts is an alias of VectorOpts.
	CounterVecOpts struct {
		Namespace   string
		Subsystem   string
		Name        string
		Help        string
		Labels      []string
		ConstLabels Labels
	}

	// CounterVec interface represents a counter vector.
	CounterVec interface {
		// Inc increments labels.
		Inc(labels ...string)
		// Add adds labels with v.
		Add(v float64, labels ...string)
		close() bool
	}

	promCounterVec struct {
		counter *prom.CounterVec
	}
)

// NewCounterVec returns a CounterVec.
func NewCounterVec(cfg *CounterVecOpts) CounterVec {
	if cfg == nil {
		return nil
	}

	vec := prom.NewCounterVec(prom.CounterOpts{
		Namespace:   cfg.Namespace,
		Subsystem:   cfg.Subsystem,
		Name:        cfg.Name,
		Help:        cfg.Help,
		ConstLabels: prom.Labels(cfg.ConstLabels),
	}, cfg.Labels)
	prom.MustRegister(vec)
	cv := &promCounterVec{
		counter: vec,
	}

	return cv
}

func (cv *promCounterVec) Inc(labels ...string) {
	cv.counter.WithLabelValues(labels...).Inc()
}

func (cv *promCounterVec) Add(v float64, labels ...string) {
	cv.counter.WithLabelValues(labels...).Add(v)
}

func (cv *promCounterVec) close() bool {
	return prom.Unregister(cv.counter)
}
