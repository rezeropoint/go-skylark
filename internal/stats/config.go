package stats

import "time"

type Config struct {
	// UserNameCacheTTL 用户名缓存时间
	// key格式: skylark:users:{tenant_id}:{user_id}
	// 默认值: 24小时
	UserNameCacheTTL time.Duration
	// StatsCacheTTL 统计数据缓存时间
	// key格式: skylark:stats:{type}:{tenant_id}:{event_config_id}:{params_hash}
	// 说明: 统计数据更新频率低，可设置较长TTL
	// 默认值: 5分钟
	StatsCacheTTL time.Duration
}
