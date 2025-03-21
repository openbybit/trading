package gapp

var global App

func Run(opts ...Option) error {
	if global == nil {
		global = New()
	}
	return global.Run(opts...)
}

func Exit() {
	if global != nil {
		global = nil
	}
}

func logf(format string, args ...interface{}) {
	if global != nil {
		global.getLogger().Printf(format, args...)
	}
}
