package glog

import "context"

func NewStdLogger(printLevel Level, log Logger) StdLogger {
	if log == nil {
		log = global
	}

	return &stdLogger{level: printLevel, log: log}
}

// StdLogger std log print接口,通常用于其他模块的调试日志
type StdLogger interface {
	Printf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type stdLogger struct {
	level Level
	log   Logger
}

func (l *stdLogger) Printf(format string, args ...interface{}) {
	l.log.doLogf(context.Background(), l.level, format, args...)
}

func (l *stdLogger) Fatalf(format string, args ...interface{}) {
	l.log.doLogf(context.Background(), FatalLevel, format, args...)
}
