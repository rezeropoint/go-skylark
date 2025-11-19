package admin

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// AdminEngine 系统管理引擎接口，专门用于系统管理微服务
// 提供组织和用户管理功能
type AdminEngine interface {
	// CreateOrganization 创建根组织
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//   - name: 组织名称
	//   - description: 组织描述
	//   - founderID: 创建人的本地用户ID
	//
	// 返回:
	//   - error: 错误信息
	//
	// 说明:
	//   - 成功后映射关系已保存，使用者无需关心 Skylark 组织ID
	//   - Skylark 组织ID由SDK内部管理，对使用者透明
	CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description, founderID string) error

	// CreateSubOrganization 创建子组织
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//   - parentLocalOrgID: 父组织的本地ID
	//   - name: 组织名称
	//   - description: 组织描述
	//   - founderID: 创建人的本地用户ID
	//
	// 返回:
	//   - error: 错误信息
	//
	// 说明:
	//   - 成功后映射关系已保存，使用者无需关心 Skylark 组织ID
	//   - Skylark 组织ID由SDK内部管理，对使用者透明
	CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description, founderID string) error

	// DeleteOrganization 删除组织
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//
	// 返回:
	//   - error: 错误信息
	DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error

	// UpdateOrganization 更新组织信息
	//
	// 参数:
	//   - ctx: 上下文
	//   - req: 更新请求参数（所有字段可选，nil 表示不修改）
	//
	// 返回:
	//   - error: 错误信息
	//
	// 说明:
	//   - 支持更新组织名称、描述、管理员
	//   - 更新管理员时使用分布式锁确保原子性（防止并发修改冲突）
	//   - SDK 内部封装多次 API 调用（查询旧管理员 → 添加新管理员 → 删除旧管理员）
	//   - 支持部分更新（只修改指定字段）
	UpdateOrganization(ctx context.Context, req *core.UpdateOrganizationRequest) error

	// GetOrgSyncStatus 获取组织同步状态
	//
	// 用途: 检查本地组织是否已同步到 Skylark 平台
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//
	// 返回:
	//   - bool: 是否已同步（true=已同步到Skylark，false=未同步）
	//   - error: 错误信息（仅数据库错误，未同步不返回错误）
	//
	// 说明:
	//   - 优先从缓存检查，缓存未命中则查询数据库
	//   - 可用于创建流程前验证组织是否已在Skylark中存在
	GetOrgSyncStatus(ctx context.Context, tenantID, localOrgID string) (bool, error)

	// CreateUser 创建 Skylark 用户
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID
	//   - name: 用户姓名
	//   - identifier: 用户标识符（可选，空字符串表示不提供）
	//   - phone: 手机号（可选，空字符串表示不提供）
	//   - openid: OpenID（可选，空字符串表示不提供）
	//
	// 返回:
	//   - error: 错误信息
	//
	// 说明:
	//   - 成功后映射关系已保存，使用者无需关心 Skylark 用户ID
	//   - Skylark 用户ID由SDK内部管理，对使用者透明
	//   - 可选参数传入空字符串时不会发送给Skylark API
	CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid string) error

	// GetUserSyncStatus 获取用户同步状态
	//
	// 用途: 检查本地用户是否已同步到 Skylark 平台
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID
	//
	// 返回:
	//   - bool: 是否已同步（true=已同步到Skylark，false=未同步）
	//   - error: 错误信息（仅数据库错误，未同步不返回错误）
	//
	// 说明:
	//   - 优先从缓存检查，缓存未命中则查询数据库
	//   - 可用于创建流程前验证用户是否已在Skylark中存在
	GetUserSyncStatus(ctx context.Context, tenantID, localUserID string) (bool, error)

	// ========== 组织成员管理 ==========

	// AddMembers 批量添加成员到组织
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//   - memberIDs: 成员ID列表（远程用户ID）
	//
	// 返回:
	//   - []int: 成功添加的成员ID列表
	//   - error: 错误信息
	//
	// 说明:
	//   - 内部调用 GetMembers 校验组织同步状态（确保之前的同步正常）
	//   - 成功后自动清理相关缓存
	AddMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error)

	// RemoveMembers 批量从组织移除成员
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localOrgID: 本地组织ID
	//   - memberIDs: 成员ID列表（远程用户ID）
	//
	// 返回:
	//   - []int: 成功移除的成员ID列表
	//   - error: 错误信息
	//
	// 说明:
	//   - 内部调用 GetMembers 校验组织同步状态（确保之前的同步正常）
	//   - 成功后自动清理相关缓存
	RemoveMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error)
}

// NewAdminEngine 创建新的系统管理引擎实例
//
// 参数:
//   - config: 管理引擎配置
//   - db: 数据库连接（用户需要自己管理连接池）
//   - redisClient: Redis 客户端（可选，用于缓存）
//
// 返回:
//   - AdminEngine: 管理引擎实例
//   - error: 错误信息
//
// 注意:
//   - 在使用 admin 包之前，系统管理微服务需要先配置平台信息
//   - 平台配置存储在数据库的 skylark_platform_configs 表中
//   - admin 包内部会初始化 platform.Manager 为组织和用户管理提供能力
func NewAdminEngine(config *Config, db sqlx.SqlConn, redisClient *redis.Redis) (AdminEngine, error) {
	return newAdminEngine(config, db, redisClient)
}
