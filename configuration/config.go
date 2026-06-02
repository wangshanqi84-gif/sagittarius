package configuration

type IConfig interface {
	LoadConfig(key string) error
	GetConfig(v interface{}) error
	PublishConfig(name string, v interface{}) error
}
