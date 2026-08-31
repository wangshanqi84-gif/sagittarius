package core

import (
	"context"
	"fmt"
	"strings"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

///////////////////////////////
// kafka消息builder
///////////////////////////////

type IMessageBuilder interface {
	ProducerMessage(ctx context.Context, topic string, key []byte, data []byte, ver sarama.KafkaVersion) *ProducerMessage
	ConsumerMessage(ctx context.Context, message *sarama.ConsumerMessage, ver sarama.KafkaVersion) *ConsumerMessage
}

type Builder struct {
	tracer tracing.Tracer
}

func NewMessageBuilder(tracer tracing.Tracer) *Builder {
	return &Builder{
		tracer: tracer,
	}
}

func (b *Builder) ProducerMessage(ctx context.Context, topic string, key []byte, data []byte, ver sarama.KafkaVersion) *ProducerMessage {
	// kafka消息信息
	pm := ProducerMessage{
		ctx: ctx,
		msg: &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.ByteEncoder(key),
			Value: sarama.ByteEncoder(data),
		},
	}
	if ver.IsAtLeast(sarama.V0_11_0_0) {
		// 创建生产者span
		nCtx, span := b.tracer.Start(ctx, fmt.Sprintf("%s", topic),
			trace.WithSpanKind(trace.SpanKindProducer),
			trace.WithAttributes(
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination", topic),
				attribute.String("messaging.operation", "send"),
				attribute.String("messaging.kafka.message_key", string(key)),
				attribute.Int("messaging.kafka.message_size", len(data)),
			),
		)

		// 注入context
		m := new(TextMapMeta)
		td, ok := gCtx.FromServerContext(ctx)
		if ok {
			m.SetUberMeta(fmt.Sprintf("%s.%s.%s", td.Namespace, td.Product, td.ServiceName))
		}
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(nCtx, m)
		// 将注入信息写入header进行传递
		pm.msg.Headers = append(pm.msg.Headers, m.Data...)
		pm.span = span
	}
	return &pm
}

func (b *Builder) ConsumerMessage(ctx context.Context, message *sarama.ConsumerMessage, ver sarama.KafkaVersion) *ConsumerMessage {
	pCtx := ctx
	if ver.IsAtLeast(sarama.V0_11_0_0) {
		// 从header中获取spanContext,如果没有，则这里新建一个
		// 提取
		m := new(TextMapMeta)
		for _, h := range message.Headers {
			if h != nil {
				m.Data = append(m.Data, *h)
			}
		}
		// 提取父上下文
		propagator := otel.GetTextMapPropagator()
		pCtx = propagator.Extract(ctx, m)
		// 提取UberMeta并设置客户端上下文
		sk := m.GetUberMeta()
		if sk != "" {
			ss := strings.Split(sk, ".")
			pCtx = gCtx.NewClientContext(pCtx, gCtx.TransData{
				Namespace:   ss[0],
				Product:     ss[1],
				ServiceName: strings.Join(ss[2:], "."),
			})
		}
	}
	// 创建消费者span
	nCtx, span := b.tracer.Start(pCtx, message.Topic,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", message.Topic),
			attribute.String("messaging.operation", "receive"),
			attribute.Int("messaging.kafka.partition", int(message.Partition)),
			attribute.Int64("messaging.kafka.offset", message.Offset),
		),
	)

	return &ConsumerMessage{
		ctx:  nCtx,
		msg:  message,
		span: span,
	}
}
