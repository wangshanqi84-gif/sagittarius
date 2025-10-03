package mysql

import (
	"context"
	"errors"
	"time"

	log "github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type OrmLogger struct {
	*log.Logger
}

// Trace 实现gorm logger接口用
func (l *OrmLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin).Nanoseconds() / 1e6
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		if rows == -1 {
			l.Error(ctx, "(GORM_ERR_LOG): err:%v, coast:%d, [-]sql:%s", err, elapsed, sql)
		} else {
			l.Error(ctx, "(GORM_ERR_LOG): err:%v, coast:%d, [%d]sql:%s", err, elapsed, rows, sql)
		}
	case elapsed > 500:
		sql, rows := fc()
		if rows == -1 {
			l.Warn(ctx, "(GORM_SLOW_LOG): coast:%d, [-]sql:%s", elapsed, sql)
		} else {
			l.Warn(ctx, "(GORM_SLOW_LOG): coast:%d, [%d]sql:%s", elapsed, rows, sql)
		}
	default:
		sql, rows := fc()
		if rows == -1 {
			l.Info(ctx, "(GORM_LOG): coast:%d, [-]sql:%s", elapsed, sql)
		} else {
			l.Info(ctx, "(GORM_LOG): coast:%d, [%d]sql:%s", elapsed, rows, sql)
		}
	}
}

// LogMode 实现gorm logger接口用
func (l *OrmLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}
