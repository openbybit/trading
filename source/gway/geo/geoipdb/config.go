package geoipdb

import "time"

type config struct {
	storePath        string
	license          string
	timeout          time.Duration
	autoUpdateEnable bool
	autoUpdatePeriod time.Duration
}

type Option interface {
	apply(*config)
}

type timeoutOption struct {
	timeout time.Duration
}

func (t timeoutOption) apply(c *config) {
	c.timeout = t.timeout
}

// WithTimeout with timeout
func WithTimeout(timeout time.Duration) Option {
	return timeoutOption{timeout: timeout}
}

type autoUpdateOption struct {
	autoUpdateEnable bool
	autoUpdatePeriod time.Duration
}

func (a autoUpdateOption) apply(c *config) {
	c.autoUpdateEnable = a.autoUpdateEnable
	c.autoUpdatePeriod = a.autoUpdatePeriod
}

// WithAutoUpdate with auto update
func WithAutoUpdate(autoUpdateEnable bool, autoUpdatePeriod time.Duration) Option {
	return autoUpdateOption{
		autoUpdateEnable: autoUpdateEnable,
		autoUpdatePeriod: autoUpdatePeriod,
	}
}
