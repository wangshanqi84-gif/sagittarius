package consul

import (
	"fmt"
	"net/http"
	"strings"
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

func NewConsulClient(addrs string, opts ...Option) *api.Client {
	o := option{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	var cli *api.Client
	var err error
	ss := strings.Split(addrs, ",")
	for _, addr := range ss {
		addr = strings.TrimRight(addr, "/")
		config := &api.Config{
			Address: addr,
		}
		if o.transport != nil {
			config.Transport = o.transport
		}
		if o.waitTime > 0 {
			config.WaitTime = o.waitTime
		}
		cli, err = api.NewClient(config)
		if err == nil {
			break
		}
	}
	if cli == nil {
		panic(fmt.Sprintf("init consul addrs:%v, failed", addrs))
	}
	return cli
}
