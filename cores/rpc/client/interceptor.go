package client

import (
	"context"
	"fmt"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"

	"go.opentelemetry.io/otel"
	oCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

///////////////////////////////////////////
// 客户端拦截器
///////////////////////////////////////////

func TracingClientUnaryInterceptor(baseCtx context.Context, tracer tracing.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 从 context 中获取当前的 span（如果有的话）
		// OpenTelemetry 会自动处理父子关系
		nCtx, span := tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCMethodKey.String(method),
			),
		)
		defer span.End()

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
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(ctx, &md)

		nCtx = metadata.NewOutgoingContext(nCtx, md.MD)
		err := invoker(nCtx, method, request, reply, cc, opts...)
		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(oCodes.Error, err.Error())
		} else {
			span.SetStatus(oCodes.Ok, "OK")
		}
		return err
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
		ctx = metadata.NewOutgoingContext(ctx, md.MD)
		return invoker(ctx, method, request, reply, cc, opts...)
	}
}
