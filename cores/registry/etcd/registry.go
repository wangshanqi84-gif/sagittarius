package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Option func(o *options)

type options struct {
	ctx       context.Context
	namespace string
	product   string
	ttl       time.Duration
	maxRetry  int
}

func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

func Namespace(namespace string) Option {
	return func(o *options) { o.namespace = namespace }
}

func Product(product string) Option {
	return func(o *options) { o.product = product }
}

func MaxRetry(num int) Option {
	return func(o *options) { o.maxRetry = num }
}

type Registry struct {
	opts   *options
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

func NewDiscovery(client *clientv3.Client, opts ...Option) (r *Registry) {
	op := &options{
		ctx:      context.Background(),
		ttl:      _defaultLessIDTTL,
		maxRetry: _defaultRetryTimes,
	}
	for _, o := range opts {
		o(op)
	}
	return &Registry{
		opts:   op,
		client: client,
		kv:     clientv3.NewKV(client),
	}
}

// Register 服务注册
func (r *Registry) Register(ctx context.Context, service *registry.Service) error {
	// 根据服务名生成key
	key := fmt.Sprintf("/%s/%s/%s/%s", r.opts.namespace, r.opts.product,
		strings.Join(strings.Split(service.ServiceName, "."), "/"), service.ID)
	value, err := json.Marshal(service)
	if err != nil {
		return err
	}
	if r.lease != nil {
		r.lease.Close()
	}
	// 创建less端
	r.lease = clientv3.NewLease(r.client)
	// 执行注册
	leaseID, err := r.registerKV(ctx, key, string(value))
	if err != nil {
		return err
	}

	// 执行ttl心跳
	go r.doTTL(r.opts.ctx, leaseID, key, string(value))
	return nil
}

// 注册流程
func (r *Registry) registerKV(ctx context.Context, key string, value string) (clientv3.LeaseID, error) {
	// 有效期
	grant, err := r.lease.Grant(ctx, int64(r.opts.ttl.Seconds()))
	if err != nil {
		return 0, err
	}
	// 数据注册推送
	_, err = r.client.Put(ctx, key, value, clientv3.WithLease(grant.ID))
	if err != nil {
		return 0, err
	}
	return grant.ID, nil
}

// 定时续期
func (r *Registry) doTTL(ctx context.Context, leaseID clientv3.LeaseID, key string, value string) {
	// 初始化当前lessID
	curLeaseID := leaseID
	// 对当前lessID进行keepalive
	kac, err := r.client.KeepAlive(ctx, leaseID)
	if err != nil {
		curLeaseID = 0
	}

	for {
		// 是否需要重新注册
		if curLeaseID == 0 {
			for retryCnt := 0; retryCnt < r.opts.maxRetry; retryCnt++ {
				// 服务关闭检查
				if ctx.Err() != nil {
					return
				}
				idChan := make(chan clientv3.LeaseID, 1)
				errChan := make(chan error, 1)
				cCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				// 重新注册
				go func() {
					defer cancel()
					id, registerErr := r.registerKV(cCtx, key, value)
					if registerErr != nil {
						errChan <- registerErr
					} else {
						idChan <- id
					}
				}()
				// 等待结果
				select {
				case <-cCtx.Done():
					if cCtx.Err() != nil {
						if errors.Is(cCtx.Err(), context.Canceled) {
							break
						}
					}
					continue
				case <-errChan:
					continue
				case curLeaseID = <-idChan:
				}
				// 对新lessID进行keepalive监听
				kac, err = r.client.KeepAlive(ctx, curLeaseID)
				if err == nil {
					break
				}
				// 等待2^n秒重试
				time.Sleep(time.Duration(1<<retryCnt) * time.Second)
			}
			if _, ok := <-kac; !ok {
				// 注册重试失败则不再使用服务发现
				return
			}
		}

		select {
		case _, ok := <-kac:
			// keepalive错误处理
			if !ok {
				if ctx.Err() != nil {
					// 服务已关闭则不再进行注册
					return
				}
				// 当前lessID失效 准备retry
				curLeaseID = 0
				continue
			}
		case <-ctx.Done():
			// 服务关闭
			return
		}
	}
}

// Deregister 取消注册
func (r *Registry) Deregister(ctx context.Context, service *registry.Service) error {
	defer func() {
		if r.lease != nil {
			r.lease.Close()
		}
	}()
	key := fmt.Sprintf("%s/%s/%s/%s", r.opts.namespace, r.opts.product,
		strings.Join(strings.Split(service.ServiceName, "."), "/"), service.ID)
	_, err := r.client.Delete(ctx, key)
	return err
}

// Stop 关闭服务发现
func (r *Registry) Stop(ctx context.Context, service *registry.Service) error {
	defer func() {
		if r.lease != nil {
			r.lease.Close()
		}
		_ = r.client.Close()
	}()
	key := fmt.Sprintf("%s/%s/%s/%s", r.opts.namespace, r.opts.product,
		strings.Join(strings.Split(service.ServiceName, "."), "/"), service.ID)
	_, err := r.client.Delete(ctx, key)
	return err
}

// Watcher 获取watcher
func (r *Registry) Watcher(ctx context.Context, namespace string, product string, serviceName string, proto string) (registry.Watcher, error) {
	key := fmt.Sprintf("/%s/%s/%s", namespace, product,
		strings.Join(strings.Split(serviceName, "."), "/"))
	key = "/" + strings.TrimLeft(key, "/")
	return newWatcher(ctx, key, serviceName, r.client)
}
