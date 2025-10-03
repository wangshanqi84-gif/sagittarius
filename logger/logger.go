package logger

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

var (
	_once  sync.Once
	busi   *logger.Logger // 业务日志
	access *logger.Logger // access日志
	gen    *logger.Logger // 框架日志
)

func init() {
	var opts []logger.Option
	if env.GetEnv(env.SgtLogPath) != "" {
		opts = append(opts, logger.SetPath(env.GetEnv(env.SgtLogPath)))
	}
	gen = logger.New("gen", opts...)
}

func InitLogger(level string, opts ...logger.Option) {
	_once.Do(func() {
		lvl := logger.DebugLevel
		switch strings.ToLower(level) {
		case "info":
			lvl = logger.InfoLevel
		case "warn":
			lvl = logger.WarnLevel
		case "error":
			lvl = logger.ErrorLevel
		}
		busi = logger.NewGroup(lvl, opts...)
		access = logger.New("access", opts...)
	})
}

func Debug(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	busi.Debug(gCtx.AsCtx(ctx), format, args...)
}

func Info(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	busi.Info(gCtx.AsCtx(ctx), format, args...)
}

func Warn(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	busi.Warn(gCtx.AsCtx(ctx), format, args...)
}

func Error(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	go func() {
		hub := sentry.CurrentHub().Clone()
		hub.CaptureException(errors.New(fmt.Sprintf(format, args)))
		hub.Flush(5 * time.Second)
	}()
	busi.Error(gCtx.AsCtx(ctx), format, args...)
}

func Panic(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	busi.Panic(gCtx.AsCtx(ctx), format, args...)
}

func Fatal(ctx context.Context, format string, args ...interface{}) {
	if busi == nil {
		return
	}
	busi.Fatal(gCtx.AsCtx(ctx), format, args...)
}

func Gen(ctx context.Context, format string, args ...interface{}) {
	if gen == nil {
		return
	}
	gen.Write(gCtx.AsCtx(ctx), format, args...)
}

func Access(ctx context.Context, format string, args ...interface{}) {
	if access == nil {
		return
	}
	access.Write(gCtx.AsCtx(ctx), format, args...)
}

func GetLogger() *logger.Logger {
	return busi
}

func GetAccess() *logger.Logger {
	return access
}

func GetGen() *logger.Logger {
	return gen
}
