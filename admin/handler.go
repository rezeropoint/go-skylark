package admin

import (
	"context"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/cache"
	"github.com/rezeropoint/go-skylark/v2/internal/organization"
	"github.com/rezeropoint/go-skylark/v2/internal/platform"
	"github.com/rezeropoint/go-skylark/v2/internal/user"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type adminEngine struct {
	cache        *cache.SkylarkCache
	platform     platform.Manager     // 内部使用，不对外暴露
	organization organization.Manager // 组织管理
	user         user.Manager         // 用户管理
}

// newAdminEngine 创建新的系统管理引擎实例
func newAdminEngine(config *Config, db sqlx.SqlConn, redisClient *redis.Redis) (*adminEngine, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}
	if db == nil {
		return nil, core.ErrLocalDBNil
	}

	// 初始化缓存
	cache := cache.NewSkylarkCache(redisClient, config.Cache)

	// 1. 初始化平台管理器（核心依赖，最先初始化）
	// 平台管理器为组织和用户管理提供 API 配置能力
	platformConfig := platform.Config{}
	if config.Platform != nil {
		platformConfig = *config.Platform
	}
	// 设置默认值
	if platformConfig.PlatformConfigCacheTTL == 0 {
		platformConfig.PlatformConfigCacheTTL = 30 * time.Minute
	}
	platformMgr, err := platform.NewManager(platformConfig, db, cache)
	if err != nil {
		return nil, err
	}

	// 2. 初始化用户管理器（依赖 Platform.GetAPIConfig）
	userMgr, err := user.NewManager(user.Config{}, db, cache, platformMgr.GetAPIConfig)
	if err != nil {
		return nil, err
	}

	// 3. 初始化组织管理器（依赖 Platform.GetAPIConfig + User.GetRemoteUserIDs + User.FillLocalUserIDMap）
	orgMgr, err := organization.NewManager(organization.Config{}, db, cache, platformMgr.GetAPIConfig, userMgr.GetRemoteUserIDs, userMgr.FillLocalUserIDMap)
	if err != nil {
		return nil, err
	}

	return &adminEngine{
		cache:        cache,
		platform:     platformMgr,
		organization: orgMgr,
		user:         userMgr,
	}, nil
}

// CreateOrganization 创建根组织
func (e *adminEngine) CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description, founderID string) error {
	return e.organization.CreateOrganization(ctx, tenantID, localOrgID, name, description, founderID)
}

// CreateSubOrganization 创建子组织
func (e *adminEngine) CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description, founderID string) error {
	return e.organization.CreateSubOrganization(ctx, tenantID, localOrgID, parentLocalOrgID, name, description, founderID)
}

// DeleteOrganization 删除组织
func (e *adminEngine) DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error {
	return e.organization.DeleteOrganization(ctx, tenantID, localOrgID)
}

// UpdateOrganization 更新组织信息
func (e *adminEngine) UpdateOrganization(ctx context.Context, req *core.UpdateOrganizationRequest) error {
	return e.organization.UpdateOrganization(ctx, req)
}

// CreateUser 创建 Skylark 用户
func (e *adminEngine) CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid string) error {
	return e.user.CreateUser(ctx, tenantID, localUserID, name, identifier, phone, openid)
}

// GetOrgSyncStatus 获取组织同步状态
func (e *adminEngine) GetOrgSyncStatus(ctx context.Context, tenantID, localOrgID string) (bool, error) {
	return e.organization.GetOrgSyncStatus(ctx, tenantID, localOrgID)
}

// GetUserSyncStatus 获取用户同步状态
func (e *adminEngine) GetUserSyncStatus(ctx context.Context, tenantID, localUserID string) (bool, error) {
	return e.user.GetUserSyncStatus(ctx, tenantID, localUserID)
}

// ========== 组织ID映射管理 ==========

// BindOrganization 绑定已存在的远程组织
func (e *adminEngine) BindOrganization(ctx context.Context, tenantID, localOrgID string, remoteOrgID int) error {
	return e.organization.BindOrganization(ctx, tenantID, localOrgID, remoteOrgID)
}

// UnbindOrganization 解绑组织映射
func (e *adminEngine) UnbindOrganization(ctx context.Context, tenantID, localOrgID string) error {
	return e.organization.UnbindOrganization(ctx, tenantID, localOrgID)
}

// ========== 用户ID映射管理 ==========

// BindUser 绑定已存在的远程用户
func (e *adminEngine) BindUser(ctx context.Context, tenantID, localUserID string, remoteUserID int) error {
	return e.user.BindUser(ctx, tenantID, localUserID, remoteUserID)
}

// UnbindUser 解绑用户映射
func (e *adminEngine) UnbindUser(ctx context.Context, tenantID, localUserID string) error {
	return e.user.UnbindUser(ctx, tenantID, localUserID)
}

// ========== 组织成员管理 ==========

// AddMembers 批量添加成员到组织
func (e *adminEngine) AddMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error) {
	return e.organization.AddMembers(ctx, tenantID, localOrgID, memberIDs)
}

// RemoveMembers 批量从组织移除成员
func (e *adminEngine) RemoveMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error) {
	return e.organization.RemoveMembers(ctx, tenantID, localOrgID, memberIDs)
}
