package file

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type ConfigClient struct {
	ctx      context.Context
	format   string
	cfgValue string
}

func NewConfigClient(ctx context.Context, format string) *ConfigClient {
	return &ConfigClient{
		ctx:    ctx,
		format: format,
	}
}

func (cc *ConfigClient) LoadConfig(key string) error {
	bs, err := os.ReadFile(key)
	if err != nil {
		return err
	}
	if len(bs) == 0 {
		return errors.New("config file does not exist")
	}
	cc.cfgValue = string(bs)
	return nil
}

func (cc *ConfigClient) GetConfig(v interface{}) error {
	if cc.cfgValue == "" {
		return errors.New("config value is empty")
	}
	err := cc.unmarshal(v)
	if err != nil {
		return err
	}
	return nil
}

// PublishConfig 指定配置文件场合无法put配置修正
func (cc *ConfigClient) PublishConfig(_ string, _ interface{}) error {
	return nil
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
