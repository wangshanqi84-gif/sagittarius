package nacos

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sync"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/nacos"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type ConfigClient struct {
	ctx       context.Context
	cli       config_client.IConfigClient
	changeCh  chan string
	format    string
	namespace string
	product   string
	name      string
	cfgValue  string
	mu        sync.RWMutex
}

func NewConfigClient(ctx context.Context, namespace string, product string,
	name string, format string, options ...nacos.Option) *ConfigClient {
	return &ConfigClient{
		ctx:       ctx,
		cli:       nacos.NewClient(namespace, product, name, options...),
		changeCh:  make(chan string),
		format:    format,
		namespace: namespace,
		product:   product,
		name:      name,
	}
}

func (cc *ConfigClient) LoadConfig(key string) error {
	if cc.cli == nil {
		return nil
	}
	param := vo.ConfigParam{
		DataId: key,
		Group:  env.GetRunEnv(),
	}
	switch cc.format {
	case "yaml":
		param.Type = vo.YAML
	case "xml":
		param.Type = vo.XML
	default:
		param.Type = vo.JSON
	}
	vs, err := cc.cli.GetConfig(param)
	if err != nil {
		return err
	}
	if len(vs) == 0 {
		return errors.New(fmt.Sprintf("config file does not exist, key:%s, env:%s",
			key, env.GetRunEnv()))
	}
	param.OnChange = func(namespace, group, dataId, data string) {
		cc.changeCh <- data
	}
	if err = cc.cli.ListenConfig(param); err != nil {
		return err
	}
	cc.mu.Lock()
	cc.cfgValue = vs
	cc.mu.Unlock()

	go func() {
		for {
			select {
			case <-cc.ctx.Done():
				return
			case s := <-cc.changeCh:
				cc.mu.Lock()
				cc.cfgValue = s
				cc.mu.Unlock()
			}
		}
	}()
	return err
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

func (cc *ConfigClient) PublishConfig(key string, v interface{}) error {
	if cc.cli == nil {
		return nil
	}
	var bs []byte
	var err error
	var cType vo.ConfigType
	switch cc.format {
	case "yaml":
		bs, err = yaml.Marshal(v)
		cType = vo.YAML
	case "xml":
		bs, err = xml.Marshal(v)
		cType = vo.XML
	default:
		bs, err = json.Marshal(v)
		cType = vo.JSON
	}
	if err != nil {
		return err
	}
	_, err = cc.cli.PublishConfig(vo.ConfigParam{
		DataId:  key,
		Group:   env.GetRunEnv(),
		Type:    cType,
		Content: string(bs),
	})
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
