package cron

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Option func(*options)

type options struct {
	tracer tracing.Tracer
	logger *logger.Logger
}

// WithTracer 链路追踪
func WithTracer(tracer tracing.Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

// WithLogger 日志
func WithLogger(logger *logger.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

type Cron struct {
	c      *cron.Cron
	tracer tracing.Tracer
	lgr    *logger.Logger
}

func NewCron(opts ...Option) *Cron {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	c := &Cron{
		c: cron.New(
			cron.WithSeconds(),
			cron.WithChain(
				cron.Recover(cron.DefaultLogger),
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
		),
		tracer: o.tracer,
		lgr:    o.logger,
	}
	return c
}

func (c *Cron) AddFunc(spec string, cmd func(ctx context.Context)) (cron.EntryID, error) {
	var cmdName string
	pc := runtime.FuncForPC(reflect.ValueOf(cmd).Pointer())
	if pc != nil {
		cmdName = pc.Name()
	} else {
		cmdName = "unknown"
	}
	fn := func() {
		sCtx := context.Background()
		// 记录任务开始时间
		beginTime := time.Now()
		var span trace.Span
		if c.tracer != nil {
			sCtx, span = c.tracer.Start(sCtx,
				fmt.Sprintf("cron.%s", cmdName),
				trace.WithSpanKind(trace.SpanKindInternal),
				trace.WithAttributes(
					attribute.String("cron.function", cmdName),
					attribute.Int64("cron.begin", beginTime.Unix()),
				),
			)
		}
		if c.lgr != nil {
			c.lgr.Debug(sCtx, "cron begin func:%v, timestamp:%v",
				cmdName, beginTime.Unix())
		}
		// 执行任务，并捕获 panic
		defer func() {
			if r := recover(); r != nil {
				// 记录Span
				if span != nil {
					span.RecordError(fmt.Errorf("panic: %v", r))
					span.SetStatus(codes.Error, "Task panicked")
				}
				// 记录panic日志
				if c.lgr != nil {
					c.lgr.Error(sCtx, "cron panic func:%v, recover:%v",
						cmdName, r)
				}
			}
			if c.lgr != nil {
				c.lgr.Info(sCtx, "cron finished func:%v, duration:%vms",
					cmdName, time.Since(beginTime).Milliseconds())
			}
			// 记录任务结束
			if span != nil {
				span.SetAttributes(
					attribute.Int64("cron.duration_ms", time.Since(beginTime).Milliseconds()),
					attribute.Int64("cron.end", time.Now().Unix()),
				)
				span.End()
			}
		}()
		// 执行业务逻辑
		cmd(sCtx)
		// 结束span
		if span != nil {
			span.SetStatus(codes.Ok, "func completed")
		}
	}
	return c.c.AddFunc(spec, fn)
}

func (c *Cron) Start() {
	c.c.Start()
}

func (c *Cron) Stop() {
	c.c.Stop()
}
