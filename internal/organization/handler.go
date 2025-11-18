package organization

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// organizationManager 组织管理器实现
type organizationManager struct {
	localDB           sqlx.SqlConn                 // 本地数据库连接
	cache             core.CacheInterface          // 缓存接口
	getPlatformConfig core.GetPlatformConfigFunc   // 获取平台配置函数
	getRemoteUserIDs  core.GetRemoteUserIDsFunc    // 批量查询远程用户ID函数
	httpClient        *skylarkHTTPClient           // HTTP 客户端
}

// newOrganizationManager 创建组织管理器
func newOrganizationManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc) (*organizationManager, error) {
	// 验证参数
	if db == nil {
		return nil, fmt.Errorf("db 不能为空")
	}
	if cache == nil {
		return nil, fmt.Errorf("cache 不能为空")
	}
	if getPlatformConfig == nil {
		return nil, fmt.Errorf("getPlatformConfig 函数不能为空")
	}
	if getRemoteUserIDs == nil {
		return nil, fmt.Errorf("getRemoteUserIDs 函数不能为空")
	}

	manager := &organizationManager{
		localDB:           db,
		cache:             cache,
		getPlatformConfig: getPlatformConfig,
		getRemoteUserIDs:  getRemoteUserIDs,
		httpClient:        newSkylarkHTTPClient(),
	}

	// 初始化数据库表（在包初始化时执行）
	ctx := context.Background()
	if err := manager.initTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化组织映射表失败: %w", err)
	}

	return manager, nil
}

// CreateOrganization 创建根组织
func (m *organizationManager) CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description string, founderID string) error {
	// 1. 获取平台配置（已验证 APIBaseURL、APIToken）
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return err
	}

	// 2. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := m.getRemoteUserIDs(ctx, tenantID, []string{founderID})
	if err != nil {
		return fmt.Errorf("转换创始人用户ID失败: %w", err)
	}
	if len(remoteUserIDs) == 0 {
		return fmt.Errorf("本地用户ID %s 未找到对应的远程用户ID", founderID)
	}
	remoteFounderID := remoteUserIDs[0]

	// 3. 调用 Skylark API 创建组织
	req := &CreateOrganizationRequest{
		Name:                name,
		Description:         description,
		ApplicationStrategy: "closed", // 默认关闭申请
		FounderID:           remoteFounderID,
		ParentID:            nil, // 根组织无父组织
	}

	orgResp, err := m.httpClient.createOrganization(ctx, platformConfig.App, platformConfig.Token, req)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrOrgCreateFailed, err)
	}

	// 4. 保存映射关系
	mapping := &core.OrgIDMapping{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		LocalOrgID:  localOrgID,
		RemoteOrgID: orgResp.ID,
	}

	if err := m.saveMapping(ctx, mapping); err != nil {
		// TODO: 考虑是否需要回滚远程创建的组织（调用删除API）
		return fmt.Errorf("保存映射失败: %w", err)
	}

	// 5. 更新缓存
	if err := m.cacheMapping(ctx, mapping); err != nil {
		logx.WithContext(ctx).Error("更新缓存失败（非致命错误）:", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "organization_manager"),
		logx.Field("operation", "create_organization"),
		logx.Field("tenant_id", tenantID),
		logx.Field("local_org_id", localOrgID),
		logx.Field("remote_org_id", orgResp.ID),
	).Info("创建组织成功")

	return nil
}

// CreateSubOrganization 创建子组织
func (m *organizationManager) CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description string, founderID string) error {
	// 1. 获取平台配置（已验证 APIBaseURL、APIToken）
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return err
	}

	// 2. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := m.getRemoteUserIDs(ctx, tenantID, []string{founderID})
	if err != nil {
		return fmt.Errorf("转换创始人用户ID失败: %w", err)
	}
	if len(remoteUserIDs) == 0 {
		return fmt.Errorf("本地用户ID %s 未找到对应的远程用户ID", founderID)
	}
	remoteFounderID := remoteUserIDs[0]

	// 3. 查询父组织的 remote_org_id
	parentRemoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, parentLocalOrgID)
	if err != nil {
		if err == core.ErrOrgNotFound {
			return core.ErrParentOrgNotFound
		}
		return fmt.Errorf("查询父组织映射失败: %w", err)
	}

	// 4. 调用 Skylark API 创建子组织
	req := &CreateOrganizationRequest{
		Name:                name,
		Description:         description,
		ApplicationStrategy: "closed",
		FounderID:           remoteFounderID,
		ParentID:            &parentRemoteOrgID, // 指定父组织ID
	}

	orgResp, err := m.httpClient.createOrganization(ctx, platformConfig.App, platformConfig.Token, req)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrOrgCreateFailed, err)
	}

	// 5. 保存映射关系
	mapping := &core.OrgIDMapping{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		LocalOrgID:  localOrgID,
		RemoteOrgID: orgResp.ID,
	}

	if err := m.saveMapping(ctx, mapping); err != nil {
		return fmt.Errorf("保存映射失败: %w", err)
	}

	// 6. 更新缓存
	if err := m.cacheMapping(ctx, mapping); err != nil {
		logx.WithContext(ctx).Error("更新缓存失败（非致命错误）:", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "organization_manager"),
		logx.Field("operation", "create_sub_organization"),
		logx.Field("tenant_id", tenantID),
		logx.Field("local_org_id", localOrgID),
		logx.Field("parent_local_org_id", parentLocalOrgID),
		logx.Field("remote_org_id", orgResp.ID),
		logx.Field("parent_remote_org_id", parentRemoteOrgID),
	).Info("创建子组织成功")

	return nil
}

// DeleteOrganization 删除组织
func (m *organizationManager) DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error {
	// 1. 查询组织的 remote_org_id
	remoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err != nil {
		if err == core.ErrOrgNotFound {
			return core.ErrOrgNotFound
		}
		return fmt.Errorf("查询组织映射失败: %w", err)
	}

	// 2. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 验证 API 配置
	// 3. 调用 Skylark API 删除组织
	if err := m.httpClient.deleteOrganization(ctx, platformConfig.App, platformConfig.Token, remoteOrgID); err != nil {
		return fmt.Errorf("%w: %v", core.ErrOrgDeleteFailed, err)
	}

	// 4. 删除数据库映射记录
	if err := m.deleteMapping(ctx, tenantID, localOrgID); err != nil {
		return fmt.Errorf("删除映射失败: %w", err)
	}

	// 5. 清理缓存
	if err := m.deleteCacheMapping(ctx, tenantID, localOrgID, remoteOrgID); err != nil {
		logx.WithContext(ctx).Error("清理缓存失败（非致命错误）:", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "organization_manager"),
		logx.Field("operation", "delete_organization"),
		logx.Field("tenant_id", tenantID),
		logx.Field("local_org_id", localOrgID),
		logx.Field("remote_org_id", remoteOrgID),
	).Info("删除组织成功")

	return nil
}

// GetRemoteOrgID 查询远程组织ID
func (m *organizationManager) GetRemoteOrgID(ctx context.Context, tenantID, localOrgID string) (int, error) {
	// 1. 优先从缓存获取
	remoteOrgID, err := m.cache.GetOrgIDMapping(ctx, tenantID, localOrgID)
	if err == nil {
		return remoteOrgID, nil
	}

	// 2. 缓存未命中，查询数据库
	query := "SELECT remote_org_id FROM skylark_org_mappings WHERE tenant_id = $1 AND local_org_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &remoteOrgID, query, tenantID, localOrgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, core.ErrOrgNotFound
		}
		return 0, fmt.Errorf("查询映射失败: %w", err)
	}

	// 3. 回写缓存（TTL 30天）
	if err := m.cache.SetOrgIDMapping(ctx, tenantID, localOrgID, remoteOrgID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
	}

	return remoteOrgID, nil
}

// GetRemoteOrgIDs 批量查询远程组织ID
func (m *organizationManager) GetRemoteOrgIDs(ctx context.Context, tenantID string, localOrgIDs []string) ([]int, error) {
	if len(localOrgIDs) == 0 {
		return []int{}, nil
	}

	result := make([]int, len(localOrgIDs))
	cacheMissIndices := []int{}
	cacheMissIDs := []string{}

	// 1. 遍历查询缓存
	for i, localOrgID := range localOrgIDs {
		remoteOrgID, err := m.cache.GetOrgIDMapping(ctx, tenantID, localOrgID)
		if err == nil {
			result[i] = remoteOrgID
		} else {
			cacheMissIndices = append(cacheMissIndices, i)
			cacheMissIDs = append(cacheMissIDs, localOrgID)
		}
	}

	// 2. 批量查询数据库（缓存未命中的）
	if len(cacheMissIDs) > 0 {
		query := "SELECT local_org_id, remote_org_id FROM skylark_org_mappings WHERE tenant_id = $1 AND local_org_id = ANY($2)"
		var mappings []struct {
			LocalOrgID  string `db:"local_org_id"`
			RemoteOrgID int    `db:"remote_org_id"`
		}
		err := m.localDB.QueryRowsCtx(ctx, &mappings, query, tenantID, pq.Array(cacheMissIDs))
		if err != nil {
			return nil, fmt.Errorf("批量查询映射失败: %w", err)
		}

		// 3. 回写缓存并填充结果
		mappingMap := make(map[string]int)
		for _, mapping := range mappings {
			mappingMap[mapping.LocalOrgID] = mapping.RemoteOrgID

			// 回写缓存
			if err := m.cache.SetOrgIDMapping(ctx, tenantID, mapping.LocalOrgID, mapping.RemoteOrgID, 30*24*3600); err != nil {
				logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
			}
		}

		// 填充结果并检查映射完整性
		var failedIDs []string
		for i, localOrgID := range cacheMissIDs {
			if remoteOrgID, exists := mappingMap[localOrgID]; exists {
				result[cacheMissIndices[i]] = remoteOrgID
			} else {
				// 记录失败的组织ID
				failedIDs = append(failedIDs, localOrgID)
				result[cacheMissIndices[i]] = 0
			}
		}

		// 如果有映射缺失，返回详细错误
		if len(failedIDs) > 0 {
			return nil, fmt.Errorf("批量查询组织映射失败: 以下本地组织ID未找到映射关系 %v，请先调用 OrganizationManager.Create() 创建组织映射 (%w)", failedIDs, core.ErrOrgNotFound)
		}
	}

	return result, nil
}

// GetLocalOrgID 反向查询本地组织ID
func (m *organizationManager) GetLocalOrgID(ctx context.Context, tenantID string, remoteOrgID int) (string, error) {
	// 1. 优先从反向缓存获取
	localOrgID, err := m.cache.GetOrgIDMappingReverse(ctx, tenantID, remoteOrgID)
	if err == nil {
		return localOrgID, nil
	}

	// 2. 缓存未命中，查询数据库
	query := "SELECT local_org_id FROM skylark_org_mappings WHERE tenant_id = $1 AND remote_org_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &localOrgID, query, tenantID, remoteOrgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", core.ErrOrgNotFound
		}
		return "", fmt.Errorf("查询映射失败: %w", err)
	}

	// 3. 回写反向缓存（TTL 30天）
	if err := m.cache.SetOrgIDMappingReverse(ctx, tenantID, remoteOrgID, localOrgID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写反向缓存失败（非致命错误）:", err)
	}

	return localOrgID, nil
}

// GetOrgSyncStatus 获取组织同步状态
func (m *organizationManager) GetOrgSyncStatus(ctx context.Context, tenantID, localOrgID string) (bool, error) {
	// 1. 优先从缓存检查
	_, err := m.cache.GetOrgIDMapping(ctx, tenantID, localOrgID)
	if err == nil {
		// 缓存命中，说明映射存在
		return true, nil
	}

	// 2. 缓存未命中，查询数据库
	var remoteOrgID int
	query := "SELECT remote_org_id FROM skylark_org_mappings WHERE tenant_id = $1 AND local_org_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &remoteOrgID, query, tenantID, localOrgID)
	if err != nil {
		if err == sql.ErrNoRows {
			// 映射不存在，返回 false（不返回错误）
			return false, nil
		}
		// 数据库查询错误
		return false, fmt.Errorf("查询组织映射失败: %w", err)
	}

	// 3. 映射存在，回写缓存（TTL 30天）
	if err := m.cache.SetOrgIDMapping(ctx, tenantID, localOrgID, remoteOrgID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
	}

	return true, nil
}

// initTable 初始化组织ID映射表（私有方法，在包初始化时调用）
func (m *organizationManager) initTable(ctx context.Context) error {
	// 检查表是否已存在
	var count int
	err := m.localDB.QueryRowCtx(ctx, &count, CheckTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查组织映射表存在性失败: %w", err)
	}

	// 如果表已存在，直接返回
	if count > 0 {
		return nil
	}

	// 表不存在，执行创建
	_, err = m.localDB.ExecCtx(ctx, CreateTableSQL)
	if err != nil {
		return fmt.Errorf("创建组织映射表失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "organization_manager"),
		logx.Field("operation", "init_table"),
		logx.Field("table", TableName),
	).Info("组织映射表初始化成功")

	return nil
}

// saveMapping 保存映射到数据库
func (m *organizationManager) saveMapping(ctx context.Context, mapping *core.OrgIDMapping) error {
	query := `
		INSERT INTO skylark_org_mappings (id, tenant_id, local_org_id, remote_org_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := m.localDB.ExecCtx(ctx, query, mapping.ID, mapping.TenantID, mapping.LocalOrgID, mapping.RemoteOrgID)
	if err != nil {
		return fmt.Errorf("插入映射失败: %w", err)
	}
	return nil
}

// deleteMapping 删除映射
func (m *organizationManager) deleteMapping(ctx context.Context, tenantID, localOrgID string) error {
	query := "DELETE FROM skylark_org_mappings WHERE tenant_id = $1 AND local_org_id = $2"
	_, err := m.localDB.ExecCtx(ctx, query, tenantID, localOrgID)
	if err != nil {
		return fmt.Errorf("删除映射失败: %w", err)
	}
	return nil
}

// cacheMapping 缓存映射（正向 + 反向）
func (m *organizationManager) cacheMapping(ctx context.Context, mapping *core.OrgIDMapping) error {
	// 正向缓存：local_org_id -> remote_org_id
	if err := m.cache.SetOrgIDMapping(ctx, mapping.TenantID, mapping.LocalOrgID, mapping.RemoteOrgID, 30*24*3600); err != nil {
		return err
	}

	// 反向缓存：remote_org_id -> local_org_id
	if err := m.cache.SetOrgIDMappingReverse(ctx, mapping.TenantID, mapping.RemoteOrgID, mapping.LocalOrgID, 30*24*3600); err != nil {
		return err
	}

	return nil
}

// deleteCacheMapping 删除缓存映射（正向 + 反向）
func (m *organizationManager) deleteCacheMapping(ctx context.Context, tenantID, localOrgID string, remoteOrgID int) error {
	// 删除正向缓存
	if err := m.cache.DeleteOrgIDMapping(ctx, tenantID, localOrgID); err != nil {
		return err
	}

	// 删除反向缓存
	if err := m.cache.DeleteOrgIDMappingReverse(ctx, tenantID, remoteOrgID); err != nil {
		return err
	}

	return nil
}
