package server

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/logger"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"
	oCodes "go.opentelemetry.io/otel/codes"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

///////////////////////////////////////////
// 服务端拦截器
///////////////////////////////////////////

func RecoverServerInterceptor(lgr *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			var rerr interface{}
			if rerr = recover(); rerr != nil {
				var buf [1 << 10]byte
				runtime.Stack(buf[:], true)
				lgr.Error(ctx, "grpc error, server:%v, method:%s, message:%v\n, stack:%s", info.Server, info.FullMethod, rerr, string(buf[:]))

				hub := sentry.CurrentHub().Clone()
				hub.CaptureException(errors.New(string(buf[:])))
				hub.Flush(5 * time.Second)

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func TracingServerUnaryInterceptor(tracer tracing.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		rpcMD, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			rpcMD = metadata.New(nil)
		}
		md := gCtx.Metadata{MD: rpcMD}

		propagator := otel.GetTextMapPropagator()
		pCtx := propagator.Extract(ctx, md)
		// 创建服务端 span
		nCtx, span := tracer.Start(pCtx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCMethodKey.String(info.FullMethod),
			),
		)
		defer span.End()
		// context 写入上游服务信息
		// 获取远端ip
		var peerIP string
		p, ok := peer.FromContext(nCtx)
		if ok {
			peerIP = p.Addr.String()
			peerIP = strings.Split(peerIP, ":")[0]
		}
		sk := gCtx.GetUberMeta(md)
		ss := strings.Split(sk, ".")
		nCtx = gCtx.NewClientContext(nCtx, gCtx.TransData{
			Endpoint:    peerIP,
			Namespace:   ss[0],
			Product:     ss[1],
			ServiceName: strings.Join(ss[2:], "."),
		})
		rsp, err := handler(nCtx, req)
		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(oCodes.Error, err.Error())
		} else {
			span.SetStatus(oCodes.Ok, "OK")
		}
		return rsp, err
	}
}

func AccessServerUnaryInterceptor(lgr *logger.Logger, requestEnable bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// 获取远端服务信息
		td, ok := gCtx.FromClientContext(ctx)
		if !ok {
			td = gCtx.TransData{}
		}
		// start时间
		start := time.Now().UnixMilli()

		defer func() {
			logData := map[string]interface{}{
				"Peer":   td,
				"Method": info.FullMethod,
				"Cost":   fmt.Sprintf("%dms", time.Now().UnixMilli()-start),
			}
			if requestEnable {
				logData["Request"] = req
			}
			if err != nil {
				logData["Error"] = err.Error()
			}
			if resp != nil {
				logData["Response"] = resp
			}
			bs, e := json.Marshal(logData)
			if e != nil {
				return
			}
			lgr.Write(ctx, "%s", string(bs))
		}()
		return handler(ctx, req)
	}
}

func LangServerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		rpcMD, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			rpcMD = metadata.New(nil)
		}
		md := gCtx.Metadata{MD: rpcMD}
		lang := gCtx.GetUberLangHeader(md)
		if lang != "" {
			ctx = gCtx.NewLangClientContext(ctx, lang)
		}
		return handler(ctx, req)
	}
}
