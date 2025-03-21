package timing

import "time"

// ParseDateTime (2018-02-03 00:00:00, 2006-01-02 15:04:05)
func ParseDateTime(t string, layout ...string) time.Time {
	def := "2006-01-02 15:04:05"
	if len(layout) > 0 {
		def = layout[0]
	}
	p, _ := time.ParseInLocation(def, t, time.Local)
	return p
}
