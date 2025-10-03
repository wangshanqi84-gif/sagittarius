package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
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
		if _, has := node.Hosts["http"]; !has {
			return nil, errors.New("no matching address found")
		}
		host := node.Hosts["http"]
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

func TracingInterceptor(baseCtx context.Context, tracer opentracing.Tracer) Interceptor {
	return func(ctx context.Context, c *Client, req *http.Request, invoker Invoker) (*http.Response, error) {
		//一个http调用的服务端的span，和http服务客户端的span构成ChildOf关系
		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		span := tracer.StartSpan(
			req.URL.String(),
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "http Client"},
		)
		defer span.Finish()

		td, ok := gCtx.FromServerContext(baseCtx)
		if ok {
			gCtx.SetUberHttpHeader(req.Header, fmt.Sprintf("%s.%s.%s", td.Namespace, td.Product, td.ServiceName))
		}
		carrier := opentracing.HTTPHeadersCarrier(req.Header)
		if err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, carrier); err != nil {
			return nil, err
		}
		return invoker(ctx, c, req)
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
