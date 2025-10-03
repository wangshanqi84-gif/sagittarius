package core

import (
	"github.com/IBM/sarama"
)

const (
	_uberCtxServiceKey = "_uber_ctx_service_key"
)

type TextMapMeta struct {
	Data []sarama.RecordHeader
}

func (tm *TextMapMeta) SetUberMeta(sk string) {
	tm.Data = append(tm.Data, sarama.RecordHeader{
		Key:   []byte(_uberCtxServiceKey),
		Value: []byte(sk),
	})
}

func (tm *TextMapMeta) GetUberMeta() string {
	for _, h := range tm.Data {
		if string(h.Key) == _uberCtxServiceKey {
			return string(h.Value)
		}
	}
	return ""
}

func (tm *TextMapMeta) Set(key, val string) {
	tm.Data = append(tm.Data, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

func (tm TextMapMeta) ForeachKey(handler func(key, val string) error) error {
	for _, h := range tm.Data {
		if err := handler(string(h.Key), string(h.Value)); err != nil {
			return err
		}
	}
	return nil
}
