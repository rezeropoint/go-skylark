// Package event 提供事件配置管理功能
package event

import "time"

// Config 事件配置管理器配置
type Config struct {
	// EventConfigCacheTTL 事件配置缓存时间（单个事件）
	// key格式: skylark:event_config:{tenant_id}:{id}
	// 默认值: 10分钟
	EventConfigCacheTTL time.Duration

	// EventConfigListCacheTTL 事件配置列表缓存时间
	// key格式: skylark:event_list:{tenant_id}:enabled={true|false|all}
	// 默认值: 5分钟
	EventConfigListCacheTTL time.Duration
}
