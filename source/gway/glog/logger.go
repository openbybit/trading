package glog

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Entry = zapcore.Entry

// Field is a field in a log message.
type Field = zap.Field

// Level returns the current logging level.
type Level = zapcore.Level

const (
	// DebugLevel is the debug level.
	DebugLevel = zap.DebugLevel
	// InfoLevel is the default logging level.
	InfoLevel = zap.InfoLevel
	// WarnLevel is the logging level for messages about possible issues.
	WarnLevel = zap.WarnLevel
	// ErrorLevel is the logging level for messages about errors.
	ErrorLevel = zap.ErrorLevel
	// PanicLevel is the logging level for messages about panics.
	PanicLevel = zap.PanicLevel
	// FatalLevel is the logging level for messages about fatal errors.
	FatalLevel = zap.FatalLevel
)

func ParseLevel(text string) (Level, error) {
	return zapcore.ParseLevel(text)
}

// Logger is a logger interface.
type Logger interface {
	Close() error
	Sync() error
	Enabled(lv Level) bool
	Named(s string) Logger
	With(fields ...Field) Logger

	GetLevel() Level
	SetLevel(lv Level)

	Log(ctx context.Context, level Level, msg string, fields ...Field)
	Logf(ctx context.Context, level Level, format string, args ...interface{})

	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Panic(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)

	Debugf(ctx context.Context, format string, args ...interface{})
	Infof(ctx context.Context, format string, args ...interface{})
	Warnf(ctx context.Context, format string, args ...interface{})
	Errorf(ctx context.Context, format string, args ...interface{})
	Panicf(ctx context.Context, format string, args ...interface{})
	Fatalf(ctx context.Context, format string, args ...interface{})

	// 所有调用都会执行这两个接口,保证skip的调用都是一致的
	doLog(ctx context.Context, level Level, msg string, fields ...Field)
	doLogf(ctx context.Context, level Level, format string, args ...interface{})
}

type DiscardFunc func(entry Entry, fields []Field)

// ContextBuildFunc 从context解析字段
type ContextBuildFunc func(ctx context.Context, fields []Field) []Field

const (
	TypeConsole    = "console"
	TypeFile       = "file"
	TypeLumberjack = "lumberjack"
)

const (
	FormatJson = "json"
)

type Config struct {
	Type             string           `json:"type" yaml:"type"`
	Level            Level            `json:"level" yaml:"level"`
	File             string           `json:"file" yaml:"file"`
	Format           string           `json:"format" yaml:"format"`
	TimeFormat       string           `json:"time_format" yaml:"time_format"`
	MaxSize          int              `json:"max_size" yaml:"max_size"`
	MaxAge           int              `json:"max_age" yaml:"max_age"`
	MaxBackups       int              `json:"max_backups" yaml:"max_backups"`
	Compress         bool             `json:"compress" yaml:"compress"`                   //
	DisableCaller    bool             `json:"disable_caller" yaml:"disable_caller"`       // 是否禁止打印caller
	DisableLevel     bool             `json:"disable_level" yaml:"disable_level"`         // 是否禁止打印level
	Skip             int              `json:"skip" yaml:"skip"`                           // call skip
	ConsoleSeparator string           `json:"console_separator" yaml:"console_separator"` // 分隔符,默认\t
	Async            bool             `json:"async" yaml:"async"`                         // 是否使用同步日志
	AsyncSize        int              `json:"async_size" yaml:"async_size"`               // 异步日志最大数量
	AsyncInterval    time.Duration    `json:"async_interval" yaml:"async_interval"`       // 异步日志最长休眠时间
	AsyncCloseTime   time.Duration    `json:"async_close_time" yaml:"async_close_time"`   // 异步日志最长等待时间
	AsyncDiscardFn   DiscardFunc      `json:"-" yaml:"-"`                                 // 异步日志超过最大数量时,会直接丢弃,业务可以设置此参数感知并埋点
	ContextFn        ContextBuildFunc `json:"-" yaml:"-"`                                 // 从context解析
	Fields           []Field          `json:"-" yaml:"-"`                                 // 默认全局fileds,比如host, project_name等
}

func New(c *Config) Logger {
	return newZapLogger(c)
}
