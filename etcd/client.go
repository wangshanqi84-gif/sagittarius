package etcd

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type Option func(*option)

type option struct {
	dialTimeout string
	username    string
	password    string
	eps         []string
}

func DialTimeout(dialTimeout string) Option {
	return func(o *option) {
		o.dialTimeout = dialTimeout
	}
}

func Username(username string) Option {
	return func(o *option) {
		o.username = username
	}
}

func Password(password string) Option {
	return func(o *option) {
		o.password = password
	}
}

func Endpoints(eps []string) Option {
	return func(o *option) {
		o.eps = eps
	}
}

func NewEtcdClient(opts ...Option) *clientv3.Client {
	o := option{
		dialTimeout: "10s",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	td, err := time.ParseDuration(o.dialTimeout)
	if err != nil {
		panic(err)
	}
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   o.eps,
		DialTimeout: td,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		panic(err)
	}
	return c
}
