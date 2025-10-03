package consul

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/hashicorp/consul/api"
)

type Option func(o *options)

type options struct {
	ctx context.Context
}

func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

type Registry struct {
	opts *options
	cli  *api.Client
}

func NewDiscovery(client *api.Client, opts ...Option) (r *Registry) {
	op := &options{
		ctx: context.Background(),
	}
	for _, o := range opts {
		o(op)
	}
	return &Registry{
		opts: op,
		cli:  client,
	}
}

// Register 服务注册
func (r *Registry) Register(_ context.Context, srv *registry.Service) error {
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

func (r *Registry) Deregister(_ context.Context, srv *registry.Service) error {
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
	return r.Deregister(ctx, service)
}

// Watcher 获取watcher
func (r *Registry) Watcher(ctx context.Context, namespace string, product string, serviceName string, proto string) (registry.Watcher, error) {
	key := strings.TrimLeft(fmt.Sprintf("%s.%s.%s", namespace, product, serviceName), ".")
	return newWatcher(ctx, key, serviceName, proto, r.cli)
}
