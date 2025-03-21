package glog

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newZapLogger(c *Config) Logger {
	if c == nil {
		c = &Config{}
	}

	if c.Skip <= 0 {
		c.Skip = 2
	}
	if c.TimeFormat == "" {
		c.TimeFormat = "2006-01-02T15:04:05.000Z"
	}
	callerKey := "C"
	levelKey := "L"
	if c.DisableCaller {
		callerKey = zapcore.OmitKey
	}
	if c.DisableLevel {
		levelKey = zapcore.OmitKey
	}
	encConfig := zapcore.EncoderConfig{
		TimeKey:          "T",
		LevelKey:         levelKey,
		NameKey:          "N",
		CallerKey:        callerKey,
		MessageKey:       "M",
		StacktraceKey:    "S",
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout(c.TimeFormat),
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		EncodeName:       zapcore.FullNameEncoder,
		ConsoleSeparator: c.ConsoleSeparator,
	}

	var enc zapcore.Encoder
	var ws zapcore.WriteSyncer
	switch c.Type {
	case TypeFile, TypeLumberjack:
		if c.MaxSize <= 0 {
			c.MaxSize = 100
		}
		if c.MaxAge <= 0 {
			c.MaxAge = 30
		}
		if c.MaxBackups <= 0 {
			c.MaxBackups = 3
		}
		if c.File == "" {
			c.File = getDefaultFilename()
		}

		if err := mkdirAll(filepath.Dir(c.File)); err != nil {
			log.Printf("make dir for logfile fail: %v, err: %v\n", c.File, err)
		}

		fileLogger := &lumberjack.Logger{
			Filename:   c.File,
			MaxSize:    c.MaxSize,
			MaxBackups: c.MaxBackups,
			MaxAge:     c.MaxAge,
			Compress:   c.Compress,
		}
		ws = zapcore.AddSync(&zapcore.BufferedWriteSyncer{
			WS:            zapcore.AddSync(fileLogger),
			FlushInterval: time.Second,
		})
	default:
		ws = zapcore.AddSync(os.Stdout)
		encConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if c.Format == FormatJson {
		enc = zapcore.NewJSONEncoder(encConfig)
	} else {
		enc = zapcore.NewConsoleEncoder(encConfig)
	}

	atom := zap.NewAtomicLevel()
	atom.SetLevel(c.Level)

	core := zapcore.NewCore(enc, ws, atom)
	if c.Async {
		core = newAsyncWriter(core, uint64(c.AsyncSize), c.AsyncInterval, c.AsyncCloseTime, c.AsyncDiscardFn)
	}

	newOptions := []zap.Option{}
	if !c.DisableCaller {
		newOptions = append(newOptions, zap.AddCaller(), zap.AddCallerSkip(c.Skip))
	}

	zlog := zap.New(core, newOptions...)
	return &zapLogger{raw: zlog, atom: atom, fn: c.ContextFn}
}

type zapLogger struct {
	raw    *zap.Logger
	atom   zap.AtomicLevel
	fields []Field
	fn     ContextBuildFunc
}

func (l *zapLogger) Close() error {
	if c, ok := l.raw.Core().(io.Closer); ok && c != nil {
		_ = c.Close()
	}

	return l.raw.Sync()
}

func (l *zapLogger) Sync() error {
	return l.raw.Sync()
}

func (l *zapLogger) Enabled(lv Level) bool {
	return l.raw.Core().Enabled(lv)
}

func (l *zapLogger) SetLevel(lv Level) {
	l.atom.SetLevel(lv)
}

func (l *zapLogger) GetLevel() Level {
	return l.atom.Level()
}

func (l *zapLogger) Named(s string) Logger {
	named := l.raw.Named(s)
	return &zapLogger{raw: named, fn: l.fn}
}

func (l *zapLogger) With(fields ...Field) Logger {
	o := l.raw.With(fields...)
	return &zapLogger{raw: o, fn: l.fn}
}

func (l *zapLogger) Log(ctx context.Context, level Level, msg string, fields ...Field) {
	l.doLog(ctx, level, msg, fields...)
}

func (l *zapLogger) Logf(ctx context.Context, level Level, format string, args ...interface{}) {
	l.doLogf(ctx, level, format, args...)
}

func (l *zapLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, DebugLevel, msg, fields...)
}

func (l *zapLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, InfoLevel, msg, fields...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, WarnLevel, msg, fields...)
}

func (l *zapLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, ErrorLevel, msg, fields...)
}

func (l *zapLogger) Panic(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, PanicLevel, msg, fields...)
}

func (l *zapLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.doLog(ctx, FatalLevel, msg, fields...)
}

func (l *zapLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, DebugLevel, format, args...)
}

func (l *zapLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, InfoLevel, format, args...)
}

func (l *zapLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, WarnLevel, format, args...)
}

func (l *zapLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, ErrorLevel, format, args...)
}

func (l *zapLogger) Panicf(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, PanicLevel, format, args...)
}

func (l *zapLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	l.doLogf(ctx, FatalLevel, format, args...)
}

func (l *zapLogger) buildFields(ctx context.Context, fields []Field) []Field {
	if l.fields != nil {
		fields = append(fields, l.fields...)
	}

	if ctx == nil || l.fn == nil {
		return fields
	}

	return l.fn(ctx, fields)
}

// 仅内部调用使用
func (l *zapLogger) doLog(ctx context.Context, level Level, msg string, fields ...Field) {
	if ce := l.raw.Check(level, msg); ce != nil {
		ce.Write(l.buildFields(ctx, fields)...)
	}
}

func (l *zapLogger) doLogf(ctx context.Context, level Level, format string, args ...interface{}) {
	if l.Enabled(level) {
		if ce := l.raw.Check(level, fmt.Sprintf(format, args...)); ce != nil {
			ce.Write(l.buildFields(ctx, nil)...)
		}
	}
}
