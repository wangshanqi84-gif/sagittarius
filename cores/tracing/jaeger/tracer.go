package jaeger

import (
	"io"

	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

type Option func(o *options)

type options struct {
	addr string
}

func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

type Tracer struct {
	tracer opentracing.Tracer
	closer io.Closer
}

func NewTracer(serviceName string, opts ...Option) tracing.Tracer {
	option := options{}
	for _, o := range opts {
		o(&option)
	}
	cfg := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 0,
		},
		Gen128Bit: true,
	}
	if option.addr != "" {
		cfg.Sampler.Param = 1
		cfg.Reporter = &jaegercfg.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: option.addr,
		}
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic(err)
	}
	return &Tracer{
		tracer: tracer,
		closer: closer,
	}
}

func (tracer *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return tracer.tracer.StartSpan(operationName, opts...)
}

func (tracer *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return tracer.tracer.Inject(sm, format, carrier)
}

func (tracer *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return tracer.tracer.Extract(format, carrier)
}

func (tracer *Tracer) Close() error {
	return tracer.closer.Close()
}
