package bootstrap

import (
	"sync"
	"vine-agent/config"
)

var (
	configOnce sync.Once
	configs    *config.Config
)

func GetConfig() *config.Config {
	configOnce.Do(func() {
		configs = config.LoadConfig()
	})
	return configs
}
