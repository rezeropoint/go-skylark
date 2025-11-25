package cache

// Config 缓存配置
type Config struct {
	// API 模块缓存配置
	FieldMappingsCacheExpiry int `json:"FieldMappingsCacheExpiry"` // 字段映射缓存过期时间（秒）

	// 分布式锁配置
	FlowLockExpiry        int `json:"FlowLockExpiry"`        // 流程锁过期时间（秒）
	FlowLockRetryCount    int `json:"FlowLockRetryCount"`    // 获取锁重试次数
	FlowLockRetryInterval int `json:"FlowLockRetryInterval"` // 重试间隔（毫秒）

	// Query 模块缓存配置
	FlowListCacheTTL   int `json:"FlowListCacheTTL"`   // Flow 列表缓存 TTL（秒），默认 3600
	FlowFieldsCacheTTL int `json:"FlowFieldsCacheTTL"` // Flow 字段缓存 TTL（秒），默认 120（2分钟，支持快速修改字段名）
	UserNameCacheTTL   int `json:"UserNameCacheTTL"`   // 用户名缓存 TTL（秒），默认 86400

	// Mapping 模块缓存配置
	OrgMappingCacheTTL     int `json:"OrgMappingCacheTTL"`     // 单个组织映射缓存 TTL（秒），默认 3600
	OrgMappingListCacheTTL int `json:"OrgMappingListCacheTTL"` // 组织映射列表缓存 TTL（秒），默认 3600

	// Stats 模块缓存配置
	StatsCacheTTL int `json:"StatsCacheTTL"` // 统计结果缓存 TTL（秒），默认 300
}
