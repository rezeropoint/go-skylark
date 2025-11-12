// Package mapping 提供组织映射管理功能
package mapping

import "time"

// Config 组织映射管理器配置
type Config struct {
	// OrgMappingCacheTTL 组织映射缓存时间
	// key格式: skylark:mapping:{id} 和 skylark:mapping_list:{tenant_id}
	// 默认值: 7天
	OrgMappingCacheTTL time.Duration
}
