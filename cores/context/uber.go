package context

import (
	"log"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

type Metadata struct {
	metadata.MD
}

func (m *Metadata) Keys() []string {
	var keys []string
	for k := range m.MD {
		keys = append(keys, k)
	}
	return keys
}

func (m *Metadata) Set(key, val string) {
	m.MD[key] = append(m.MD[key], val)
}

func (m *Metadata) Get(key string) string {
	if _, has := m.MD[key]; !has {
		return ""
	}
	if len(m.MD[key]) == 0 {
		return ""
	}
	return m.MD[key][0]
}

type HttpMetadata struct {
	Header http.Header
}

func (m *HttpMetadata) Keys() []string {
	var keys []string
	for k := range m.Header {
		keys = append(keys, k)
	}
	return keys
}

func (m *HttpMetadata) Set(key, val string) {
	log.Printf("[SGT HttpMetadata.Set] ===== CALLED ===== key=%s, val=%s, header=%p", key, val, m.Header)
	m.Header[key] = append(m.Header[key], val)
	log.Printf("[SGT HttpMetadata.Set] After set, header traceparent: %s", m.Header.Get("traceparent"))
}

func (m *HttpMetadata) Get(key string) string {
	log.Printf("[SGT HttpMetadata.Get] ===== CALLED ===== key=%s, header=%p", key, m.Header)
	if _, has := m.Header[key]; !has {
		return ""
	}
	if len(m.Header[key]) == 0 {
		return ""
	}
	return m.Header[key][0]
}

const (
	_uberCtxServiceKey    = "_uber_ctx_service_key"
	_uberCtxTimeoutKey    = "_uber_ctx_timeout_key"
	_uberCtxLangKey       = "lang"
	_uberCtxLangAcceptKey = "Accept-Language"
)

func GetUberMeta(md Metadata) string {
	return md.Get(_uberCtxServiceKey)
}

func SetUberMeta(md Metadata, sk string) {
	md.Set(_uberCtxServiceKey, sk)
}

func GetUberHttpHeader(h http.Header) string {
	return h.Get(_uberCtxServiceKey)
}

func SetUberHttpHeader(h http.Header, sk string) {
	h.Set(_uberCtxServiceKey, sk)
}

func GetUberHttpTimeoutHeader(h http.Header) string {
	return h.Get(_uberCtxTimeoutKey)
}

func SetUberHttpTimeoutHeader(h http.Header, t string) {
	h.Set(_uberCtxTimeoutKey, t)
}

func GetUberHttpLangHeader(h http.Header) string {
	lang := h.Get(_uberCtxLangAcceptKey)
	if lang == "" {
		lang = h.Get(_uberCtxLangKey)
	}
	if lang != "" {
		lang = strings.Split(lang, ",")[0]
	}
	return lang
}

func SetUberHttpLangHeader(h http.Header, lang string) {
	h.Set(_uberCtxLangKey, lang)
}

func GetUberLangHeader(md Metadata) string {
	return md.Get(_uberCtxLangKey)
}

func SetUberLangHeader(md Metadata, lang string) {
	md.Set(_uberCtxLangKey, lang)
}
