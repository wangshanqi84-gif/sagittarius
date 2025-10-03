package mysql

import "time"

// MySQL 默认配置

const (
	DefaultMaxIdle  = 50
	DefaultMaxOpen  = 200
	DefaultLifeTime = time.Hour
	DefaultIdleTime = 15 * time.Minute
)
