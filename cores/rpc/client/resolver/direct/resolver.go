package direct

import (
	"google.golang.org/grpc/resolver"
)

type directResolver struct{}

func (r *directResolver) Close() {
}

func (r *directResolver) ResolveNow(options resolver.ResolveNowOptions) {
}
