// Package query 提供远程查询管理功能
package query

import "time"

// Config 远程查询管理器配置
type Config struct {
	// Redis缓存TTL配置

	// FlowListCacheTTL flows列表缓存时间
	// key格式: skylark:flows:{tenant_id}:{namespace_id}
	// 默认值: 1小时
	FlowListCacheTTL time.Duration

	// FlowFieldsCacheTTL flow字段列表缓存时间
	// key格式: skylark:flow_fields:{tenant_id}:{flow_id}
	// 默认值: 1小时
	FlowFieldsCacheTTL time.Duration

	// UserNameCacheTTL 用户名缓存时间
	// key格式: skylark:users:{tenant_id}:{user_id}
	// 默认值: 24小时
	UserNameCacheTTL time.Duration

	// 查询限制配置

	// MaxPageSize 每页最大记录数
	// 防止单次查询数据量过大
	// 默认值: 1000
	MaxPageSize int

	// DefaultPageSize 默认每页记录数
	// 当请求未指定PageSize时使用
	// 默认值: 20
	DefaultPageSize int

	// QueryTimeout 单次查询超时时间
	// 防止慢查询阻塞
	// 默认值: 30秒
	QueryTimeout time.Duration
}
