package context

import (
	"context"
)

type (
	clientTransportKey struct{}
)

type TransData struct {
	Endpoint    string `json:"host"`
	Namespace   string `json:"namespace"`
	Product     string `json:"product"`
	ServiceName string `json:"serviceName"`
}

func FromClientContext(ctx context.Context) (TransData, bool) {
	td, ok := ctx.Value(clientTransportKey{}).(TransData)
	return td, ok
}
