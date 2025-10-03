package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type watcher struct {
	key         string
	count       int64
	serviceName string
	watchChan   clientv3.WatchChan
	watcher     clientv3.Watcher
	kv          clientv3.KV
	ctx         context.Context
	cancel      context.CancelFunc
}

func newWatcher(ctx context.Context, key string, sn string, client *clientv3.Client) (*watcher, error) {
	w := &watcher{
		key:         key,
		serviceName: sn,
		count:       0,
		watcher:     clientv3.NewWatcher(client),
		kv:          clientv3.NewKV(client),
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.watchChan = w.watcher.Watch(w.ctx, key, clientv3.WithPrefix(), clientv3.WithRev(0), clientv3.WithKeysOnly())
	err := w.watcher.RequestProgress(context.Background())
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Start 开始监听key变化
func (w *watcher) Start() ([]*registry.Service, error) {
	fn := func() ([]*registry.Service, error) {
		resp, err := w.kv.Get(w.ctx, w.key, clientv3.WithPrefix())
		if err != nil {
			log.Println(fmt.Sprintf("etcd watcher health check error:%v", err))
			return nil, err
		}
		items := make([]*registry.Service, 0, len(resp.Kvs))
		for _, kv := range resp.Kvs {
			var svc registry.Service
			if err = json.Unmarshal(kv.Value, &svc); err != nil {
				return nil, err
			}
			if svc.ServiceName != w.serviceName {
				continue
			}
			if !strings.Contains(svc.Tags, env.GetRunEnv()) {
				continue
			}
			items = append(items, &svc)
		}
		return items, nil
	}

	defer func() {
		w.count++
	}()

	if w.count == 0 {
		item, err := fn()
		return item, err
	}

	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-w.watchChan:
		return fn()
	}
}

// Stop 停止监听
func (w *watcher) Stop() error {
	w.cancel()
	return w.watcher.Close()
}
