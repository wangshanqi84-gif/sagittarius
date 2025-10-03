package random

import (
	"context"
	"math/rand"

	"github.com/wangshanqi84-gif/sagittarius/cores/http/client/balancer"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/pkg/errors"
)

var ErrNoAvailable = errors.New("no_available_node")

type Option func(o *options)

type options struct{}

type Balancer struct {
	nodes []*registry.Service
}

func (b *Balancer) Pick(_ context.Context) (*registry.Service, error) {
	if len(b.nodes) == 0 {
		return nil, ErrNoAvailable
	}
	cur := rand.Intn(len(b.nodes))
	return b.nodes[cur], nil
}

func (b *Balancer) Update(_ context.Context, service []*registry.Service) {
	if len(service) == 0 {
		return
	}
	b.nodes = service
}

type Builder struct{}

func (b *Builder) Build() balancer.Balancer {
	return &Balancer{}
}

func NewBuilder(opts ...Option) balancer.Builder {
	var option options
	for _, opt := range opts {
		opt(&option)
	}
	return &Builder{}
}
