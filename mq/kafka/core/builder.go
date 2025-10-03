package core

import (
	"context"
	"fmt"
	"strings"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"

	"github.com/IBM/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

///////////////////////////////
// kafka消息builder
///////////////////////////////

type IMessageBuilder interface {
	ProducerMessage(ctx context.Context, topic string, key []byte, data []byte, ver sarama.KafkaVersion) *ProducerMessage
	ConsumerMessage(ctx context.Context, message *sarama.ConsumerMessage, ver sarama.KafkaVersion) *ConsumerMessage
}

type Builder struct {
	tracer opentracing.Tracer
}

func NewMessageBuilder(tracer opentracing.Tracer) *Builder {
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
		// 从context中获取spanContext,如果上层没有开启追踪，则这里新建一个
		var parentCtx opentracing.SpanContext
		if parent := opentracing.SpanFromContext(ctx); parent != nil {
			parentCtx = parent.Context()
		}
		span := b.tracer.StartSpan(
			topic,
			opentracing.ChildOf(parentCtx),
			ext.SpanKindProducer,
			opentracing.Tag{Key: string(ext.Component), Value: "kafka"},
		)
		// 注入context
		m := new(TextMapMeta)
		td, ok := gCtx.FromServerContext(ctx)
		if ok {
			m.SetUberMeta(fmt.Sprintf("%s.%s.%s", td.Namespace, td.Product, td.ServiceName))
		}
		err := b.tracer.Inject(span.Context(), opentracing.TextMap, m)
		if err != nil {
			// 注入失败则直接返回消息
			return &pm
		}
		// 将注入信息写入header进行传递
		pm.msg.Headers = append(pm.msg.Headers, m.Data...)
	}
	return &pm
}

func (b *Builder) ConsumerMessage(ctx context.Context, message *sarama.ConsumerMessage, ver sarama.KafkaVersion) *ConsumerMessage {
	var opts []opentracing.StartSpanOption
	if ver.IsAtLeast(sarama.V0_11_0_0) {
		// 从header中获取spanContext,如果没有，则这里新建一个
		// 提取
		m := new(TextMapMeta)
		for _, h := range message.Headers {
			if h != nil {
				m.Data = append(m.Data, *h)
			}
		}
		spanCtx, err := b.tracer.Extract(opentracing.TextMap, m)
		if err != nil {
			// 如果提取失败
			opts = []opentracing.StartSpanOption{
				ext.SpanKindConsumer,
				opentracing.Tag{Key: string(ext.Component), Value: "kafka"},
			}
		} else {
			opts = []opentracing.StartSpanOption{
				opentracing.ChildOf(spanCtx),
				ext.SpanKindConsumer,
				opentracing.Tag{Key: string(ext.Component), Value: "kafka"},
			}
		}
		sk := m.GetUberMeta()
		if sk != "" {
			ss := strings.Split(sk, ".")
			ctx = gCtx.NewClientContext(ctx, gCtx.TransData{
				Namespace:   ss[0],
				Product:     ss[1],
				ServiceName: strings.Join(ss[2:], "."),
			})
		}
	} else {
		opts = []opentracing.StartSpanOption{
			ext.SpanKindConsumer,
			opentracing.Tag{Key: string(ext.Component), Value: "kafka"},
		}
	}
	span := b.tracer.StartSpan(message.Topic, opts...)
	c := opentracing.ContextWithSpan(ctx, span)
	return &ConsumerMessage{
		ctx: c,
		msg: message,
	}
}
