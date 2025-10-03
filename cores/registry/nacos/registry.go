package nacos

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
)

type Option func(o *options)

type options struct {
	ctx       context.Context
	namespace string
	product   string
}

func Namespace(namespace string) Option {
	return func(o *options) { o.namespace = namespace }
}

func Product(product string) Option {
	return func(o *options) { o.product = product }
}

type Registry struct {
	opts *options
	cli  naming_client.INamingClient
}

func NewDiscovery(cli naming_client.INamingClient, opts ...Option) (r *Registry) {
	op := &options{
		ctx: context.Background(),
	}
	for _, o := range opts {
		o(op)
	}
	return &Registry{
		opts: op,
		cli:  cli,
	}
}

// Register 服务注册
func (r *Registry) Register(_ context.Context, srv *registry.Service) error {
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

func (r *Registry) Deregister(_ context.Context, srv *registry.Service) error {
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
	return r.Deregister(ctx, service)
}

// Watcher 获取watcher
func (r *Registry) Watcher(ctx context.Context, namespace string, product string, serviceName string, proto string) (registry.Watcher, error) {
	key := fmt.Sprintf("/%s/%s/%s", namespace, product,
		strings.Join(strings.Split(serviceName, "."), "/"))
	key = "/" + strings.TrimLeft(key, "/")
	return newWatcher(ctx, key, serviceName, namespace, proto, r.cli)
}
