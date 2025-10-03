package env

import (
	"os"
	"strings"
)

func GetEnv(key string) string {
	return os.Getenv(key)
}

func GetRunEnv() string {
	return strings.ToLower(os.Getenv(SgtEnvService))
}

func GetNacosEnv() (string, string, string, string, string) {
	return os.Getenv(SgtNacosServerPath), os.Getenv(SgtNacosAccess),
		os.Getenv(SgtNacosSecret), os.Getenv(SgtNacosUsername), os.Getenv(SgtNacosPassword)
}

func GetEtcdEnv() (string, string, string, string) {
	return os.Getenv(SgtEtcdEndpoints), os.Getenv(SgtEtcdUsername),
		os.Getenv(SgtEtcdPassword), os.Getenv(SgtEtcdDailTimeout)
}
