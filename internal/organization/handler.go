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
	localDB            sqlx.SqlConn                // 本地数据库连接
	cache              core.CacheInterface         // 缓存接口
	getPlatformConfig  core.GetPlatformConfigFunc  // 获取平台配置函数
	getRemoteUserIDs   core.GetRemoteUserIDsFunc   // 批量查询远程用户ID函数
	fillLocalUserIDMap core.FillLocalUserIDMapFunc // 批量反向转换远程用户ID为本地用户ID函数
	httpClient         *skylarkHTTPClient          // HTTP 客户端
}

// newOrganizationManager 创建组织管理器
func newOrganizationManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, fillLocalUserIDMap core.FillLocalUserIDMapFunc) (*organizationManager, error) {
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
	if fillLocalUserIDMap == nil {
		return nil, fmt.Errorf("fillLocalUserIDMap 函数不能为空")
	}

	manager := &organizationManager{
		localDB:            db,
		cache:              cache,
		getPlatformConfig:  getPlatformConfig,
		getRemoteUserIDs:   getRemoteUserIDs,
		fillLocalUserIDMap: fillLocalUserIDMap,
		httpClient:         newSkylarkHTTPClient(),
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

// BindOrganization 绑定已存在的远程组织
func (m *organizationManager) BindOrganization(ctx context.Context, tenantID, localOrgID string, remoteOrgID int) error {
	// 1. 获取平台配置（验证租户配置是否存在）
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return err
	}

	// 2. 调用 Skylark API 验证远程组织是否存在
	_, err = m.httpClient.getOrganization(ctx, platformConfig.App, platformConfig.Token, remoteOrgID)
	if err != nil {
		// 如果是404错误，返回 ErrOrgNotFound
		return fmt.Errorf("%w: 远程组织ID %d 不存在", core.ErrOrgNotFound, remoteOrgID)
	}

	// 3. 检查本地组织ID是否已绑定
	_, err = m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err == nil {
		// 映射已存在
		return fmt.Errorf("%w: 本地组织ID %s 已绑定", core.ErrOrgIDMappingExists, localOrgID)
	}
	if err != core.ErrOrgNotFound {
		// 数据库查询错误
		return fmt.Errorf("检查映射失败: %w", err)
	}

	// 4. 保存映射关系到数据库
	mapping := &core.OrgIDMapping{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		LocalOrgID:  localOrgID,
		RemoteOrgID: remoteOrgID,
	}

	if err := m.saveMapping(ctx, mapping); err != nil {
		return fmt.Errorf("保存映射失败: %w", err)
	}

	// 5. 更新缓存（正向 + 反向）
	if err := m.cacheMapping(ctx, mapping); err != nil {
		logx.WithContext(ctx).Error("更新缓存失败（非致命错误）:", err)
	}

	return nil
}

// UnbindOrganization 解绑组织映射
func (m *organizationManager) UnbindOrganization(ctx context.Context, tenantID, localOrgID string) error {
	// 1. 检查映射是否存在
	remoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err != nil {
		// 映射不存在
		return err
	}

	// 2. 删除数据库映射记录
	if err := m.deleteMapping(ctx, tenantID, localOrgID); err != nil {
		return fmt.Errorf("删除映射失败: %w", err)
	}

	// 3. 清理缓存（正向 + 反向）
	if err := m.cache.DeleteOrgIDMapping(ctx, tenantID, localOrgID); err != nil {
		logx.WithContext(ctx).Error("清理正向缓存失败（非致命错误）:", err)
	}
	if err := m.cache.DeleteOrgIDMappingReverse(ctx, tenantID, remoteOrgID); err != nil {
		logx.WithContext(ctx).Error("清理反向缓存失败（非致命错误）:", err)
	}

	return nil
}

// ========== 组织成员管理实现 ==========

// GetMembers 获取组织成员列表
func (m *organizationManager) GetMembers(ctx context.Context, tenantID, localOrgID string, withDescendants bool) ([]*core.OrganizationMember, error) {
	// 1. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 查询远程组织ID
	remoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err != nil {
		return nil, err
	}

	// 3. 尝试从缓存获取
	members, err := m.cache.GetOrgMembers(ctx, tenantID, remoteOrgID, withDescendants)
	if err == nil {
		return members, nil
	}

	// 4. 缓存未命中，调用 Skylark API
	memberModels, err := m.httpClient.getMembers(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, withDescendants)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrGetMembersFailed, err)
	}

	// 5. 转换为领域模型
	members = make([]*core.OrganizationMember, 0, len(memberModels))
	for i := range memberModels {
		members = append(members, memberModels[i].ToDomain())
	}

	// 6. 回写缓存（TTL 5分钟）
	if err := m.cache.SetOrgMembers(ctx, tenantID, remoteOrgID, withDescendants, members, 300); err != nil {
		logx.Error("缓存组织成员列表失败:", err)
	}

	return members, nil
}

// AddMembers 批量增加组织成员
func (m *organizationManager) AddMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error) {
	// 1. 验证参数
	if len(memberIDs) == 0 {
		return nil, core.ErrEmptyMemberIDList
	}

	// 2. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. 查询远程组织ID
	remoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err != nil {
		return nil, err
	}

	// 4. 内部调用 GetMembers 获取当前成员列表并校验映射表
	currentMembers, err := m.GetMembers(ctx, tenantID, localOrgID, false)
	if err != nil {
		return nil, fmt.Errorf("获取当前成员列表失败: %w", err)
	}

	// 5. 校验当前成员是否都在本地映射表中（确保之前的同步正常）
	if err := m.validateMemberMappings(ctx, tenantID, currentMembers); err != nil {
		return nil, fmt.Errorf("成员映射校验失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("current_member_count", len(currentMembers)),
		logx.Field("adding_count", len(memberIDs)),
	).Info("添加成员前校验通过")

	// 6. 调用 Skylark API 批量增加成员
	req := &AddMembersRequest{
		MemberIDs: memberIDs,
	}

	addedIDs, err := m.httpClient.addMembers(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrMemberAddFailed, err)
	}

	// 6. 清除成员列表缓存
	m.deleteMemberCache(ctx, tenantID, remoteOrgID)

	return addedIDs, nil
}

// RemoveMembers 批量移除组织成员
func (m *organizationManager) RemoveMembers(ctx context.Context, tenantID, localOrgID string, memberIDs []int) ([]int, error) {
	// 1. 验证参数
	if len(memberIDs) == 0 {
		return nil, core.ErrEmptyMemberIDList
	}

	// 2. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. 查询远程组织ID
	remoteOrgID, err := m.GetRemoteOrgID(ctx, tenantID, localOrgID)
	if err != nil {
		return nil, err
	}

	// 4. 内部调用 GetMembers 获取当前成员列表并校验映射表
	currentMembers, err := m.GetMembers(ctx, tenantID, localOrgID, false)
	if err != nil {
		return nil, fmt.Errorf("获取当前成员列表失败: %w", err)
	}

	// 5. 校验当前成员是否都在本地映射表中（确保之前的同步正常）
	if err := m.validateMemberMappings(ctx, tenantID, currentMembers); err != nil {
		return nil, fmt.Errorf("成员映射校验失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("current_member_count", len(currentMembers)),
		logx.Field("removing_count", len(memberIDs)),
	).Info("移除成员前校验通过")

	// 6. 调用 Skylark API 批量移除成员
	req := &RemoveMembersRequest{
		MemberIDs: memberIDs,
	}

	removedIDs, err := m.httpClient.removeMembers(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrMemberRemoveFailed, err)
	}

	// 6. 清除成员列表缓存
	m.deleteMemberCache(ctx, tenantID, remoteOrgID)

	return removedIDs, nil
}

// ========== UpdateOrganization 实现 ==========

// UpdateOrganization 更新组织信息
func (m *organizationManager) UpdateOrganization(ctx context.Context, req *core.UpdateOrganizationRequest) error {
	// 1. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, req.TenantID)
	if err != nil {
		return err
	}

	// 2. 查询远程组织ID
	remoteOrgID, err := m.GetRemoteOrgID(ctx, req.TenantID, req.LocalOrgID)
	if err != nil {
		return err
	}

	// 3. 内部调用 GetMembers 校验组织同步状态并校验映射表
	currentMembers, err := m.GetMembers(ctx, req.TenantID, req.LocalOrgID, false)
	if err != nil {
		return fmt.Errorf("获取当前成员列表失败: %w", err)
	}

	// 4. 校验当前成员是否都在本地映射表中（确保之前的同步正常）
	if err := m.validateMemberMappings(ctx, req.TenantID, currentMembers); err != nil {
		return fmt.Errorf("成员映射校验失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("current_member_count", len(currentMembers)),
	).Info("更新组织前校验通过")

	// 5. 更新管理员（如果指定）
	if req.ManagerID != "" {
		if err := m.updateOrganizationManager(ctx, req.TenantID, req.LocalOrgID, remoteOrgID, req.ManagerID, platformConfig); err != nil {
			return err
		}
	}

	// 6. 更新组织名称（如果指定）
	if req.Name != "" {
		name := req.Name
		updateReq := &UpdateOrganizationBasicInfoRequest{
			Name: &name,
		}
		if err := m.httpClient.updateOrganizationBasicInfo(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, updateReq); err != nil {
			return fmt.Errorf("%w: %v", core.ErrOrganizationUpdateFailed, err)
		}
	}

	return nil
}
