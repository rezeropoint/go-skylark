package cache

type Config struct {
	FieldMappingsCacheExpiry int `json:"FieldMappingsCacheExpiry"`
	// 分布式锁配置
	FlowLockExpiry        int `json:"FlowLockExpiry"`        // 流程锁过期时间（秒）
	FlowLockRetryCount    int `json:"FlowLockRetryCount"`    // 获取锁重试次数
	FlowLockRetryInterval int `json:"FlowLockRetryInterval"` // 重试间隔（毫秒）
}
