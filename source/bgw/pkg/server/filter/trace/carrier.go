package trace

import (
	"bgw/pkg/common/types"
)

type carrier struct {
	h *types.Header
}

func newCarrier(h *types.Header) carrier {
	return carrier{h: h}
}

// Set conforms to the TextMapWriter interface.
func (c carrier) Set(key, val string) {
	c.h.Set(key, val)
}

// ForeachKey conforms to the TextMapReader interface.
func (c carrier) ForeachKey(handler func(key, val string) error) (err error) {
	c.h.VisitAll(func(key, value []byte) {
		if err != nil {
			return
		}

		if err = handler(string(key), string(value)); err != nil {
			return
		}
	})

	return err
}
