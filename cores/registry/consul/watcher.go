package consul

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/hashicorp/consul/api"
)

type watcher struct {
	cli    *api.Client
	ctx    context.Context
	cancel context.CancelFunc

	index       uint64
	key         string
	serviceName string
	proto       string
}

func newWatcher(ctx context.Context, key string, sn string, proto string, client *api.Client) (*watcher, error) {
	w := &watcher{
		key:         key,
		cli:         client,
		serviceName: sn,
		proto:       proto,
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	return w, nil
}

// Start 开始监听
func (w *watcher) Start() ([]*registry.Service, error) {
	fn := func() ([]*registry.Service, error) {
		key := fmt.Sprintf("%s-%s", w.key, w.proto)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
		opts := &api.QueryOptions{
			WaitIndex: w.index,
			WaitTime:  time.Second * 30,
		}
		opts = opts.WithContext(ctx)
		entries, meta, err := w.cli.Health().Service(key, "", true, opts)
		cancel()
		if err != nil {
			log.Println(fmt.Sprintf("consul watcher health check error:%v", err))
			return nil, err
		}
		var srvs []*registry.Service
		if len(entries) != 0 && meta.LastIndex != w.index {
			for _, entry := range entries {
				srv := &registry.Service{
					ID:          entry.Service.ID,
					Namespace:   entry.Service.Meta["namespace"],
					Product:     entry.Service.Meta["product"],
					ServiceName: entry.Service.Meta["serviceName"],
					Tags:        strings.Join(entry.Service.Tags, ","),
				}
				if srv.ServiceName != w.serviceName {
					continue
				}
				if !strings.Contains(srv.Tags, env.GetRunEnv()) {
					continue
				}
				srv.Hosts = map[string]string{
					w.proto: fmt.Sprintf("%s:%d", entry.Service.Address, entry.Service.Port),
				}
				srvs = append(srvs, srv)
				log.Println(fmt.Sprintf("consul watcher srv:%+v", *srv))
			}
			w.index = meta.LastIndex
		}
		return srvs, nil
	}

	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	default:
		return fn()
	}
}

// Stop 停止监听
func (w *watcher) Stop() error {
	w.cancel()
	return nil
}
