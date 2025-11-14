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
	//   - *core.Organization: 创建的组织信息（包含 Skylark 组织ID）
	//   - error: 错误信息
	CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description, founderID string) (*core.Organization, error)

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
	//   - *core.Organization: 创建的组织信息（包含父组织ID和祖先路径）
	//   - error: 错误信息
	CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description, founderID string) (*core.Organization, error)

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

	// CreateUser 创建 Skylark 用户
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID
	//   - name: 用户姓名
	//   - identifier: 用户标识符（可选）
	//   - phone: 手机号（可选）
	//   - openid: OpenID（可选）
	//
	// 返回:
	//   - *core.User: 创建的用户信息（包含 Skylark 用户ID）
	//   - error: 错误信息
	CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid *string) (*core.User, error)

	// GetUser 查询用户
	//
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID
	//
	// 返回:
	//   - *core.User: 用户信息
	//   - error: 错误信息（如果映射不存在返回 ErrUserMappingNotFound）
	GetUser(ctx context.Context, tenantID, localUserID string) (*core.User, error)
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
