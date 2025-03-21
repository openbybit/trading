package gmetric

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// GaugeVecOpts is an alias of VectorOpts.
	GaugeVecOpts struct {
		Namespace   string
		Subsystem   string
		Name        string
		Help        string
		Labels      []string
		ConstLabels Labels
	}

	// GaugeVec represents a gauge vector.
	GaugeVec interface {
		// Set sets v to labels.
		Set(v float64, labels ...string)
		// Inc increments labels.
		Inc(labels ...string)
		// Dec increments labels.
		Dec(labels ...string)
		// Add adds v to labels.
		Add(v float64, labels ...string)
		// Sub adds v to labels.
		Sub(v float64, labels ...string)
		close() bool
	}

	promGaugeVec struct {
		gauge *prom.GaugeVec
	}
)

// NewGaugeVec returns a GaugeVec.
func NewGaugeVec(cfg *GaugeVecOpts) GaugeVec {
	if cfg == nil {
		return nil
	}

	vec := prom.NewGaugeVec(
		prom.GaugeOpts{
			Namespace:   cfg.Namespace,
			Subsystem:   cfg.Subsystem,
			Name:        cfg.Name,
			Help:        cfg.Help,
			ConstLabels: prom.Labels(cfg.ConstLabels),
		}, cfg.Labels)
	prom.MustRegister(vec)
	gv := &promGaugeVec{
		gauge: vec,
	}

	return gv
}

func (gv *promGaugeVec) Inc(labels ...string) {
	gv.gauge.WithLabelValues(labels...).Inc()
}

func (gv *promGaugeVec) Dec(labels ...string) {
	gv.gauge.WithLabelValues(labels...).Dec()
}

func (gv *promGaugeVec) Add(v float64, labels ...string) {
	gv.gauge.WithLabelValues(labels...).Add(v)
}

func (gv *promGaugeVec) Sub(v float64, labels ...string) {
	gv.gauge.WithLabelValues(labels...).Sub(v)
}

func (gv *promGaugeVec) Set(v float64, labels ...string) {
	gv.gauge.WithLabelValues(labels...).Set(v)
}

func (gv *promGaugeVec) close() bool {
	return prom.Unregister(gv.gauge)
}
