package jaeger

import (
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerCfg "github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

// Jaeger
type Config struct {
	Host        string
	ServiceName string
	LogSpans    bool
}

// Init Jaeger
func InitJaeger(cfg Config) (opentracing.Tracer, io.Closer, error) {
	jaegerCfgInstance := jaegerCfg.Configuration{
		ServiceName: cfg.ServiceName,
		Sampler: &jaegerCfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegerCfg.ReporterConfig{
			LogSpans:           cfg.LogSpans,
			LocalAgentHostPort: cfg.Host,
		},
	}

	return jaegerCfgInstance.NewTracer(
		jaegerCfg.Logger(jaegerLog.StdLogger),
		jaegerCfg.Metrics(metrics.NullFactory),
	)
}
