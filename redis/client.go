package redis

import (
	"context"
	"time"

	redisgo "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/pkg/errors"
)

type Option func(c *Client)

// Model
// 设置redis模式
func Model(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// DB
// 设置redis Node
func DB(db int) Option {
	return func(c *Client) {
		c.db = db
	}
}

// Retry
// 设置redis重试次数
func Retry(retry int) Option {
	return func(c *Client) {
		c.retry = retry
	}
}

// IdleTimeout
// 设置redis链接空闲
func IdleTimeout(idleTimeout int) Option {
	return func(c *Client) {
		c.idleTimeout = idleTimeout
	}
}

// ReadTimeout
// 设置redis读超时
func ReadTimeout(readTimeout int) Option {
	return func(c *Client) {
		c.readTimeout = readTimeout
	}
}

// WriteTimeout
// 设置redis写超时
func WriteTimeout(writeTimeout int) Option {
	return func(c *Client) {
		c.writeTimeout = writeTimeout
	}
}

// PoolSize
// 连接池大小
func PoolSize(poolSize int) Option {
	return func(c *Client) {
		c.poolSize = poolSize
	}
}

// MinIdleConn
// 最小空闲连接数
func MinIdleConn(minIdleConn int) Option {
	return func(c *Client) {
		c.minIdleConn = minIdleConn
	}
}

// Name
// 定义名称
func Name(name string) Option {
	return func(c *Client) {
		c.name = name
	}
}

// Addrs
// 链接地址
func Addrs(addrs []string) Option {
	return func(c *Client) {
		c.addrs = addrs
	}
}

// Username
// 账户
func Username(username string) Option {
	return func(c *Client) {
		c.username = username
	}
}

// Password
// 密码
func Password(password string) Option {
	return func(c *Client) {
		c.password = password
	}
}

type IRedisCmd interface {
	redisgo.Cmdable
	Subscribe(ctx context.Context, channels ...string) *redisgo.PubSub
}

type Mutex struct {
	*redsync.Mutex
	expire time.Duration
	extCh  chan struct{}
	name   string
}

func (m *Mutex) TryLock(ctx context.Context) (bool, error) {
	if err := m.TryLockContext(ctx); err != nil {
		if err == redsync.ErrFailed {
			return false, nil
		}
		return false, err
	}
	if m.extCh != nil {
		go func() {
			extendMax, count := 3, 0
			for {
				timer := time.NewTimer(m.expire / 3 * 2)
				select {
				case <-m.extCh:
					return
				case <-timer.C:
					timer.Stop()
					ok, err := m.ExtendContext(ctx)
					if !ok {
						return
					}
					if err != nil {
						return
					}
					if count > extendMax {
						return
					}
					count++
				}
			}
		}()
	}
	return true, nil
}

func (m *Mutex) UnLock(ctx context.Context) (bool, error) {
	if m.extCh != nil {
		m.extCh <- struct{}{}
	}
	return m.UnlockContext(ctx)
}

type Client struct {
	IRedisCmd
	rs *redsync.Redsync

	name         string
	addrs        []string
	model        string
	db           int
	retry        int
	idleTimeout  int
	readTimeout  int
	writeTimeout int
	poolSize     int
	minIdleConn  int
	username     string
	password     string
}

func NewClient(opts ...Option) (*Client, error) {
	c := Client{
		model:        typeSingleton,
		retry:        defaultRetry,
		idleTimeout:  defaultIdleTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		poolSize:     defaultPoolSize,
		minIdleConn:  defaultMinIdleConn,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&c)
		}
	}
	if len(c.addrs) == 0 {
		return nil, errors.New("server addr is empty")
	}
	if _, has := builders[c.model]; !has {
		return nil, errors.New("redis model not support")
	}
	builders[c.model](&c)
	return &c, nil
}

func (c *Client) NewMutex(name string, expired time.Duration) *Mutex {
	var opts []redsync.Option
	if expired == 0 {
		expired = time.Second * 8
	}
	opts = append(opts, redsync.WithExpiry(expired))
	m := c.rs.NewMutex(name, opts...)
	return &Mutex{
		name:   name,
		expire: expired,
		Mutex:  m,
	}
}

func (c *Client) NewMutexWithExtend(name string, expired time.Duration) *Mutex {
	var opts []redsync.Option
	if expired == 0 {
		expired = time.Second * 8
	}
	opts = append(opts, redsync.WithExpiry(expired))
	m := c.rs.NewMutex(name, opts...)
	return &Mutex{
		name:   name,
		expire: expired,
		Mutex:  m,
		extCh:  make(chan struct{}),
	}
}
