package db

import "time"

// MySQL 默认配置

const (
	DefaultMysqlMaxIdle    = 50
	DefaultMysqlMaxOpen    = 200
	DefaultPostgresMaxIdle = 5
	DefaultPostgresMaxOpen = 25
	DefaultLifeTime        = time.Hour
	DefaultIdleTime        = 15 * time.Minute
)
