package nacos

import (
	"context"
	"encoding/json"
	"encoding/xml"

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

func (cc *ConfigClient) GetConfig(name string, v interface{}) error {
	if cc.cli == nil {
		return nil
	}
	param := vo.ConfigParam{
		DataId: name,
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
		return errors.New("config file does not exist")
	}
	param.OnChange = func(namespace, group, dataId, data string) {
		cc.changeCh <- data
	}
	if err = cc.cli.ListenConfig(param); err != nil {
		return err
	}
	switch cc.format {
	case "yaml":
		err = yaml.Unmarshal([]byte(vs), v)
	case "xml":
		err = xml.Unmarshal([]byte(vs), v)
	default:
		err = json.Unmarshal([]byte(vs), v)
	}
	go func() {
		for {
			select {
			case <-cc.ctx.Done():
				return
			case s := <-cc.changeCh:
				switch cc.format {
				case "yaml":
					err = yaml.Unmarshal([]byte(s), v)
				case "xml":
					err = xml.Unmarshal([]byte(s), v)
				default:
					err = json.Unmarshal([]byte(s), v)
				}
			}
		}
	}()
	return err
}

func (cc *ConfigClient) PublishConfig(name string, v interface{}) error {
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
		DataId:  name,
		Group:   env.GetRunEnv(),
		Type:    cType,
		Content: string(bs),
	})
	return err
}
