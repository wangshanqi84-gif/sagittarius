package db

import (
	"context"
	"database/sql"
	"errors"
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
// 设置空闲连接池中连接的最大数量
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
	sqlDB       *sql.DB
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
	if c.driverName != "" {
		if c.driverName != "db" && c.driverName != "postgres" {
			return nil, errors.New(`driverName must be "db" or "postgres"`)
		}
	}
	// 链接
	sqlDB, err := sql.Open(c.driverName, master)
	if err != nil {
		return nil, err
	}
	c.sqlDB = sqlDB
	// 设置打开数据库连接的最大数量
	if c.maxOpen == 0 {
		if c.driverName == "postgres" {
			c.maxOpen = DefaultPostgresMaxOpen
		} else {
			c.maxOpen = DefaultMysqlMaxOpen
		}
	}
	c.sqlDB.SetMaxOpenConns(c.maxOpen)
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
	// 读写分离配置
	if len(slave) > 0 {
		var dail []gorm.Dialector
		for _, dns := range slave {
			dail = append(dail, c.createDial(dns))
		}
		resolver := dbresolver.Register(dbresolver.Config{
			Replicas: dail,
		})
		c.resolver = resolver
	}
	if err = c.open(); err != nil {
		return nil, err
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

func (c *Client) open() error {
	if c.sqlDB == nil {
		return errors.New("init db client error, no connection")
	}
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
		db, err = gorm.Open(postgres.New(postgres.Config{
			Conn: c.sqlDB,
		}), c.gormCfg)
	default:
		db, err = gorm.Open(mysql.New(mysql.Config{
			Conn: c.sqlDB,
		}), c.gormCfg)
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

func (c *Client) DB(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx)
}

func (c *Client) Master(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx).Clauses(dbresolver.Write)
}

func (c *Client) Ping() error {
	return c.sqlDB.Ping()
}

func (c *Client) Close() error {
	return c.sqlDB.Close()
}
