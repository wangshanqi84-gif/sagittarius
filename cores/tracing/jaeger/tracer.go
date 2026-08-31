package jaeger

import (
	"context"
	"net/url"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Option func(o *options)

type options struct {
	addr  string
	ratio float64
	ebs   int
	qs    int
	bto   time.Duration
	eto   time.Duration
}

func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

func WithRatio(ratio float64) Option {
	return func(o *options) {
		o.ratio = ratio
	}
}

func WithExportBatchSize(ebs int) Option {
	return func(o *options) {
		o.ebs = ebs
	}
}

func WithMaxQueueSize(qs int) Option {
	return func(o *options) {
		o.qs = qs
	}
}

func WithBatchTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.bto = timeout
	}
}

func WithExportTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.eto = timeout
	}
}

type Tracer struct {
	ctx      context.Context
	tracer   trace.Tracer
	provider *sdk.TracerProvider
}

func NewTracer(ctx context.Context, serviceName string, opts ...Option) tracing.Tracer {
	option := options{
		ratio: 0.5,
		ebs:   512,
		qs:    2048,
		bto:   5 * time.Second,
		eto:   30 * time.Second,
	}
	for _, o := range opts {
		o(&option)
	}
	// 参数检查
	if option.ratio < 0 || option.ratio > 1 {
		option.ratio = 0.5
	}
	if option.ebs <= 0 {
		option.ebs = 512
	}
	// 公共的 Resource 配置
	resourceOpt := sdk.WithResource(
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.DeploymentName(env.GetRunEnv()),
		),
	)
	var provider *sdk.TracerProvider
	var tracer trace.Tracer
	if option.addr == "" {
		opt := []sdk.TracerProviderOption{
			resourceOpt,
			sdk.WithSampler(sdk.NeverSample()),
		}
		provider = sdk.NewTracerProvider(opt...)
		tracer = provider.Tracer(serviceName)
	} else {
		// 解析addr
		u, err := url.Parse(option.addr)
		if err != nil {
			panic(err)
		}
		var exporter *otlptrace.Exporter
		schema := u.Scheme
		if schema == "" {
			schema = "grpc"
		}
		switch schema {
		case "grpc":
			exporter, err = otlptracegrpc.New(
				ctx,
				otlptracegrpc.WithEndpoint(u.Host),
				otlptracegrpc.WithInsecure(),
			)
		case "http":
			exporter, err = otlptracehttp.New(
				ctx,
				otlptracehttp.WithEndpoint(u.Host),
				otlptracehttp.WithInsecure(),
			)
		case "https":
			exporter, err = otlptracehttp.New(
				ctx,
				otlptracehttp.WithEndpoint(u.Host),
			)
		default:
			panic(errors.New("unknown otlptrace.Exporter schema: " + schema))
		}
		if err != nil {
			panic(err)
		}
		provider = sdk.NewTracerProvider(
			sdk.WithBatcher(exporter,
				sdk.WithMaxExportBatchSize(option.ebs),
				sdk.WithBatchTimeout(option.bto),
				sdk.WithMaxQueueSize(option.qs),
				sdk.WithExportTimeout(option.eto),
			),
			resourceOpt,
			sdk.WithSampler(
				sdk.ParentBased(sdk.TraceIDRatioBased(option.ratio)),
			),
		)
		tracer = provider.Tracer(serviceName)
	}
	otel.SetTracerProvider(provider)
	return &Tracer{
		ctx:      ctx,
		tracer:   tracer,
		provider: provider,
	}
}

func (tracer *Tracer) Start(ctx context.Context,
	spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.tracer.Start(ctx, spanName, opts...)
}

func (tracer *Tracer) Close() error {
	if tracer.provider != nil {
		return tracer.provider.Shutdown(tracer.ctx)
	}
	return nil
}
