package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Invoker func(ctx context.Context, c *Client, req *http.Request) (*http.Response, error)

func getInvoker(interceptors []Interceptor, curr int, finalInvoker Invoker) Invoker {
	if curr == len(interceptors)-1 {
		return finalInvoker
	}
	return func(ctx context.Context, c *Client, req *http.Request) (*http.Response, error) {
		return interceptors[curr+1](ctx, c, req, getInvoker(interceptors, curr+1, finalInvoker))
	}
}

type Interceptor func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error)

func doInterceptors(ctx context.Context, cc *Client, req *http.Request) (*http.Response, error) {
	interceptors := cc.interceptors
	var start Interceptor
	if len(interceptors) == 0 {
		start = nil
	} else if len(interceptors) == 1 {
		start = interceptors[0]
	} else {
		start = func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error) {
			return interceptors[0](ctx, c, req, getInvoker(interceptors, 0, invoker))
		}
	}
	if start != nil {
		return start(ctx, cc, req, invoke)
	}
	return invoke(ctx, cc, req)
}

func invoke(ctx context.Context, c *Client, req *http.Request) (*http.Response, error) {
	if c.resolver != nil {
		var (
			node *registry.Service
			err  error
		)
		if node, err = c.resolver.balancer.Pick(ctx); err != nil {
			return nil, errors.New("SERVER_NOT_FOUND")
		}
		if c.insecure {
			req.URL.Scheme = "http"
		} else {
			req.URL.Scheme = "https"
		}
		host, ok := node.Endpoint(registry.ProtoHTTP)
		if !ok {
			return nil, errors.New("no matching address found")
		}
		if strings.Contains(host, "://") {
			ss := strings.Split(host, "://")
			req.URL.Scheme = ss[0]
			host = ss[1]
		}
		req.Host = host
		req.URL.Host = host
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

///////////////////////////////////////////
// 客户端拦截器
///////////////////////////////////////////

func TracingInterceptor(baseCtx context.Context, tracer tracing.Tracer) Interceptor {
	return func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error) {
		// 创建一个新的span 自动继承父span(如果有的话)
		nCtx, span := tracer.Start(ctx, req.URL.String(),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(req.Method),
				semconv.URLFullKey.String(req.URL.String()),
				semconv.URLSchemeKey.String(req.URL.Scheme),
				semconv.ServerAddressKey.String(req.URL.Host),
				semconv.URLPathKey.String(req.URL.Path),
			),
		)
		defer span.End()

		td, ok := gCtx.FromServerContext(baseCtx)
		if ok {
			gCtx.SetUberHttpHeader(req.Header, fmt.Sprintf("%s.%s.%s", td.Namespace, td.Product, td.ServiceName))
		}
		md := gCtx.HttpMetadata{Header: req.Header}
		// 注入上下文到 HTTP headers
		propagator := otel.GetTextMapPropagator()
		// todo test
		log.Printf("[SGT CLIENT] Propagator type: %T\n", propagator)
		log.Printf("[SGT CLIENT] Propagator fields: %+v\n", propagator)
		log.Printf("[SGT CLIENT] Before Inject, Header traceparent: %s\n", req.Header.Get("traceparent"))
		propagator.Inject(nCtx, &md)
		log.Printf("[SGT CLIENT] After Inject, Header traceparent: %s\n", req.Header.Get("traceparent"))

		rsp, err := invoker(nCtx, c, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		return rsp, err
	}
}

func WithLangInterceptor() Interceptor {
	return func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error) {
		gCtx.SetUberHttpLangHeader(req.Header, gCtx.FromLangClientContext(ctx))
		return invoker(ctx, c, req)
	}
}

func SyncTimeoutInterceptor() Interceptor {
	return func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error) {
		if c.syncTimeout {
			deadline := time.Now().UnixMilli() + c.httpClient.Timeout.Milliseconds()
			parDeadline := gCtx.FromTimeoutClientContext(ctx)
			if parDeadline != 0 {
				if deadline > parDeadline.Milliseconds() {
					deadline = parDeadline.Milliseconds()
				}
			}
			gCtx.SetUberHttpTimeoutHeader(req.Header, fmt.Sprintf("%d", deadline))
		}
		return invoker(ctx, c, req)
	}
}
