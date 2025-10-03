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
	ctx    context.Context
	format string
}

func NewConfigClient(ctx context.Context, format string) *ConfigClient {
	return &ConfigClient{
		ctx:    ctx,
		format: format,
	}
}

func (cc *ConfigClient) GetConfig(name string, v interface{}) error {
	bs, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if len(bs) == 0 {
		return errors.New("config file does not exist")
	}
	switch cc.format {
	case "yaml":
		err = yaml.Unmarshal(bs, v)
	case "xml":
		err = xml.Unmarshal(bs, v)
	default:
		err = json.Unmarshal(bs, v)
	}
	return err
}

// PublishConfig 指定配置文件场合无法put配置修正
func (cc *ConfigClient) PublishConfig(_ string, _ interface{}) error {
	return nil
}
