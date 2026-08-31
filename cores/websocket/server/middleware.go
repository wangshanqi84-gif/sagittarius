package server

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/logger"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"
	"github.com/wangshanqi84-gif/sagittarius/cores/websocket/context"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

///////////////////////////////////////////
// 服务端中间件
///////////////////////////////////////////

func PanicHandler(lgr *logger.Logger) core {
	return func(c *Context) {
		defer func() {
			var rerr interface{}
			if rerr = recover(); rerr != nil {
				var buf [1 << 10]byte
				runtime.Stack(buf[:], true)
				lgr.Error(c.Ctx(), "http error, message:%v\n, stack:%s", rerr, string(buf[:]))

				hub := sentry.CurrentHub().Clone()
				hub.CaptureException(errors.New(string(buf[:])))
				hub.Flush(5 * time.Second)
			}
		}()
		c.Next()
	}
}

func TracingHandler(tracer tracing.Tracer) core {
	return func(c *Context) {
		// 从二进制数据中提取父上下文
		if len(c.Header().Trace()) > 0 {
			var traceCtx context.TraceContext
			if err := json.Unmarshal(c.Header().Trace(), &traceCtx); err == nil {
				carrier := propagation.MapCarrier{
					"trace": fmt.Sprintf("%s-%s",
						traceCtx.TraceID,
						traceCtx.SpanID,
					),
				}
				propagator := otel.GetTextMapPropagator()
				c.ctx = propagator.Extract(c.ctx, carrier)
			}
		}
		// 创建服务端 span
		nCtx, span := tracer.Start(c.ctx,
			fmt.Sprintf("%d", c.Header().MsgID()),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("ws.message_id", fmt.Sprintf("%d", c.Header().MsgID())),
			),
		)
		defer span.End()

		c.ctx = nCtx
		c.Next()
	}
}

func LogHandler(lgr *logger.Logger, requestEnable bool) core {
	return func(c *Context) {
		// 获取远端服务信息
		td, ok := gCtx.FromClientContext(c.ctx)
		if !ok {
			td = gCtx.TransData{}
		}
		// start时间
		start := time.Now().UnixMilli()

		defer func() {
			if c.disableAccess {
				return
			}
			logData := map[string]interface{}{
				"Peer":      td,
				"MessageID": c.Header().MsgID(),
				"Cost":      fmt.Sprintf("%dms", time.Now().UnixMilli()-start),
			}
			if requestEnable {
				logData["Request"] = string(c.data)
			}
			bs, e := json.Marshal(logData)
			if e != nil {
				return
			}
			lgr.Write(c.Ctx(), "%s", string(bs))
		}()
		c.Next()
	}
}
