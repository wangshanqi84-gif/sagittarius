package client

import (
	"context"
	"fmt"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

///////////////////////////////////////////
// 客户端拦截器
///////////////////////////////////////////

func TracingClientUnaryInterceptor(baseCtx context.Context, tracer opentracing.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		//一个RPC调用的服务端的span，和RPC服务客户端的span构成ChildOf关系
		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		span := tracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC Client"},
			ext.SpanKindRPCClient,
		)
		defer span.Finish()

		rpcMD, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			rpcMD = metadata.New(nil)
		} else {
			rpcMD = rpcMD.Copy()
		}
		md := gCtx.Metadata{MD: rpcMD}
		td, ok := gCtx.FromServerContext(baseCtx)
		if ok {
			gCtx.SetUberMeta(md, fmt.Sprintf("%s.%s.%s", td.Namespace, td.Product, td.ServiceName))
		}
		if err := tracer.Inject(span.Context(), opentracing.TextMap, md); err == nil {
			ctx = metadata.NewOutgoingContext(ctx, md.MD)
		}
		return invoker(ctx, method, request, reply, cc, opts...)
	}
}

func TimeoutClientUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return invoker(ctx, method, request, reply, cc, opts...)
	}
}

func RetryClientUnaryInterceptor(maxAttempts int) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var err error
		for att := 0; att <= maxAttempts; att++ {
			err = invoker(ctx, method, request, reply, cc, opts...)
			if err != nil {
				if status.Convert(err).Code() == codes.Unavailable ||
					status.Convert(err).Code() == codes.DeadlineExceeded {
					continue
				}
			}
			break
		}
		return err
	}
}

func LangClientUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rpcMD, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			rpcMD = metadata.New(nil)
		} else {
			rpcMD = rpcMD.Copy()
		}
		md := gCtx.Metadata{MD: rpcMD}
		gCtx.SetUberLangHeader(md, gCtx.FromLangClientContext(ctx))
		return nil
	}
}
