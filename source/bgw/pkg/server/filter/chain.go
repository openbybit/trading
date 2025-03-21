package filter

import (
	"context"

	"bgw/pkg/common/types"
)

// Chain decorator pattern of filter
// NOTE: not thread safe
type Chain struct {
	filters []Filter
	index   map[string]int // filter name -> filter index
}

// NewChain creates a new chain,
func NewChain(filters ...Filter) *Chain {
	return &Chain{
		index:   make(map[string]int),
		filters: append(([]Filter)(nil), filters...),
	}
}

// Finally the final target (the most inside handler) handler
func (c *Chain) Finally(h types.Handler) types.Handler {
	for i := range c.filters {
		h = c.filters[len(c.filters)-1-i].Do(h)
	}

	return h
}

// Append extends a chain, adding the specified constructors
func (c *Chain) Append(filters ...Filter) *Chain {
	for _, filter := range filters {
		if filter == nil {
			continue
		}

		name := filter.GetName()
		i, ok := c.index[name]
		// named filter, check unique
		if ok {
			c.filters[i] = filter // already exists, replace with new filter
		} else {
			c.filters = append(c.filters, filter) // append new filter
			c.index[name] = len(c.filters) - 1
		}
	}

	return c
}

// Extend merge chain
func (c *Chain) Extend(chain Chain) *Chain {
	return c.Append(chain.filters...)
}

// AppendNames append filter by registered names
func (c *Chain) AppendNames(names ...string) (*Chain, error) {
	for _, name := range names {
		filter, err := GetFilter(context.Background(), name)
		if err != nil {
			return nil, err
		}
		c.Append(filter)
	}

	return c, nil
}

// GlobalChain get default filter chain
func GlobalChain() *Chain {
	must := []string{
		MetricsFilterKey,
		TracingFilterKey,
		ContextFilterKeyGlobal,
		AccessLogFilterKey,
		CorsFilterKey,
		QPSRateLimitFilterKeyGlobal,
	}

	fc := NewChain()
	fc, _ = fc.AppendNames(must...)

	return fc
}
