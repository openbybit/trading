package gmetric

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

type (
	// A SummaryVecOpts is a summary vector options.
	SummaryVecOpts struct {
		Namespace   string
		Subsystem   string
		Name        string
		Help        string
		Labels      []string
		ConstLabels Labels
		Objectives  map[float64]float64
	}

	// A SummaryVec interface represents a summary vector.
	SummaryVec interface {
		// Observe adds observation v to labels.
		Observe(v float64, labels ...string)
		close() bool
	}

	promSummaryVec struct {
		summary *prom.SummaryVec
	}
)

// NewSummaryVec returns a SummaryVec.
func NewSummaryVec(cfg *SummaryVecOpts) SummaryVec {
	if cfg == nil {
		return nil
	}

	vec := prom.NewSummaryVec(prom.SummaryOpts{
		Namespace:   cfg.Namespace,
		Subsystem:   cfg.Subsystem,
		Name:        cfg.Name,
		Help:        cfg.Help,
		ConstLabels: prom.Labels(cfg.ConstLabels),
		Objectives:  cfg.Objectives,
	}, cfg.Labels)
	prom.MustRegister(vec)
	hv := &promSummaryVec{
		summary: vec,
	}

	return hv
}

func (hv *promSummaryVec) Observe(v float64, labels ...string) {
	hv.summary.WithLabelValues(labels...).Observe(v)
}

func (hv *promSummaryVec) close() bool {
	return prom.Unregister(hv.summary)
}
