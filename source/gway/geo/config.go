package geo

import "time"

// Config config
type Config struct {
	License       string
	UpdateTimeout time.Duration
	UpdatePeriod  time.Duration
	AutoUpdate    bool
	DbStorePath   string
}
