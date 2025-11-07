package cache

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// NewSkylarkCache 创建新的 Skylark 缓存实例
// 参数:
//   - redisClient: Redis 客户端
//   - config: 缓存配置
func NewSkylarkCache(redisClient *redis.Redis, config *Config) *SkylarkCache {
	return newSkylarkCache(redisClient, config)
}
