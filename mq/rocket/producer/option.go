package producer

import (
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/opentracing/opentracing-go"
)

type Option func(*producerOption)

type producerOption struct {
	nameServer   []string
	credentials  primitive.Credentials
	timeout      time.Duration
	retry        int
	topics       map[string]string
	interceptors []primitive.Interceptor
	tracer       opentracing.Tracer
}

func WithNameServer(ns []string) Option {
	return func(o *producerOption) {
		o.nameServer = ns
	}
}

func WithCredentials(credentials primitive.Credentials) Option {
	return func(o *producerOption) {
		o.credentials = credentials
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *producerOption) {
		o.timeout = timeout
	}
}

func WithRetry(retry int) Option {
	return func(o *producerOption) {
		o.retry = retry
	}
}

func WithInterceptors(interceptors []primitive.Interceptor) Option {
	return func(o *producerOption) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

func WithTracer(tracer opentracing.Tracer) Option {
	return func(o *producerOption) {
		o.tracer = tracer
	}
}

func WithTopics(topics map[string]string) Option {
	return func(o *producerOption) {
		o.topics = make(map[string]string)
		for k, v := range topics {
			o.topics[k] = v
		}
	}
}

type SendOption func(*sendOption)

type sendOption struct {
	sharding string
	keys     []string
	tags     string
}

func WithSharding(sharding string) SendOption {
	return func(o *sendOption) {
		o.sharding = sharding
	}
}

func WithKeys(keys []string) SendOption {
	return func(o *sendOption) {
		o.keys = keys
	}
}

func WithTags(tags string) SendOption {
	return func(o *sendOption) {
		o.tags = tags
	}
}
