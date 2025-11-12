package platform

import "time"

// Config 平台配置管理器配置
type Config struct {
	// PlatformConfigCacheTTL 平台配置缓存时间
	// key格式: skylark:platform_config:{tenant_id}
	// 默认值: 30分钟
	PlatformConfigCacheTTL time.Duration
}
