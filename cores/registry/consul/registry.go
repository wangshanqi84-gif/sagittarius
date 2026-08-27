package consul

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/hashicorp/consul/api"
)

type Option func(o *options)

type options struct {
	ctx           context.Context
	retryTimes    int
	retryInterval time.Duration
}

// Retry 设置重试参数
func Retry(times int, interval time.Duration) Option {
	return func(o *options) {
		o.retryTimes = times
		o.retryInterval = interval
	}
}

func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

type Registry struct {
	opts   *options
	cli    *api.Client
	mu     sync.RWMutex
	srv    *registry.Service
	stopCh chan struct{}
	once   sync.Once
}

func NewDiscovery(client *api.Client, opts ...Option) (r *Registry) {
	op := &options{
		ctx:           context.Background(),
		retryTimes:    3,
		retryInterval: time.Second * 2,
	}
	for _, o := range opts {
		o(op)
	}
	return &Registry{
		opts:   op,
		cli:    client,
		stopCh: make(chan struct{}),
	}
}

// Register 服务注册
func (r *Registry) Register(_ context.Context, srv *registry.Service) error {
	r.mu.Lock()
	r.srv = srv
	r.mu.Unlock()

	var err error
	for i := 0; i <= r.opts.retryTimes; i++ {
		err = r.do(srv)
		if err != nil {
			if i < r.opts.retryTimes {
				// 指数退避
				time.Sleep(r.opts.retryInterval * time.Duration(i+1))
			}
		} else {
			break
		}
	}
	if err == nil {
		go r.ttl()
	}
	return err
}

// 服务注册
func (r *Registry) do(srv *registry.Service) error {
	key := fmt.Sprintf("%s.%s.%s", srv.Namespace, srv.Product, srv.ServiceName)
	// 多次注册
	for proto, host := range srv.Hosts {
		asrHost := host
		if strings.Index(asrHost, "://") < 0 {
			asrHost = fmt.Sprintf("discovery://%s", asrHost)
		}
		raw, err := url.Parse(asrHost)
		if err != nil {
			return err
		}
		port, err := strconv.ParseUint(raw.Port(), 10, 16)
		if err != nil {
			return err
		}
		asr := &api.AgentServiceRegistration{
			ID:      fmt.Sprintf("%s-%s", srv.ID, proto),
			Name:    fmt.Sprintf("%s-%s", key, proto),
			Address: raw.Hostname(),
			Port:    int(port),
			Meta: map[string]string{
				"namespace":   srv.Namespace,
				"product":     srv.Product,
				"serviceName": srv.ServiceName,
			},
			Tags: strings.Split(srv.Tags, ","),
			Check: &api.AgentServiceCheck{
				TCP:                            host,
				Interval:                       fmt.Sprintf("%ds", 10),
				DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", 300),
				Timeout:                        "5s",
			},
		}
		if err = r.cli.Agent().ServiceRegister(asr); err != nil {
			return err
		}
	}
	return nil
}

// 定期检查
func (r *Registry) ttl() {
	// 每30秒检查一次
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	checkRegistryFn := func(srv *registry.Service) bool {
		for proto := range srv.Hosts {
			id := fmt.Sprintf("%s-%s", srv.ID, proto)
			if _, _, err := r.cli.Agent().Service(id, nil); err != nil {
				return false
			}
		}
		return true
	}
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.RLock()
			srv := r.srv
			r.mu.RUnlock()

			if srv == nil {
				continue
			}
			if !checkRegistryFn(srv) {
				var err error
				for i := 0; i <= r.opts.retryTimes; i++ {
					err = r.do(srv)
					if err != nil {
						if i < r.opts.retryTimes {
							// 指数退避
							time.Sleep(r.opts.retryInterval * time.Duration(i+1))
						}
					} else {
						break
					}
				}
				if err != nil {
					log.Println("re-register service failed:", srv.ServiceName, "err:", err)
				}
			}
		}
	}
}

func (r *Registry) Deregister(_ context.Context, srv *registry.Service) error {
	r.mu.Lock()
	r.srv = nil
	r.mu.Unlock()

	var errors []error
	for proto, _ := range srv.Hosts {
		id := fmt.Sprintf("%s-%s", srv.ID, proto)
		if err := r.cli.Agent().ServiceDeregister(id); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

// Stop 关闭服务发现
func (r *Registry) Stop(ctx context.Context, service *registry.Service) error {
	r.once.Do(func() {
		close(r.stopCh)
	})
	return r.Deregister(ctx, service)
}

// Watcher 获取watcher
func (r *Registry) Watcher(ctx context.Context, namespace string, product string, serviceName string, proto string) (registry.Watcher, error) {
	key := strings.TrimLeft(fmt.Sprintf("%s.%s.%s", namespace, product, serviceName), ".")
	return newWatcher(ctx, key, serviceName, proto, r.cli)
}
