package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type CustomJsonEncoder func(context.Context) (string, string)

type LevelEncoder func(Level) string
type TimeEncoder func(time.Time) string
type CallerEncoder func() string

func defaultLevelEncoder(l Level) string {
	if l.isNoneLevel() {
		return ""
	}
	return "[" + l.String() + "]"
}

func defaultTimeEncoder(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func defaultCallEncoder() string {
	_, file, line, _ := runtime.Caller(4)

	ss := strings.Split(file, "/")
	if len(ss) > PathDeep {
		ss = ss[len(ss)-PathDeep:]
		file = "/" + strings.Join(ss, "/")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func traceEncoder(ctx context.Context) string {
	var traceID string
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		traceID = ""
	} else {
		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			traceID = sc.TraceID().String()
		}
	}
	return traceID
}
