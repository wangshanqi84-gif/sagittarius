package configuration

type IConfig interface {
	GetConfig(name string, v interface{}) error
	PublishConfig(name string, v interface{}) error
}
