package consumer

import (
	"context"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/opentracing/opentracing-go"
)

type OnMessage func(context.Context, ...*primitive.MessageExt) error

type handler func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)

type Option func(*consumerOption)

type consumerOption struct {
	credentials       primitive.Credentials
	nameServer        []string
	retry             int
	consumeTimeout    time.Duration
	interceptors      []primitive.Interceptor
	tracer            opentracing.Tracer
	model             consumer.MessageModel
	from              consumer.ConsumeFromWhere
	groupName         string
	goroutineNums     int
	maxReconsumeTimes int32
	expression        string
}

// WithModel 0:BroadCasting 1:Clustering
func WithModel(model int) Option {
	return func(o *consumerOption) {
		if model == int(consumer.BroadCasting) || model == int(consumer.Clustering) {
			o.model = consumer.MessageModel(model)
		}
	}
}

// WithFrom 0:last commit 1:first 2:timestamp
func WithFrom(from int) Option {
	return func(o *consumerOption) {
		if from == int(consumer.ConsumeFromLastOffset) ||
			from == int(consumer.ConsumeFromFirstOffset) ||
			from == int(consumer.ConsumeFromTimestamp) {
			o.from = consumer.ConsumeFromWhere(from)
		}
	}
}

// WithCredentials 鉴权
func WithCredentials(credentials primitive.Credentials) Option {
	return func(o *consumerOption) {
		o.credentials = credentials
	}
}

// WithNameServer 服务地址
func WithNameServer(ns []string) Option {
	return func(o *consumerOption) {
		o.nameServer = ns
	}
}

// WithConsumeTimeout 消息在消费队列最大时间
func WithConsumeTimeout(consumeTimeout time.Duration) Option {
	return func(o *consumerOption) {
		o.consumeTimeout = consumeTimeout
	}
}

// WithRetry 重试设置
func WithRetry(retry int) Option {
	return func(o *consumerOption) {
		o.retry = retry
	}
}

// WithInterceptors 拦截器
func WithInterceptors(interceptors []primitive.Interceptor) Option {
	return func(o *consumerOption) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

// WithTracer 链路追踪
func WithTracer(tracer opentracing.Tracer) Option {
	return func(o *consumerOption) {
		o.tracer = tracer
	}
}

// WithGroupName 分组消费者
func WithGroupName(groupName string) Option {
	return func(o *consumerOption) {
		o.groupName = groupName
	}
}

// WithGoroutineNums 消费者启动goroutine数量
func WithGoroutineNums(goroutineNums int) Option {
	return func(o *consumerOption) {
		o.goroutineNums = goroutineNums
	}
}

// WithMaxReconsumeTimes 消费重试次数
func WithMaxReconsumeTimes(maxReconsumeTimes int32) Option {
	return func(o *consumerOption) {
		o.maxReconsumeTimes = maxReconsumeTimes
	}
}

// WithTagExpression tag过滤其
func WithTagExpression(expression string) Option {
	return func(o *consumerOption) {
		o.expression = expression
	}
}
