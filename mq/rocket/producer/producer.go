package producer

import (
	"context"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/mq/rocket/metadata"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

type Producer struct {
	ctx    context.Context
	cli    rocketmq.Producer
	topics map[string]string
	tracer opentracing.Tracer
}

func NewProducer(ctx context.Context, opts ...Option) (*Producer, error) {
	options := producerOption{
		timeout: 3 * time.Second,
		retry:   2,
	}
	for _, o := range opts {
		o(&options)
	}
	if len(options.nameServer) == 0 {
		return nil, errors.New("ROCKETMQ_NONE_SERVERNAME")
	}
	var proOpts []producer.Option
	proOpts = append(proOpts, producer.WithNsResolver(primitive.NewPassthroughResolver(options.nameServer)))
	if options.retry != 0 {
		if options.retry == -1 {
			options.retry = 0
		}
		proOpts = append(proOpts, producer.WithRetry(options.retry))
	}
	if !options.credentials.IsEmpty() {
		proOpts = append(proOpts, producer.WithCredentials(options.credentials))
	}
	if len(options.interceptors) > 0 {
		proOpts = append(proOpts, producer.WithInterceptor(options.interceptors...))
	}
	proOpts = append(proOpts, producer.WithSendMsgTimeout(options.timeout))
	cli, err := rocketmq.NewProducer(proOpts...)
	if err != nil {
		return nil, err
	}
	p := &Producer{
		ctx:    ctx,
		cli:    cli,
		topics: make(map[string]string),
	}
	if options.tracer != nil {
		p.tracer = options.tracer
	}
	for k, v := range options.topics {
		p.topics[k] = v
	}
	err = p.cli.Start()
	if err != nil {
		return nil, err
	}
	go func() {
		select {
		case <-p.ctx.Done():
			p.cli.Shutdown()
		}
	}()
	return p, nil
}

func (p *Producer) SyncSend(ctx context.Context, alias string,
	data []byte, opts ...SendOption) (*primitive.SendResult, error) {
	o := sendOption{}
	for _, opt := range opts {
		opt(&o)
	}
	topic := alias
	if _, has := p.topics[alias]; has {
		topic = p.topics[alias]
	}
	// 做成消息
	msg := primitive.NewMessage(topic, data)
	if o.sharding != "" {
		msg = msg.WithShardingKey(o.sharding)
	}
	if len(o.keys) > 0 {
		msg = msg.WithKeys(o.keys)
	}
	if o.tags != "" {
		msg = msg.WithTag(o.tags)
	}
	// 链路追踪
	if p.tracer != nil {
		// 从context中获取spanContext,如果上层没有开启追踪，则这里新建一个
		var parentCtx opentracing.SpanContext
		if parent := opentracing.SpanFromContext(ctx); parent != nil {
			parentCtx = parent.Context()
		}
		span := p.tracer.StartSpan(
			topic,
			opentracing.ChildOf(parentCtx),
			ext.SpanKindProducer,
			opentracing.Tag{Key: string(ext.Component), Value: "rocket"},
		)
		// 注入context
		m := metadata.NewMetaMap()
		err := p.tracer.Inject(span.Context(), opentracing.TextMap, m)
		if err != nil {
			// 注入失败则直接发送消息
			return p.cli.SendSync(ctx, msg)
		}
		// 将注入信息写入header进行传递
		msg.WithProperties(m.Data())
	}
	// 发送消息
	return p.cli.SendSync(ctx, msg)
}
