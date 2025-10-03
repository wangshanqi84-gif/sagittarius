package context

import (
	"context"
	"time"
)

type (
	serverTransportKey struct{}
	clientTransportKey struct{}
	clientTimeoutKey   struct{}
	clientLangKey      struct{}
	forContextKey      struct{}
)

func AsCtx(ctx context.Context) context.Context {
	if ctx.Value(forContextKey{}) == nil {
		return ctx
	}
	if _, ok := ctx.Value(forContextKey{}).(context.Context); !ok {
		return ctx
	}
	return ctx.Value(forContextKey{}).(context.Context)
}

func NewTimeoutClientContext(ctx context.Context, t time.Duration) context.Context {
	return context.WithValue(ctx, clientTimeoutKey{}, t)
}

func FromTimeoutClientContext(ctx context.Context) time.Duration {
	to, ok := ctx.Value(clientTimeoutKey{}).(time.Duration)
	if !ok {
		return -1
	}
	return to
}

func NewLangClientContext(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, clientLangKey{}, lang)
}

func FromLangClientContext(ctx context.Context) string {
	lang, ok := ctx.Value(clientLangKey{}).(string)
	if !ok {
		return ""
	}
	return lang
}

type TransData struct {
	Endpoint    string `json:"host"`
	Namespace   string `json:"namespace"`
	Product     string `json:"product"`
	ServiceName string `json:"serviceName"`
}

func NewServerContext(ctx context.Context, td TransData) context.Context {
	return context.WithValue(ctx, serverTransportKey{}, td)
}

func FromServerContext(ctx context.Context) (TransData, bool) {
	td, ok := ctx.Value(serverTransportKey{}).(TransData)
	return td, ok
}

func NewClientContext(ctx context.Context, td TransData) context.Context {
	return context.WithValue(ctx, clientTransportKey{}, td)
}

func FromClientContext(ctx context.Context) (TransData, bool) {
	td, ok := ctx.Value(clientTransportKey{}).(TransData)
	return td, ok
}
