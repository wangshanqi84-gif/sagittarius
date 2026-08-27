package nacos

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
)

type Option func(o *options)

type options struct {
	ctx           context.Context
	namespace     string
	product       string
	retryTimes    int
	retryInterval time.Duration
}

func Namespace(namespace string) Option {
	return func(o *options) { o.namespace = namespace }
}

func Product(product string) Option {
	return func(o *options) { o.product = product }
}

func Retry(times int, interval time.Duration) Option {
	return func(o *options) {
		o.retryTimes = times
		o.retryInterval = interval
	}
}

type Registry struct {
	opts   *options
	cli    naming_client.INamingClient
	mu     sync.RWMutex
	srv    *registry.Service
	stopCh chan struct{}
	once   sync.Once
}

func NewDiscovery(cli naming_client.INamingClient, opts ...Option) (r *Registry) {
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
		cli:    cli,
		stopCh: make(chan struct{}),
	}
}

// Register 服务注册
func (r *Registry) Register(ctx context.Context, srv *registry.Service) error {
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

func (r *Registry) do(srv *registry.Service) error {
	// 根据服务名生成key
	key := fmt.Sprintf("/%s/%s/%s/%s", r.opts.namespace, r.opts.product,
		strings.Join(strings.Split(srv.ServiceName, "."), "/"), srv.ID)
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
		meta := make(map[string]string)
		for k, v := range srv.Metadata {
			meta[k] = v
		}
		meta["serviceId"] = srv.ID
		meta["proto"] = proto
		meta["namespace"] = srv.Namespace
		meta["product"] = srv.Product
		meta["serviceName"] = srv.ServiceName
		meta["tags"] = srv.Tags
		ok, err := r.cli.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          raw.Hostname(),
			Port:        port,
			ClusterName: r.opts.namespace,
			ServiceName: fmt.Sprintf("%s-%s", key, proto),
			GroupName:   env.GetRunEnv(),
			Weight:      1.0,
			Enable:      true,
			Healthy:     true,
			Metadata:    meta,
		})
		if err != nil {
			return err
		}
		if !ok {
			return errors.New(fmt.Sprintf("registry server failed, key:%v", key))
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
		key := fmt.Sprintf("/%s/%s/%s/%s", r.opts.namespace, r.opts.product,
			strings.Join(strings.Split(srv.ServiceName, "."), "/"), srv.ID)
		for proto := range srv.Hosts {
			instances, err := r.cli.SelectInstances(vo.SelectInstancesParam{
				Clusters:    []string{r.opts.namespace},
				ServiceName: fmt.Sprintf("%s-%s", key, proto),
				GroupName:   env.GetRunEnv(),
				HealthyOnly: false,
			})
			if err != nil {
				return false
			}
			// 检查是否有匹配的实例
			found := false
			for _, ins := range instances {
				if ins.Metadata != nil && ins.Metadata["serviceId"] == srv.ID {
					found = true
					break
				}
			}
			if !found {
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
	// 从缓存中删除
	r.mu.Lock()
	r.srv = nil
	r.mu.Unlock()

	// 根据服务名生成key
	key := fmt.Sprintf("/%s/%s/%s/%s", r.opts.namespace, r.opts.product,
		strings.Join(strings.Split(srv.ServiceName, "."), "/"), srv.ID)
	var errs []error
	for proto, host := range srv.Hosts {
		asrHost := host
		if strings.Index(asrHost, "://") < 0 {
			asrHost = fmt.Sprintf("discovery://%s", asrHost)
		}
		raw, err := url.Parse(asrHost)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		port, err := strconv.ParseUint(raw.Port(), 10, 16)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ok, err := r.cli.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          raw.Hostname(),
			Port:        port,
			Cluster:     r.opts.namespace,
			ServiceName: fmt.Sprintf("%s-%s", key, proto),
			GroupName:   env.GetRunEnv(),
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !ok {
			errs = append(errs, errors.New(fmt.Sprintf("registry server failed, key:%v", key)))
		}
	}
	if len(errs) > 0 {
		return errs[0]
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
	key := fmt.Sprintf("/%s/%s/%s", namespace, product,
		strings.Join(strings.Split(serviceName, "."), "/"))
	key = "/" + strings.TrimLeft(key, "/")
	return newWatcher(ctx, key, serviceName, namespace, proto, r.cli)
}
