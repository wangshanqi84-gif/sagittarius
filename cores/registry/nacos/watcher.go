package nacos

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type watcher struct {
	cli naming_client.INamingClient

	key         string
	serviceName string
	namespace   string
	proto       string
	ctx         context.Context
	cancel      context.CancelFunc
}

func newWatcher(ctx context.Context, key string, sn string, namespace string, proto string, cli naming_client.INamingClient) (*watcher, error) {
	w := &watcher{
		key:         key,
		cli:         cli,
		serviceName: sn,
		namespace:   namespace,
		proto:       proto,
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	return w, nil
}

// Start 开始监听key变化
func (w *watcher) Start() ([]*registry.Service, error) {
	fn := func() ([]*registry.Service, error) {
		key := fmt.Sprintf("%s-%s", w.key, w.proto)
		instances, err := w.cli.SelectAllInstances(vo.SelectAllInstancesParam{
			Clusters:    []string{w.namespace},
			ServiceName: key,
			GroupName:   env.GetRunEnv(),
		})
		if err != nil {
			log.Println(fmt.Sprintf("nacos watcher select instances error:%v", err))
			return nil, err
		}
		var srvs []*registry.Service
		if len(instances) != 0 {
			for _, ins := range instances {
				if !ins.Healthy || !ins.Enable || ins.Weight <= 0 {
					continue
				}
				srv := &registry.Service{
					ID:          ins.Metadata["serviceId"],
					Namespace:   ins.Metadata["namespace"],
					Product:     ins.Metadata["product"],
					ServiceName: ins.Metadata["serviceName"],
					Tags:        ins.Metadata["tags"],
				}
				if srv.ServiceName != w.serviceName {
					continue
				}
				if !strings.Contains(srv.Tags, env.GetRunEnv()) {
					continue
				}
				srv.Hosts = map[string]string{
					w.proto: fmt.Sprintf("%s:%d", ins.Ip, ins.Port),
				}
				srvs = append(srvs, srv)
				log.Println(fmt.Sprintf("consul watcher srv:%+v", *srv))
			}
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
