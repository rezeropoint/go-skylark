package organization

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 组织管理器接口
// 职责：
//  1. 调用 Skylark REST API 管理远程组织（创建、删除、查询）
//  2. 管理本地组织ID与远程组织ID的双向映射
//  3. 提供缓存支持（Redis 30天 TTL）
//  4. 提供批量查询优化（减少数据库访问）
//
// 映射关系：(TenantID + LocalOrgID) ↔ RemoteOrgID
type Manager interface {
	// CreateOrganization 创建根组织（无父组织）
	// 流程：
	//   1. 验证平台配置是否存在
	//   2. 转换创始人本地用户ID为远程用户ID
	//   3. 调用 Skylark API: POST /api/v4/organizations
	//   4. 保存映射关系到数据库
	//   5. 更新缓存（正向 + 反向）
	// 参数：
	//   - founderID: 创始人本地用户ID（string 类型）
	// 返回：错误信息
	// 说明：
	//   - 成功后映射关系已保存，使用者无需关心远程组织ID
	//   - 远程组织ID由SDK内部管理，对使用者透明
	CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description string, founderID string) error

	// CreateSubOrganization 创建子组织
	// 流程：
	//   1. 验证平台配置是否存在
	//   2. 转换创始人本地用户ID为远程用户ID
	//   3. 查询父组织的 remote_org_id
	//   4. 调用 Skylark API: POST /api/v4/organizations（带 parent_id）
	//   5. 保存映射关系到数据库
	//   6. 更新缓存（正向 + 反向）
	// 参数：
	//   - founderID: 创始人本地用户ID（string 类型）
	// 返回：错误信息
	// 错误：如果 parentLocalOrgID 不存在，返回 core.ErrParentOrgNotFound
	// 说明：
	//   - 成功后映射关系已保存，使用者无需关心远程组织ID
	//   - 远程组织ID由SDK内部管理，对使用者透明
	CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description string, founderID string) error

	// DeleteOrganization 删除组织
	// 流程：
	//   1. 查询组织的 remote_org_id
	//   2. 调用 Skylark API: DELETE /api/v4/organizations/:id
	//   3. 删除数据库映射记录
	//   4. 清理缓存（正向 + 反向）
	// 错误：如果组织不存在，返回 core.ErrOrgNotFound
	DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error

	// UpdateOrganization 更新组织信息
	// 流程：
	//   1. 验证平台配置是否存在
	//   2. 查询组织的 remote_org_id
	//   3. 如果需要更新管理员（ManagerID 不为 nil）：
	//      a. 获取分布式锁（防止并发修改）
	//      b. 查询当前所有管理员
	//      c. 转换新管理员的本地用户ID为远程用户ID
	//      d. 添加新管理员（Skylark API）
	//      e. 删除所有旧管理员（Skylark API，多次调用）
	//      f. 释放分布式锁
	//      g. 清理管理员缓存
	//   4. 如果需要更新基本信息（Name 或 Description 不为 nil）：
	//      - 调用 Skylark API: PATCH /api/v4/organizations/:id
	// 参数：req - 更新请求（所有字段可选，nil 表示不修改）
	// 返回：错误信息
	// 说明：
	//   - 使用分布式锁确保管理员更新的原子性
	//   - 支持部分更新（只修改指定字段）
	UpdateOrganization(ctx context.Context, req *core.UpdateOrganizationRequest) error

	// GetRemoteOrgID 查询远程组织ID（内部使用）
	// 流程：
	//   1. 优先从缓存获取
	//   2. 缓存未命中则查询数据库
	//   3. 回写缓存（TTL 30天）
	// 返回：远程组织ID，如果不存在返回错误
	GetRemoteOrgID(ctx context.Context, tenantID, localOrgID string) (int, error)

	// GetRemoteOrgIDs 批量查询远程组织ID（内部使用，性能优化）
	// 流程：
	//   1. 遍历 localOrgIDs，优先从缓存获取
	//   2. 收集缓存未命中的 localOrgIDs
	//   3. 批量查询数据库（IN 子句，减少查询次数）
	//   4. 回写缓存
	//   5. 返回合并结果
	// 返回：远程组织ID列表，顺序与输入一致
	GetRemoteOrgIDs(ctx context.Context, tenantID string, localOrgIDs []string) ([]int, error)

	// GetLocalOrgID 反向查询本地组织ID（内部使用）
	// 流程：
	//   1. 优先从反向缓存获取
	//   2. 缓存未命中则查询数据库
	//   3. 回写反向缓存（TTL 30天）
	// 返回：本地组织ID，如果不存在返回错误
	GetLocalOrgID(ctx context.Context, tenantID string, remoteOrgID int) (string, error)

	// GetOrgSyncStatus 获取组织同步状态
	// 用途：检查本地组织是否已同步到 Skylark 平台
	// 流程：
	//   1. 优先从缓存检查映射是否存在
	//   2. 缓存未命中则查询数据库
	// 返回：
	//   - bool: 是否已同步（true=已同步，false=未同步）
	//   - error: 错误信息（仅数据库错误，未同步不返回错误）
	GetOrgSyncStatus(ctx context.Context, tenantID, localOrgID string) (bool, error)

	// ========== 组织成员管理 ==========

	// GetMembers 获取组织成员列表
	// 流程：
	//   1. 查询本地组织ID对应的远程组织ID
	//   2. 优先从缓存获取成员列表
	//   3. 缓存未命中则调用 Skylark API: GET /api/v4/organizations/:id/members
	//   4. 回写缓存（TTL 5分钟）
	// 参数：
	//   - withDescendants: 是否包含子孙后代组织的成员
	// 返回：成员列表
	GetMembers(ctx context.Context, tenantID, localOrgID string, withDescendants bool) ([]*core.OrganizationMember, error)

	// AddMembers 批量增加组织成员
	// 流程：
	//   1. 查询本地组织ID对应的远程组织ID
	//   2. 调用 Skylark API: PUT /api/v4/organizations/:organization_id/members/add
	//   3. 清除相关缓存（成员列表缓存）
	// 返回：成功添加的成员ID列表
	AddMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error)

	// RemoveMembers 批量移除组织成员
	// 流程：
	//   1. 查询本地组织ID对应的远程组织ID
	//   2. 调用 Skylark API: PUT /api/v4/organizations/:organization_id/members/remove
	//   3. 清除相关缓存（成员列表缓存）
	// 返回：成功移除的成员ID列表
	RemoveMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error)
}

// NewManager 创建组织管理器
// 参数：
//   - config: 组织管理器配置
//   - db: 本地数据库连接（sqlx.SqlConn，用于存储组织ID映射）
//   - cache: 缓存接口（用于缓存组织ID映射，TTL 30天）
//   - getPlatformConfig: 获取平台配置的函数（用于获取 API BaseURL 和 Token）
//   - getRemoteUserIDs: 批量查询远程用户ID的函数（用于转换本地用户ID为远程用户ID）
//   - fillLocalUserIDMap: 批量反向转换远程用户ID为本地用户ID的函数（用于校验成员映射）
//
// 返回：组织管理器实例
func NewManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, fillLocalUserIDMap core.FillLocalUserIDMapFunc) (Manager, error) {
	return newOrganizationManager(config, db, cache, getPlatformConfig, getRemoteUserIDs, fillLocalUserIDMap)
}
