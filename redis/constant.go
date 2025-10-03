package redis

const (
	defaultDialTimeout  = 5   // default: 5s
	defaultReadTimeout  = 3   // default: 3s
	defaultWriteTimeout = 3   // default: 3s
	defaultIdleTimeout  = 30  // default: 30
	defaultRetry        = 3   // 默认重试次数
	defaultPoolSize     = 100 // 默认连接池大小
	defaultMinIdleConn  = 35  // 默认最小连接数辆
)

const (
	typeSingleton = "singleton" // 标准模式
	typeSentinel  = "sentinel"  // 哨兵模式
	typeCluster   = "cluster"   // 集群模式
)
