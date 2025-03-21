package glog

import (
	"context"
	"testing"
	"time"
)

func TestConsole(t *testing.T) {
	c := &Config{
		Type:  TypeConsole,
		Level: DebugLevel,
	}
	SetLogger(New(c))
	Info(nil, "TestConsole", String("key", "value"))
	time.Sleep(time.Second)
}

func TestDisableLevel(t *testing.T) {
	c := &Config{
		Type:          TypeConsole,
		Level:         DebugLevel,
		DisableCaller: true,
		DisableLevel:  true,
	}
	SetLogger(New(c))
	Info(nil, "TestConsole", String("key", "value"))
	time.Sleep(time.Second)
}

func TestFile(t *testing.T) {
	c := &Config{
		Type:  TypeFile,
		File:  "./test.log",
		Level: DebugLevel,
	}
	SetLogger(New(c))
	Info(nil, "TestFile", String("key", "value"))
	Close()
	// time.Sleep(time.Second * 3)
}

func TestAsync(t *testing.T) {
	c := &Config{
		Type:  TypeConsole,
		Level: DebugLevel,
		Async: true,
	}
	SetLogger(New(c))
	Info(nil, "TestAsync", String("key", "value"))
	time.Sleep(time.Second)
	Close()
}

func TestContext(t *testing.T) {
	c := &Config{
		Type:  TypeConsole,
		Level: DebugLevel,
		ContextFn: func(ctx context.Context, fields []Field) []Field {
			traceId, ok := ctx.Value("traceid").(string)
			if ok {
				fields = append(fields, String("traceid", traceId))
			}
			return fields
		},
	}
	SetLogger(New(c))
	ctx := context.WithValue(context.Background(), "traceid", "123")
	Info(ctx, "TestContext", String("key", "value"))
	time.Sleep(time.Second * 3)
}

func TestLevel(t *testing.T) {
	c := &Config{
		Type:  TypeConsole,
		Level: ErrorLevel,
		Async: false,
	}
	l := New(c)
	// 预期打印两条日志
	l.Info(nil, "TestLevel", String("key", "info"))
	l.Error(nil, "TestLevel", String("key", "error"))
	l.SetLevel(DebugLevel)
	l.Info(nil, "TestLevel", String("key", "debug"))
	time.Sleep(time.Second)
}

func TestErrorNoStack(t *testing.T) {
	// 测试error不打印stack
	Error(nil, "TestError", String("key", "xxx"))
}

func TestStack(t *testing.T) {
	// 测试主动打印stack
	Error(nil, "TestError with stack", Stack("stack"))
}

func TestDefault(t *testing.T) {
	Info(nil, "TestDefault", String("key", "xxx"))
}

func TestDefaultFilename(t *testing.T) {
	t.Log(getDefaultFilename())
}

func TestParseLevel(t *testing.T) {
	t.Log(ParseLevel("debug"))
	t.Log(ParseLevel("123"))
}
