// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：组织管理
// 说明：本文件定义的类型用于管理 Skylark 组织及其映射关系
package core

import (
	"context"
	"time"
)

// 组织ID映射缓存键常量
const (
	// CacheOrgIDMappingKeyPrefix 组织ID映射缓存键前缀（正向：local_org_id -> remote_org_id）
	// 格式：skylark:org_mapping:{tenantID}:{localOrgID}
	// TTL: 30天
	CacheOrgIDMappingKeyPrefix = "skylark:org_mapping"

	// CacheOrgIDMappingReverseKeyPrefix 组织ID映射反向缓存键前缀（反向：remote_org_id -> local_org_id）
	// 格式：skylark:org_mapping_rev:{tenantID}:{remoteOrgID}
	// TTL: 30天
	CacheOrgIDMappingReverseKeyPrefix = "skylark:org_mapping_rev"
)

// Organization Skylark 组织领域模型（内部使用）
// 说明：存储 Skylark 组织的 ID 和层级关系信息（纯 Go 类型，无框架依赖）
// 注意：
//   - 不存储 name、description 等本地已有的信息，本地组织信息由调用方的 GO 项目管理
//   - 这个结构体仅用于 SDK 内部（如 API 响应解析），不暴露给使用者
//   - 远程组织ID对SDK使用者透明，使用者只需关心本地组织ID
type Organization struct {
	ID       int     // Skylark 组织ID（整数）
	ParentID *int    // 父组织的 Skylark ID（可空，顶级组织为 nil）
	Ancestry *string // 祖先路径（可空，如 "14516/15113"，用于快速查询组织层级）
}

// OrgIDMapping 组织ID映射领域模型
// 说明：管理本地组织ID与 Skylark 组织ID的映射关系
// 用途：支持双向查询（local_org_id ↔ remote_org_id）
// 注意：与 OrgMapping（业务字段值映射）不同，OrgIDMapping 映射的是组织ID本身
type OrgIDMapping struct {
	ID          string    // UUID
	TenantID    string    // 租户ID
	LocalOrgID  string    // 本地组织ID（由调用方的 GO 项目生成和管理）
	RemoteOrgID int       // Skylark 组织ID（整数，调用 Skylark API 创建后获得）
	CreatedAt   time.Time // 创建时间
	UpdatedAt   time.Time // 更新时间
}

// GetRemoteOrgIDsFunc 批量查询远程组织ID的函数类型
// 用途：供 Query/Stats Manager 根据本地组织ID批量查询 Skylark 组织ID
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - localOrgIDs: 本地组织ID列表
//
// 返回：
//   - []int: Skylark 组织ID列表
//   - error: 错误信息（如映射不存在）
//
// 说明：
//   - 该函数会优先查询缓存，缓存未命中时查询数据库
//   - 用于组织权限过滤场景
//   - 由 Organization Manager 的 GetRemoteOrgIDs 方法实现
type GetRemoteOrgIDsFunc func(ctx context.Context, tenantID string, localOrgIDs []string) ([]int, error)
