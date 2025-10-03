package server

import (
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

		var opts []opentracing.StartSpanOption
		opts = append(opts, opentracing.Tag{Key: string(ext.Component), Value: "socket.io server"})
		span := tracer.StartSpan(
			fmt.Sprintf("%s", c.conn.Namespace()),
			opts...,
		)
		defer span.Finish()

		c.ctx = opentracing.ContextWithSpan(c.ctx, span)
		c.Next()
	}
}

func WithLangHandler() core {
	return func(c *Context) {
		lang := gCtx.GetUberHttpLangHeader(c.conn.RemoteHeader())
		if lang != "" {
			c.ctx = gCtx.NewLangClientContext(c.ctx, lang)
		}
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
			logData := map[string]interface{}{
				"Peer":   td,
				"Method": c.conn.URL(),
				"Event":  c.event,
				"Cost":   fmt.Sprintf("%dms", time.Now().UnixMilli()-start),
			}
			if requestEnable {
				logData["Request"] = c.data
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
