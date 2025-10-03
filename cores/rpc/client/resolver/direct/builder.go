package direct

import (
	"google.golang.org/grpc/resolver"
)

type Option func(o *builder)

func WithEps(eps ...string) Option {
	return func(b *builder) {
		b.eps = eps
	}
}

type builder struct {
	eps []string
}

func NewBuilder(opts ...Option) resolver.Builder {
	b := new(builder)
	for _, o := range opts {
		o(b)
	}
	return b
}

func (b *builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	addrs := make([]resolver.Address, 0)
	for _, ep := range b.eps {
		addrs = append(addrs, resolver.Address{Addr: ep})
	}
	err := cc.UpdateState(resolver.State{
		Addresses: addrs,
	})
	if err != nil {
		return nil, err
	}
	return &directResolver{}, nil
}

func (b *builder) Scheme() string {
	return "direct"
}
