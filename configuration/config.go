package configuration

type IConfig interface {
	LoadConfig() error
	GetConfig(v interface{}) error
	PublishConfig(name string, v interface{}) error
}

type IWatcherConfig interface {
	AddWatcher(watcher func())
}
