package gmetric

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// A HistogramVecOpts is a histogram vector options.
	HistogramVecOpts struct {
		Namespace   string
		Subsystem   string
		Name        string
		Help        string
		Labels      []string
		ConstLabels Labels
		Buckets     []float64
	}

	// A HistogramVec interface represents a histogram vector.
	HistogramVec interface {
		// Observe adds observation v to labels.
		Observe(v float64, labels ...string)
		close() bool
	}

	promHistogramVec struct {
		histogram *prom.HistogramVec
	}
)

// NewHistogramVec returns a HistogramVec.
func NewHistogramVec(cfg *HistogramVecOpts) HistogramVec {
	if cfg == nil {
		return nil
	}

	vec := prom.NewHistogramVec(prom.HistogramOpts{
		Namespace:   cfg.Namespace,
		Subsystem:   cfg.Subsystem,
		Name:        cfg.Name,
		Help:        cfg.Help,
		ConstLabels: prom.Labels(cfg.ConstLabels),
		Buckets:     cfg.Buckets,
	}, cfg.Labels)
	prom.MustRegister(vec)
	hv := &promHistogramVec{
		histogram: vec,
	}

	return hv
}

func (hv *promHistogramVec) Observe(v float64, labels ...string) {
	hv.histogram.WithLabelValues(labels...).Observe(v)
}

func (hv *promHistogramVec) close() bool {
	return prom.Unregister(hv.histogram)
}
