package nacos

import (
	"context"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"
)

type Logger struct {
	*logger.Logger
}

func (lgr *Logger) Info(args ...interface{}) {
	lgr.Logger.Info(context.TODO(), "", args...)
}

func (lgr *Logger) Warn(args ...interface{}) {
	lgr.Logger.Warn(context.TODO(), "", args...)
}

func (lgr *Logger) Error(args ...interface{}) {
	lgr.Logger.Error(context.TODO(), "", args...)
}

func (lgr *Logger) Debug(args ...interface{}) {
	lgr.Logger.Debug(context.TODO(), "", args...)
}

func (lgr *Logger) Infof(fmt string, args ...interface{}) {
	lgr.Logger.Info(context.TODO(), fmt, args...)
}

func (lgr *Logger) Warnf(fmt string, args ...interface{}) {
	lgr.Logger.Warn(context.TODO(), fmt, args...)
}

func (lgr *Logger) Errorf(fmt string, args ...interface{}) {
	lgr.Logger.Error(context.TODO(), fmt, args...)
}

func (lgr *Logger) Debugf(fmt string, args ...interface{}) {
	lgr.Logger.Debug(context.TODO(), fmt, args...)
}
