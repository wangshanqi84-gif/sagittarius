package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

///////////////////////////////////
// 当前暂无自定义需求 后续需要可自定义实现tracer span context
///////////////////////////////////

type Tracer interface {
	Start(context.Context, string, ...trace.SpanStartOption) (context.Context, trace.Span)
	Close() error
}
