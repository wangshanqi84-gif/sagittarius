package tracing

import "github.com/opentracing/opentracing-go"

///////////////////////////////////
// 当前暂无自定义需求 后续需要可自定义实现tracer span context
///////////////////////////////////

type Tracer interface {
	StartSpan(string, ...opentracing.StartSpanOption) opentracing.Span
	Inject(opentracing.SpanContext, interface{}, interface{}) error
	Extract(interface{}, interface{}) (opentracing.SpanContext, error)
	Close() error
}
