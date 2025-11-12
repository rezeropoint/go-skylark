package cache

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// SkylarkCache 缓存实现
type SkylarkCache struct {
	redisClient *redis.Redis
	config      *Config
	metrics     *CacheMetrics // 缓存监控指标
}

// newSkylarkCache 创建缓存实例（私有构造函数）
func newSkylarkCache(redisClient *redis.Redis, config *Config) *SkylarkCache {
	return &SkylarkCache{
		redisClient: redisClient,
		config:      config,
		metrics:     &CacheMetrics{}, // 初始化监控指标
	}
}

// GetMetrics 获取缓存监控指标
func (f *SkylarkCache) GetMetrics() *CacheMetrics {
	return f.metrics
}
