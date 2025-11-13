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
	//   1. 调用 Skylark API: POST /api/v4/organizations
	//   2. 保存映射关系到数据库
	//   3. 更新缓存（正向 + 反向）
	// 返回：创建成功的组织信息（包含 RemoteOrgID、ParentID、Ancestry）
	CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description string, founderID int) (*core.Organization, error)

	// CreateSubOrganization 创建子组织
	// 流程：
	//   1. 查询父组织的 remote_org_id
	//   2. 调用 Skylark API: POST /api/v4/organizations（带 parent_id）
	//   3. 保存映射关系到数据库
	//   4. 更新缓存（正向 + 反向）
	// 返回：创建成功的组织信息（包含 RemoteOrgID、ParentID、Ancestry）
	// 错误：如果 parentLocalOrgID 不存在，返回 core.ErrParentOrgNotFound
	CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description string, founderID int) (*core.Organization, error)

	// DeleteOrganization 删除组织
	// 流程：
	//   1. 查询组织的 remote_org_id
	//   2. 调用 Skylark API: DELETE /api/v4/organizations/:id
	//   3. 删除数据库映射记录
	//   4. 清理缓存（正向 + 反向）
	// 错误：如果组织不存在，返回 core.ErrOrgNotFound
	DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error

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
}

// NewManager 创建组织管理器
// 参数：
//   - config: 组织管理器配置
//   - db: 本地数据库连接（sqlx.SqlConn，用于存储组织ID映射）
//   - cache: 缓存接口（用于缓存组织ID映射，TTL 30天）
//   - getPlatformConfig: 获取平台配置的函数（用于获取 API BaseURL 和 Token）
//
// 返回：组织管理器实例
func NewManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc) (Manager, error) {
	return newOrganizationManager(config, db, cache, getPlatformConfig)
}
