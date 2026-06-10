package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type Option func(c *Client)

// GormConfig
// 设置gorm配置
func GormConfig(cfg *gorm.Config) Option {
	return func(c *Client) {
		c.gormCfg = cfg
	}
}

// MaxOpen
// 设置打开数据库连接的最大数量
func MaxOpen(maxOpen int) Option {
	return func(c *Client) {
		c.maxOpen = maxOpen
	}
}

// MaxIdle
// 设置空闲连接池中连接的最大数量
func MaxIdle(maxIdle int) Option {
	return func(c *Client) {
		c.maxIdle = maxIdle
	}
}

// Logger
// 设置内部日志记录器
func Logger(lgr *logger.Logger) Option {
	return func(c *Client) {
		c.logger = lgr
	}
}

// MaxIdleTime
// 设置连接池链接最大空闲时长
func MaxIdleTime(maxIdleTime time.Duration) Option {
	return func(c *Client) {
		c.maxIdleTime = maxIdleTime
	}
}

// MaxLifeTime
// 设置连接可复用的最大时间
func MaxLifeTime(maxLifetime time.Duration) Option {
	return func(c *Client) {
		c.maxLifetime = maxLifetime
	}
}

// DriverName
// 设置db类型
func DriverName(driverName string) Option {
	return func(c *Client) {
		c.driverName = driverName
	}
}

type Client struct {
	driverName  string
	resolver    *dbresolver.DBResolver
	db          *gorm.DB
	gormCfg     *gorm.Config
	logger      *logger.Logger
	maxOpen     int
	maxIdle     int
	maxLifetime time.Duration
	maxIdleTime time.Duration
}

func NewClient(master string, slave []string, opts ...Option) (*Client, error) {
	// 初始化
	c := &Client{
		driverName: "mysql",
	}
	// option 执行
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	// driverName 检查
	if c.driverName != "mysql" && c.driverName != "postgres" {
		return nil, errors.New(`driverName must be "mysql" or "postgres"`)
	}
	// 读写分离配置
	if len(slave) > 0 {
		var dail []gorm.Dialector
		for _, dns := range slave {
			dail = append(dail, c.createDial(dns))
		}
		resolver := dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{c.createDial(master)},
			Replicas: dail,
		})
		c.resolver = resolver
	}
	// 链接
	if err := c.open(master); err != nil {
		return nil, err
	}
	sqlDB, err := c.db.DB()
	if err != nil {
		return nil, err
	}
	// 设置打开数据库连接的最大数量
	if c.maxOpen == 0 {
		if c.driverName == "postgres" {
			c.maxOpen = DefaultPostgresMaxOpen
		} else {
			c.maxOpen = DefaultMysqlMaxOpen
		}
	}
	sqlDB.SetMaxOpenConns(c.maxOpen)
	// 设置空闲连接池中连接的最大数量
	if c.maxIdle == 0 {
		if c.driverName == "postgres" {
			c.maxIdle = DefaultPostgresMaxIdle
		} else {
			c.maxIdle = DefaultMysqlMaxIdle
		}
	}
	sqlDB.SetMaxIdleConns(c.maxIdle)
	// 设置连接可复用的最大时间
	if c.maxLifetime == 0 {
		c.maxLifetime = DefaultLifeTime
	}
	sqlDB.SetConnMaxLifetime(c.maxLifetime)
	// 设置连接池链接最大空闲时长
	if c.maxIdleTime == 0 {
		c.maxIdleTime = DefaultIdleTime
	}
	sqlDB.SetConnMaxIdleTime(c.maxIdleTime)
	if err = sqlDB.Ping(); err != nil {
		return nil, errors.New(fmt.Sprintf("database ping failed: %v", err))
	}
	return c, nil
}

func (c *Client) createDial(dsn string) gorm.Dialector {
	switch c.driverName {
	case "postgres":
		return postgres.Open(dsn)
	default:
		return mysql.Open(dsn)
	}
}

func (c *Client) open(master string) error {
	if c.gormCfg == nil {
		c.gormCfg = &gorm.Config{}
	}
	if c.logger != nil {
		c.gormCfg.Logger = &OrmLogger{
			Logger: c.logger,
		}
	}
	var db *gorm.DB
	var err error
	switch c.driverName {
	case "postgres":
		db, err = gorm.Open(postgres.Open(master), c.gormCfg)
	default:
		db, err = gorm.Open(mysql.Open(master), c.gormCfg)
	}
	if err != nil {
		return err
	}
	if c.resolver != nil {
		err = db.Use(c.resolver)
		if err != nil {
			return err
		}
	}
	c.db = db
	return nil
}

// DB 默认库(读写分离环境下会根据语句类型选择主库或者从库)
func (c *Client) DB(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx)
}

// Master 强制指定主库(读写分离环境下可以通过Master方法将读操作指定到主库)
func (c *Client) Master(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx).Clauses(dbresolver.Write)
}

func (c *Client) Ping() error {
	db, err := c.db.DB()
	if err != nil {
		return err
	}
	return db.Ping()
}

func (c *Client) Close() error {
	db, err := c.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
