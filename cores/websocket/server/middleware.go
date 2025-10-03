package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"github.com/getsentry/sentry-go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
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

func TracingHandler(tracer opentracing.Tracer) core {
	return func(c *Context) {
		buffer := bytes.NewBuffer(c.Header().(IHeader).Trace())
		spanContext, err := tracer.Extract(
			opentracing.Binary,
			buffer,
		)
		var opts []opentracing.StartSpanOption
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			opts = append(opts, opentracing.Tag{Key: string(ext.Component), Value: "websocket Server"})
		} else {
			opts = append(opts, opentracing.ChildOf(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "websocket Server"})
		}
		span := tracer.StartSpan(
			fmt.Sprintf("%d", c.Header().(IHeader).MsgID()),
			opts...,
		)
		defer span.Finish()

		c.ctx = opentracing.ContextWithSpan(c.ctx, span)
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
				"MessageID": c.Header().(IHeader).MsgID(),
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
