package server

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
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
		spanContext, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.Request().Header),
		)
		var opts []opentracing.StartSpanOption
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			opts = append(opts, opentracing.Tag{Key: string(ext.Component), Value: "http Server"})
		} else {
			opts = append(opts, opentracing.ChildOf(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "http Server"})
		}
		span := tracer.StartSpan(
			c.Request().URL.String(),
			opts...,
		)
		defer span.Finish()

		c.ctx = opentracing.ContextWithSpan(c.ctx, span)
		// context 写入上游服务信息
		sk := gCtx.GetUberHttpHeader(c.Request().Header)
		if sk != "" {
			ss := strings.Split(sk, ".")
			c.ctx = gCtx.NewClientContext(c.ctx, gCtx.TransData{
				Endpoint:    c.Request().RemoteAddr,
				Namespace:   ss[0],
				Product:     ss[1],
				ServiceName: strings.Join(ss[2:], "."),
			})
		}
		c.Next()
	}
}

func SyncTimeoutHandler(lgr *logger.Logger) core {
	return func(c *Context) {
		sd := gCtx.GetUberHttpTimeoutHeader(c.Request().Header)
		if sd != "" {
			deadline, err := strconv.ParseInt(sd, 10, 64)
			if err != nil {
				lgr.Error(c.ctx, "strconv.ParseInt err:%v", err)
			}
			to := time.Duration(deadline - time.Now().UnixMilli())
			c.ctx = gCtx.NewTimeoutClientContext(c.ctx, to*time.Millisecond)
			c.ctx, _ = context.WithTimeout(c.ctx, to*time.Millisecond)
		}
		c.Next()
	}
}

func WithTimeoutHandler(sec int64, milliSec int64) core {
	return func(c *Context) {
		dc := time.Duration(sec)*time.Second + time.Duration(milliSec)*time.Millisecond

		c.ctx = gCtx.NewTimeoutClientContext(c.ctx, dc)
		c.ctx, _ = context.WithTimeout(c.ctx, dc)

		c.Next()
	}
}

func WithLangHandler() core {
	return func(c *Context) {
		lang := gCtx.GetUberHttpLangHeader(c.Request().Header)
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
				"Method": c.Request().URL.String(),
				"Cost":   fmt.Sprintf("%dms", time.Now().UnixMilli()-start),
			}
			if requestEnable && len(c.reqBody) != 0 {
				logData["Request"] = string(c.reqBody)
			}
			if c.respData != nil && !c.logWithoutResp {
				logData["Response"] = c.respData
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
