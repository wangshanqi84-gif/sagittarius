package discovery

import (
	"context"
	"errors"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

// 服务发现resolver

type discoveryResolver struct {
	ctx       context.Context
	cancel    context.CancelFunc
	watcher   registry.Watcher
	cc        resolver.ClientConn
	eps       []resolver.Address
	firstChan chan struct{}
}

// 监控watch
func (r *discoveryResolver) watch() {
	isFirst := true
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		srvs, err := r.watcher.Start()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			time.Sleep(time.Second)
			continue
		}
		r.updateCC(srvs)
		if isFirst {
			isFirst = false
			r.firstChan <- struct{}{}
		}
	}
}

// 更新client conn
func (r *discoveryResolver) updateCC(srvs []*registry.Service) {
	addrs := make([]resolver.Address, 0)
	for _, srv := range srvs {
		if _, has := srv.Hosts["rpc"]; !has {
			continue
		}
		addr := resolver.Address{
			ServerName: srv.ServiceName,
			Attributes: parseAttributes(srv.Metadata),
			Addr:       srv.Hosts["rpc"],
		}
		addrs = append(addrs, addr)
	}
	// 如果服务发现失败且有兜底配置 则改为使用兜底配置
	if len(addrs) == 0 {
		if len(r.eps) == 0 {
			return
		}
		addrs = r.eps
	}
	_ = r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *discoveryResolver) Close() {
	r.cancel()
	_ = r.watcher.Stop()
}

func (r *discoveryResolver) ResolveNow(options resolver.ResolveNowOptions) {}

// 解析扩展信息
func parseAttributes(md map[string]string) *attributes.Attributes {
	var a *attributes.Attributes
	for k, v := range md {
		if a == nil {
			a = attributes.New(k, v)
		} else {
			a = a.WithValue(k, v)
		}
	}
	return a
}
