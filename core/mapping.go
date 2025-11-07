// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：组织映射管理、权限过滤）
package core

import "time"

// 组织映射缓存键常量
const (
	// CacheMappingKeyPrefix 单个组织映射缓存键前缀
	// 完整格式：skylark:mapping:{id}
	CacheMappingKeyPrefix = "skylark:mapping:"

	// CacheMappingListKeyPrefix 组织映射列表缓存键前缀
	// 完整格式：skylark:mapping_list:{tenant_id}
	CacheMappingListKeyPrefix = "skylark:mapping_list:"
)

// OrgMapping 组织映射领域模型（纯领域模型）
// 核心作用：将远程业务字段值（如"仓库名称"）映射到本地组织ID，实现组织架构管理
// 注意：OrgMapping 是租户级别的全局配置，多个事件类型可以共享同一个组织映射
// 映射关系：(TenantID + RemoteOrgValue) → LocalOrgID
//
// 使用场景：
// 1. 平台配置一次映射："自贡网络维护中心地市仓库" → org-001
// 2. 事件A配置 org_field_name = "username"，查询时过滤 username = "自贡网络维护中心地市仓库"
// 3. 事件B同样配置 org_field_name = "username"，使用相同的映射关系
// 4. 多个事件共享映射，避免重复配置，集中管理
type OrgMapping struct {
	ID             string    // 映射UUID
	RemoteOrgValue string    // 远程表组织字段的值（如："自贡网络维护中心地市仓库"）
	LocalOrgID     string    // 本地组织ID（如：org-001）
	LocalOrgName   string    // 本地组织名称（查询时通过JOIN获取，不存储）
	TenantID       string    // 租户ID（多个事件可共享同一租户下的映射）
	CreatedAt      time.Time // 创建时间
	UpdatedAt      time.Time // 更新时间
}
