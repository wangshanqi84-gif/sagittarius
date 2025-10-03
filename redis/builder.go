package redis

import (
	"time"

	redisgo "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type Builder func(c *Client)

var (
	builders = map[string]Builder{
		typeSingleton: buildSingleton,
		typeSentinel:  buildSentinel,
		typeCluster:   buildCluster,
	}
)

func buildSentinel(c *Client) {
	cmd := redisgo.NewFailoverClient(
		&redisgo.FailoverOptions{
			MasterName:    c.name + "_master",
			SentinelAddrs: c.addrs,
			DB:            c.db,
			MaxRetries:    c.retry,
			IdleTimeout:   time.Duration(c.idleTimeout) * time.Second,
			DialTimeout:   time.Duration(defaultDialTimeout) * time.Second,
			ReadTimeout:   time.Duration(c.readTimeout) * time.Second,
			WriteTimeout:  time.Duration(c.writeTimeout) * time.Second,
			PoolSize:      c.poolSize,
			MinIdleConns:  c.minIdleConn,
			Username:      c.username,
			Password:      c.password,
		})
	// 创建redsync实例
	c.rs = redsync.New(goredis.NewPool(cmd))
	c.IRedisCmd = cmd
}

func buildCluster(c *Client) {
	cmd := redisgo.NewClusterClient(&redisgo.ClusterOptions{
		Addrs:        c.addrs,
		MaxRetries:   c.retry,
		IdleTimeout:  time.Duration(c.idleTimeout) * time.Second,
		DialTimeout:  time.Duration(defaultDialTimeout) * time.Second,
		ReadTimeout:  time.Duration(c.readTimeout) * time.Second,
		WriteTimeout: time.Duration(c.writeTimeout) * time.Second,
		PoolSize:     c.poolSize,
		MinIdleConns: c.minIdleConn,
		Username:     c.username,
		Password:     c.password,
	})
	// 创建redsync实例
	c.rs = redsync.New(goredis.NewPool(cmd))
	c.IRedisCmd = cmd
}

func buildSingleton(c *Client) {
	cmd := redisgo.NewClient(&redisgo.Options{
		Addr:         c.addrs[0],
		DB:           c.db,
		MaxRetries:   c.retry,
		IdleTimeout:  time.Duration(c.idleTimeout) * time.Second,
		DialTimeout:  time.Duration(defaultDialTimeout) * time.Second,
		ReadTimeout:  time.Duration(c.readTimeout) * time.Second,
		WriteTimeout: time.Duration(c.writeTimeout) * time.Second,
		PoolSize:     c.poolSize,
		MinIdleConns: c.minIdleConn,
		Username:     c.username,
		Password:     c.password,
	})
	// 创建redsync实例
	c.rs = redsync.New(goredis.NewPool(cmd))
	c.IRedisCmd = cmd
}
