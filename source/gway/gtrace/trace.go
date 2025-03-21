package gtrace

import (
	"io"
	"os"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	SamplerTypeRemote = jaeger.SamplerTypeRemote
	SamplerTypeConst  = jaeger.SamplerTypeConst
)

const (
	defaultSamplingServerURL = "http://trace-service.prod.efficiency.ww5sawfyut0k.bitsvc.io/api/v1/trace/sampling"
)

const (
	envKeyGwayServiceName = "GWAY_SERVICE_NAME"
	envKeyMyProjectName   = "MY_PROJECT_NAME"
)

var globalCloser io.Closer

type Config struct {
	Address           string
	ServiceName       string
	SamplerType       string
	SamplingServerURL string
}

// https://uponly.larksuite.com/wiki/wikus9zf9uFWwBQVPXXxFg4COig
// config.JaegerInit()
func Init(conf *Config) error {
	if conf.ServiceName == "" {
		conf.ServiceName = os.Getenv(envKeyGwayServiceName)
		if conf.ServiceName == "" {
			conf.ServiceName = os.Getenv(envKeyMyProjectName)
		}
	}

	if conf.SamplerType == "" {
		conf.SamplerType = SamplerTypeConst
	}

	if conf.SamplerType == SamplerTypeRemote && conf.SamplingServerURL == "" {
		conf.SamplingServerURL = defaultSamplingServerURL
	}

	sampler := &config.SamplerConfig{
		Type:                    conf.SamplerType,
		SamplingServerURL:       conf.SamplingServerURL,
		SamplingRefreshInterval: 10 * time.Second,
		Param:                   1,
	}
	cfg := config.Configuration{
		Gen128Bit:   true,
		ServiceName: conf.ServiceName,
		Sampler:     sampler,
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: time.Second,
		},
		Tags: []opentracing.Tag{
			{Key: "env", Value: os.Getenv("MY_ENV_NAME")},
			{Key: "projectEnv", Value: os.Getenv("MY_PROJECT_ENV_NAME")},
			{Key: "project", Value: os.Getenv("MY_PROJECT_NAME")},
		},
	}

	transport, err := jaeger.NewUDPTransport(conf.Address, 0)
	if err != nil {
		return err
	}

	reporter := jaeger.NewRemoteReporter(transport)
	tracer, closer, err := cfg.NewTracer(config.Reporter(reporter))
	if err != nil {
		return err
	}

	globalCloser = closer
	opentracing.SetGlobalTracer(tracer)
	return nil
}

func Close() {
	if globalCloser != nil {
		globalCloser.Close()
		globalCloser = nil
	}
}
