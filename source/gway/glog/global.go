package glog

import "context"

var global Logger

func init() {
	global = New(&Config{
		Type:  TypeConsole,
		Level: DebugLevel,
	})
}

// SetLogger sets the global logger. not thread safe, must set on startup.
func SetLogger(l Logger) {
	if l != nil {
		global = l
	}
}

// Default return default logger
func Default() Logger {
	return global
}

func Sync() error {
	return global.Sync()
}

func Close() error {
	if global != nil {
		return global.Close()
	}

	return nil
}

// Enabled test if the logger is enabled
func Enabled(lv Level) bool {
	return global.Enabled(lv)
}

// Named new logger with name
func Named(s string) Logger {
	return global.Named(s)
}

// With new logger with fields
func With(fields ...Field) Logger {
	return global.With(fields...)
}

// GetLevel get global logger level
func GetLevel() Level {
	return global.GetLevel()
}

// SetLevel set global logger level
func SetLevel(lv Level) {
	global.SetLevel(lv)
}

// Log logs a message at the given level. The message includes any fields passed
func Log(ctx context.Context, level Level, msg string, fields ...Field) {
	global.doLog(ctx, level, msg, fields...)
}

// Logf logs a message at the given level. The message includes any fields passed
func Logf(ctx context.Context, level Level, format string, args ...interface{}) {
	global.doLogf(ctx, level, format, args...)
}

// Debug logs a message at debug level. The message includes any fields passed
func Debug(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, DebugLevel, msg, fields...)
}

// Info logs a message at info level. The message includes any fields passed
func Info(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, InfoLevel, msg, fields...)
}

// Warn logs a message at warn level. The message includes any fields passed
func Warn(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, WarnLevel, msg, fields...)
}

// Error logs a message at error level. The message includes any fields passed
func Error(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, ErrorLevel, msg, fields...)
}

// Panic logs a message at panic level. The message includes any fields passed
func Panic(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, PanicLevel, msg, fields...)
}

// Fatal logs a message at fatal level. The message includes any fields passed
func Fatal(ctx context.Context, msg string, fields ...Field) {
	global.doLog(ctx, FatalLevel, msg, fields...)
}

// Debugf logs a message at debug level. The message includes any fields passed
func Debugf(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, DebugLevel, format, args...)
}

// Infof logs a message at info level. The message includes any fields passed
func Infof(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, InfoLevel, format, args...)
}

// Warnf logs a message at warn level. The message includes any fields passed
func Warnf(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, WarnLevel, format, args...)
}

// Errorf logs a message at error level. The message includes any fields passed
func Errorf(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, ErrorLevel, format, args...)
}

// Panicf logs a message at panic level. The message includes any fields passed
func Panicf(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, PanicLevel, format, args...)
}

// Fatalf logs a message at fatal level. The message includes any fields passed
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	global.doLogf(ctx, FatalLevel, format, args...)
}
