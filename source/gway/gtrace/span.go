package gtrace

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.opentelemetry.io/otel/trace"
)

const (
	UberTraceIdKey = "uber-trace-id"
)

// type TraceID = jaeger.TraceID
// type SpanID = jaeger.SpanID

// 低版本fasthttp不支持标准的context接口,需要从value中提取context
type ExtractCtxFunc func(ctx context.Context) context.Context
type InjectCtxFunc func(oldCtx, newCtx context.Context) context.Context
type BuildTagsFunc func(ctx context.Context) opentracing.Tags

var (
	extractCtxFn = func(ctx context.Context) context.Context { return ctx }
	injectCtxFn  = func(oldCtx, newCtx context.Context) context.Context { return newCtx }
	buildTagsFn  BuildTagsFunc
)

// SetContextFunc 设置context解析方式
func SetContextFunc(ex ExtractCtxFunc, inject InjectCtxFunc) {
	extractCtxFn = ex
	injectCtxFn = inject
}

// SetBuildTagsFunc 设置全局BuildTagsFunc
func SetBuildTagsFunc(fn BuildTagsFunc) {
	buildTagsFn = fn
}

// Begin 封装StartSpanFromContext
func Begin(ctx context.Context, op string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	if buildTagsFn != nil {
		tags := buildTagsFn(ctx)
		if len(tags) > 0 {
			opts = append(opts, tags)
		}
	}

	realCtx := extractCtxFn(ctx)
	span, newCtx := opentracing.StartSpanFromContext(realCtx, op, opts...)
	outCtx := injectCtxFn(ctx, newCtx)
	return span, outCtx
}

// Finish .
func Finish(span opentracing.Span) {
	if span != nil {
		span.Finish()
	}
}

// WithTextMapCarrier carrier必须实现TextMapReader接口
func WithTextMapCarrier(carrier opentracing.TextMapReader) opentracing.StartSpanOption {
	spanCtx, _ := opentracing.GlobalTracer().Extract(opentracing.TextMap, carrier)
	return opentracing.ChildOf(spanCtx)
}

// WithUberTraceID 解析UberTraceID
func WithUberTraceID(uberTraceID string) opentracing.StartSpanOption {
	carrier := opentracing.TextMapCarrier(map[string]string{UberTraceIdKey: uberTraceID})
	spanCtx, _ := opentracing.GlobalTracer().Extract(opentracing.TextMap, carrier)
	return opentracing.ChildOf(spanCtx)
}

// WithStartTime 含有起始时间
func WithStartTime(t time.Time) opentracing.StartSpanOption {
	return opentracing.StartTime(t)
}

// SpanFromContext 从context解析span
func SpanFromContext(ctx context.Context) opentracing.Span {
	realCtx := extractCtxFn(ctx)
	return opentracing.SpanFromContext(realCtx)
}

// TraceparentFromSpan 从span中提取W3C规定是traceparent
// traceparent format: {version}-{trace_id}-{parent_id}-{trace_flags}
// https://www.w3.org/TR/trace-context/
func TraceparentFromSpan(span opentracing.Span) string {
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			return x.StringTraceparent()
		}
	}

	return ""
}

// UberTraceIDFromSpan 从span中提取uber-trace-id
// uber-trace-id format:  {traceid}:{spanid}:{parentid}:{flags}
func UberTraceIDFromSpan(span opentracing.Span) string {
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			return x.String()
		}
	}

	return ""
}

// TraceIDFromSpan 从span中提取trace-id
func TraceIDFromSpan(span opentracing.Span) string {
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			return x.TraceID().String()
		}
	}

	return ""
}

// SpanIDFromSpan 从span中提取span-id
func SpanIDFromSpan(span opentracing.Span) string {
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			return x.SpanID().String()
		}
	}

	return ""
}

// ParentIDFromSpan 从span中其他parent-id
func ParentIDFromSpan(span opentracing.Span) string {
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			return x.ParentID().String()
		}
	}

	return ""
}

// OtelCtxFromOtraCtx is a bridge api for change opentracing ctx to opentelemetry ctx
func OtelCtxFromOtraCtx(ctx context.Context) context.Context {
	span := SpanFromContext(ctx)
	if span != nil {
		if x, ok := span.Context().(jaeger.SpanContext); ok {
			tid := x.TraceID()
			buf := [16]byte{}
			binary.BigEndian.PutUint64(buf[:8], tid.High)
			binary.BigEndian.PutUint64(buf[8:], tid.Low)
			otelSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: buf})
			return trace.ContextWithSpanContext(context.Background(), otelSpanCtx)
		}
	}

	return ctx
}
