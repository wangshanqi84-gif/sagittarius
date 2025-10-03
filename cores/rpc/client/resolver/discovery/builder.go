package discovery

import (
	"context"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"google.golang.org/grpc/resolver"
)

const name = "discovery"

type Option func(o *builder)

// WithEps 兜底endpoints
func WithEps(eps ...string) Option {
	return func(b *builder) {
		b.eps = eps
	}
}

type builder struct {
	watcher registry.Watcher
	eps     []string
}

func NewBuilder(watcher registry.Watcher, opts ...Option) resolver.Builder {
	b := &builder{
		watcher: watcher,
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

func (b *builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &discoveryResolver{
		watcher:   b.watcher,
		cc:        cc,
		ctx:       ctx,
		cancel:    cancel,
		firstChan: make(chan struct{}),
	}
	for _, ep := range b.eps {
		addr := resolver.Address{
			Addr: ep,
		}
		r.eps = append(r.eps, addr)
	}
	go r.watch()
	<-r.firstChan
	return r, nil
}

func (*builder) Scheme() string {
	return name
}
