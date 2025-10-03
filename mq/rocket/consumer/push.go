package consumer

import (
	"context"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/mq/rocket/metadata"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

type PushConsumer struct {
	ctx        context.Context
	cli        rocketmq.PushConsumer
	handlers   map[string]handler // key:topic value:handler
	tracer     opentracing.Tracer
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
		for _, msg := range msgs {
			if p.tracer != nil {
				var opts []opentracing.StartSpanOption
				// 从header中获取spanContext,如果没有，则这里新建一个
				// 提取
				m := metadata.NewMetaMapWithData(msg.GetProperties())
				spanCtx, err := p.tracer.Extract(opentracing.TextMap, m)
				if err != nil {
					// 如果提取失败
					opts = []opentracing.StartSpanOption{
						ext.SpanKindConsumer,
						opentracing.Tag{Key: string(ext.Component), Value: "rocket"},
					}
				} else {
					opts = []opentracing.StartSpanOption{
						opentracing.ChildOf(spanCtx),
						ext.SpanKindConsumer,
						opentracing.Tag{Key: string(ext.Component), Value: "rocket"},
					}
				}
				span := p.tracer.StartSpan(msg.Topic, opts...)
				ctx = opentracing.ContextWithSpan(context.TODO(), span)
				defer span.Finish()
			}
		}
		err := f(ctx, msgs...)
		if err != nil {
			return consumer.ConsumeRetryLater, err
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
