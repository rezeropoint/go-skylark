package cache

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// SkylarkCache 缓存实现
type SkylarkCache struct {
	redisClient *redis.Redis
	config      *Config
}

// newSkylarkCache 创建缓存实例（私有构造函数）
func newSkylarkCache(redisClient *redis.Redis, config *Config) *SkylarkCache {
	return &SkylarkCache{
		redisClient: redisClient,
		config:      config,
	}
}
