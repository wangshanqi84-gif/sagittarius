package balancer

import (
	"context"

	"github.com/wangshanqi84-gif/sagittarius/cores/registry"
)

type Balancer interface {
	Pick(context.Context) (*registry.Service, error)
	Update(context.Context, []*registry.Service)
}

type Builder interface {
	Build() Balancer
}
