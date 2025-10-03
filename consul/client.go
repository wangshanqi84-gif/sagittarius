package consul

import (
	"net/http"
	"time"

	"github.com/hashicorp/consul/api"
)

type Option func(*option)

type option struct {
	transport *http.Transport
	waitTime  time.Duration
}

func Transport(transport *http.Transport) Option {
	return func(o *option) {
		o.transport = transport
	}
}

func WaitTime(waitTime time.Duration) Option {
	return func(o *option) {
		o.waitTime = waitTime
	}
}

func NewConsulClient(addr string, opts ...Option) *api.Client {
	o := option{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	config := &api.Config{
		Address: addr,
	}
	if o.transport != nil {
		config.Transport = o.transport
	}
	if o.waitTime > 0 {
		config.WaitTime = o.waitTime
	}
	cli, err := api.NewClient(config)
	if err != nil {
		panic(err)
	}
	return cli
}
