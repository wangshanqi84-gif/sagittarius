package etcd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

type ConfigClient struct {
	ctx       context.Context
	cli       *clientv3.Client
	format    string
	namespace string
	product   string
	name      string
	cfgValue  string
	mu        sync.RWMutex
}

func NewConfigClient(ctx context.Context, namespace string, product string,
	name string, format string, options ...etcd.Option) *ConfigClient {
	return &ConfigClient{
		ctx:       ctx,
		cli:       etcd.NewEtcdClient(options...),
		format:    format,
		namespace: namespace,
		product:   product,
		name:      name,
	}
}

func (cc *ConfigClient) LoadConfig(key string) error {
	basePath := fmt.Sprintf("%s/%s", cc.namespace, env.GetRunEnv())
	fullKey := basePath + "/" + strings.TrimLeft(key, "/")
	resp, err := cc.cli.Get(cc.ctx, fullKey)
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 0 {
		return nil
	}
	cc.mu.Lock()
	cc.cfgValue = string(resp.Kvs[0].Value)
	cc.mu.Unlock()

	watchChan := cc.cli.Watch(cc.ctx, fullKey)
	go func() {
		for {
			select {
			case <-cc.ctx.Done():
				return
			case res := <-watchChan:
				if res.Err() != nil {
					break
				}
				for _, event := range res.Events {
					switch event.Type {
					case clientv3.EventTypePut:
						cc.mu.Lock()
						cc.cfgValue = string(resp.Kvs[0].Value)
						cc.mu.Unlock()
					case clientv3.EventTypeDelete:
						cc.mu.Lock()
						cc.cfgValue = ""
						cc.mu.Unlock()
					}
				}
			}
		}
	}()
	return nil
}

func (cc *ConfigClient) GetConfig(v interface{}) error {
	if cc.cfgValue == "" {
		return errors.New("config value is empty")
	}
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	err := cc.unmarshal(v)
	if err != nil {
		return err
	}
	return nil
}

func (cc *ConfigClient) PublishConfig(name string, v interface{}) error {
	if cc.cli == nil {
		return nil
	}
	basePath := fmt.Sprintf("%s/%s", cc.namespace, env.GetRunEnv())
	key := basePath + "/" + strings.TrimLeft(name, "/")

	var bs []byte
	var err error
	switch cc.format {
	case "yaml":
		bs, err = yaml.Marshal(v)
	case "xml":
		bs, err = xml.Marshal(v)
	default:
		bs, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	_, err = cc.cli.Put(cc.ctx, key, string(bs))
	return err
}

func (cc *ConfigClient) unmarshal(v interface{}) error {
	var err error
	switch cc.format {
	case "yaml":
		err = yaml.Unmarshal([]byte(cc.cfgValue), v)
	case "xml":
		err = xml.Unmarshal([]byte(cc.cfgValue), v)
	default:
		err = json.Unmarshal([]byte(cc.cfgValue), v)
	}
	return err
}
