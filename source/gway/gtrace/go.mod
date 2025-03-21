module code.bydev.io/fbu/gateway/gway.git/gtrace

go 1.18

require (
	github.com/opentracing/opentracing-go v1.2.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	go.opentelemetry.io/otel/trace v1.14.0
)

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
)

replace github.com/uber/jaeger-client-go => code.bydev.io/public-lib/infra/trace/jaeger-client-go.git v1.0.0
