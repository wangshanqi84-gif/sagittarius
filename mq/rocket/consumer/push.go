package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"
	"github.com/wangshanqi84-gif/sagittarius/mq/rocket/metadata"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PushConsumer struct {
	ctx        context.Context
	cli        rocketmq.PushConsumer
	handlers   map[string]handler // key:topic value:handler
	tracer     tracing.Tracer
	expression string
}

func NewPushConsumer(ctx context.Context, opts ...Option) (*PushConsumer, error) {
	options := consumerOption{
		consumeTimeout:    30 * time.Minute,
		retry:             2,
		from:              consumer.ConsumeFromLastOffset,
		model:             consumer.Clustering,
		maxReconsumeTimes: -1,
		goroutineNums:     20,
		expression:        "*",
	}
	for _, o := range opts {
		o(&options)
	}
	if len(options.nameServer) == 0 {
		return nil, errors.New("ROCKETMQ_NONE_SERVERNAME")
	}
	var conOpts []consumer.Option
	conOpts = append(conOpts, consumer.WithNsResolver(primitive.NewPassthroughResolver(options.nameServer)))
	if options.retry != 0 {
		if options.retry == -1 {
			options.retry = 0
		}
		conOpts = append(conOpts, consumer.WithRetry(options.retry))
	}
	if options.groupName != "" {
		conOpts = append(conOpts, consumer.WithGroupName(options.groupName))
	}
	if !options.credentials.IsEmpty() {
		conOpts = append(conOpts, consumer.WithCredentials(options.credentials))
	}
	if len(options.interceptors) > 0 {
		conOpts = append(conOpts, consumer.WithInterceptor(options.interceptors...))
	}
	conOpts = append(conOpts,
		consumer.WithConsumeFromWhere(options.from),
		consumer.WithConsumerModel(options.model),
		consumer.WithConsumeTimeout(options.consumeTimeout),
		consumer.WithConsumeGoroutineNums(options.goroutineNums),
		consumer.WithMaxReconsumeTimes(options.maxReconsumeTimes),
	)
	cli, err := rocketmq.NewPushConsumer(conOpts...)
	if err != nil {
		return nil, err
	}
	c := &PushConsumer{
		ctx:      ctx,
		cli:      cli,
		handlers: make(map[string]handler),
		tracer:   options.tracer,
	}
	return c, nil
}

func (p *PushConsumer) RegisterHandler(topic string, f OnMessage) error {
	if _, has := p.handlers[topic]; has {
		return errors.New("topic already registered")
	}
	fn := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		var errs []error
		for _, msg := range msgs {
			var span trace.Span
			nCtx := ctx
			if p.tracer != nil {
				// 从消息属性中提取父上下文
				pCtx := ctx
				// 提取
				if msg.GetProperties() != nil && len(msg.GetProperties()) > 0 {
					m := metadata.NewMetaMapWithData(msg.GetProperties())
					propagator := otel.GetTextMapPropagator()
					pCtx = propagator.Extract(ctx, m)
				}
				// 创建消费者 span
				nCtx, span = p.tracer.Start(pCtx,
					fmt.Sprintf("%s receive", msg.Topic),
					trace.WithSpanKind(trace.SpanKindConsumer),
					trace.WithAttributes(
						attribute.String("messaging.system", "rocketmq"),
						attribute.String("messaging.destination", msg.Topic),
						attribute.String("messaging.operation", "receive"),
						attribute.String("messaging.rocketmq.message_id", msg.MsgId),
						attribute.Int("messaging.rocketmq.reconsume_times", int(msg.ReconsumeTimes)),
					),
				)
			}
			err := f(nCtx, msg)
			if err != nil {
				errs = append(errs, err)
			}
			if span.SpanContext().IsValid() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				} else {
					span.SetStatus(codes.Ok, "")
				}
				span.End()
			}
		}
		if len(errs) > 0 {
			return consumer.ConsumeRetryLater, errs[0]
		}
		return consumer.ConsumeSuccess, nil
	}
	p.handlers[topic] = fn
	return nil
}

func (p *PushConsumer) Start() error {
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "*",
	}
	if p.expression != "*" {
		selector.Expression = p.expression
	}

	for topic, fn := range p.handlers {
		if err := p.cli.Subscribe(topic, selector, fn); err != nil {
			return err
		}
	}
	if err := p.cli.Start(); err != nil {
		return err
	}
	go func() {
		<-p.ctx.Done()
		p.cli.Shutdown()
	}()
	return nil
}
