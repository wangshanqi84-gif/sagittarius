package etcd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

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

func (cc *ConfigClient) GetConfig(name string, v interface{}) error {
	basePath := fmt.Sprintf("%s/%s", cc.namespace, env.GetRunEnv())
	key := basePath + "/" + strings.TrimLeft(name, "/")
	resp, err := cc.cli.Get(cc.ctx, key)
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 0 {
		return nil
	}
	switch cc.format {
	case "yaml":
		err = yaml.Unmarshal(resp.Kvs[0].Value, v)
	case "xml":
		err = xml.Unmarshal(resp.Kvs[0].Value, v)
	default:
		err = json.Unmarshal(resp.Kvs[0].Value, v)
	}
	// 监听处理
	go func() {
		watchChan := cc.cli.Watch(cc.ctx, key)
		for {
			select {
			case <-cc.ctx.Done():
				return
			case res := <-watchChan:
				for _, event := range res.Events {
					switch event.Type {
					case clientv3.EventTypePut:
						if key != string(event.Kv.Value) {
							continue
						}
						switch cc.format {
						case "yaml":
							err = yaml.Unmarshal(event.Kv.Value, v)
						case "xml":
							err = xml.Unmarshal(event.Kv.Value, v)
						default:
							err = json.Unmarshal(event.Kv.Value, v)
						}
						break
					}
				}
			}
		}
	}()
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
