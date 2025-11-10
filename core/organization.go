// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：组织管理
// 说明：本文件定义的类型用于管理 Skylark 组织及其映射关系
package core

import (
	"context"
	"time"
)

// Organization Skylark 组织领域模型
// 说明：存储 Skylark 组织的 ID 和层级关系信息（纯 Go 类型，无框架依赖）
// 注意：不存储 name、description 等本地已有的信息，本地组织信息由调用方的 GO 项目管理
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

// GetPlatformConfigFunc 获取平台配置的函数类型
// 用途：供 Organization/User Manager 获取 Skylark 平台配置，实现 Manager 之间解耦
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//
// 返回：
//   - *PlatformConfig: 平台配置信息（包含 BaseURL、Authorization、NamespaceID 等）
//   - error: 错误信息（如平台配置不存在）
//
// 说明：
//   - 该函数会从 skylark_platform_configs 表读取配置
//   - 由 Platform Manager 的 Get 方法实现
type GetPlatformConfigFunc func(ctx context.Context, tenantID string) (*PlatformConfig, error)

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
