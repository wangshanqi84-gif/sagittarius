package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/http/client/balancer"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/pkg/errors"
)

type resolver struct {
	eps       []string
	watcher   registry.Watcher
	balancer  balancer.Balancer
	insecure  bool
	firstChan chan struct{}
}

func newResolver(ctx context.Context, watcher registry.Watcher, balanceBuilder balancer.Builder, eps []string, insecure bool) (*resolver, error) {
	r := &resolver{
		watcher:   watcher,
		balancer:  balanceBuilder.Build(),
		eps:       eps,
		insecure:  insecure,
		firstChan: make(chan struct{}),
	}
	isFirst := true
	go func() {
		if r.watcher == nil {
			var services []*registry.Service
			for _, ep := range r.eps {
				services = append(services, &registry.Service{
					Hosts: map[string]string{"http": ep},
				})
			}
			r.balancer.Update(ctx, services)
			isFirst = false
			r.firstChan <- struct{}{}
		} else {
			for {
				services, err := r.watcher.Start()
				if err != nil {
					log.Println(fmt.Sprintf("resolver watcher err:%v", err))
					if errors.Is(err, context.Canceled) {
						return
					}
					time.Sleep(time.Second)
					continue
				}
				if len(services) == 0 && len(r.eps) != 0 {
					for _, ep := range r.eps {
						services = append(services, &registry.Service{
							Hosts: map[string]string{"http": ep},
						})
					}
				}
				if len(services) > 0 {
					r.balancer.Update(ctx, services)
				}
				if isFirst {
					isFirst = false
					r.firstChan <- struct{}{}
				}
				time.Sleep(time.Second)
			}
		}
	}()
	<-r.firstChan
	return r, nil
}

func (r *resolver) Close() error {
	log.Println("resolver closed.")
	return r.watcher.Stop()
}
