package organization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"
	"github.com/zeromicro/go-zero/core/logx"
)

// createOrganization 调用 Skylark API 创建组织
func (c *skylarkHTTPClient) createOrganization(ctx context.Context, apiBaseURL, apiToken string, req *CreateOrganizationRequest) (*OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations", apiBaseURL)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（自动处理错误和JSON解析）
	var orgResp OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgResp); err != nil {
		return nil, err
	}

	return &orgResp, nil
}

// deleteOrganization 调用 Skylark API 删除组织
func (c *skylarkHTTPClient) deleteOrganization(ctx context.Context, apiBaseURL, apiToken string, remoteOrgID int) error {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d", apiBaseURL, remoteOrgID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（DELETE请求无响应体，传入nil）
	return httputils.ReadJSONResponse(resp, nil)
}

// listOrganizations 调用 Skylark API 列出所有组织
func (c *skylarkHTTPClient) listOrganizations(ctx context.Context, apiBaseURL, apiToken string) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations", apiBaseURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// listRootOrganizations 调用 Skylark API 列出根组织
func (c *skylarkHTTPClient) listRootOrganizations(ctx context.Context, apiBaseURL, apiToken string) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/roots", apiBaseURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// listChildOrganizations 调用 Skylark API 列出子组织
func (c *skylarkHTTPClient) listChildOrganizations(ctx context.Context, apiBaseURL, apiToken string, parentRemoteOrgID int) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/children", apiBaseURL, parentRemoteOrgID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// ========== 组织成员管理 HTTP 接口 ==========

// getMembers 调用 Skylark API 获取组织成员
func (c *skylarkHTTPClient) getMembers(ctx context.Context, apiBaseURL, apiToken string, orgID int, withDescendants bool) ([]OrganizationMemberModel, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/members", apiBaseURL, orgID)
	if withDescendants {
		url += "?with_descendants=true"
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var members []OrganizationMemberModel
	if err := httputils.ReadJSONResponse(resp, &members); err != nil {
		return nil, err
	}

	return members, nil
}

// addMembers 调用 Skylark API 批量增加组织成员
func (c *skylarkHTTPClient) addMembers(ctx context.Context, apiBaseURL, apiToken string, orgID int, req *AddMembersRequest) ([]int, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/members/add", apiBaseURL, orgID)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（返回成功添加的成员ID列表）
	var addedIDs []int
	if err := httputils.ReadJSONResponse(resp, &addedIDs); err != nil {
		return nil, err
	}

	return addedIDs, nil
}

// removeMembers 调用 Skylark API 批量移除组织成员
func (c *skylarkHTTPClient) removeMembers(ctx context.Context, apiBaseURL, apiToken string, orgID int, req *RemoveMembersRequest) ([]int, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/members/remove", apiBaseURL, orgID)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（返回成功移除的成员ID列表）
	var removedIDs []int
	if err := httputils.ReadJSONResponse(resp, &removedIDs); err != nil {
		return nil, err
	}

	return removedIDs, nil
}

// ========== UpdateOrganization 内部使用的 HTTP 方法 ==========

// updateOrganizationBasicInfo 调用 Skylark API 更新组织基本信息（内部使用）
func (c *skylarkHTTPClient) updateOrganizationBasicInfo(ctx context.Context, apiBaseURL, apiToken string, orgID int, req *UpdateOrganizationBasicInfoRequest) error {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d", apiBaseURL, orgID)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（PATCH请求通常返回更新后的对象，但我们不需要解析）
	return httputils.ReadJSONResponse(resp, nil)
}

// getAdministrators 调用 Skylark API 获取组织管理员列表（内部使用）
func (c *skylarkHTTPClient) getAdministrators(ctx context.Context, apiBaseURL, apiToken string, orgID int) ([]OrganizationAdministratorModel, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/administrators", apiBaseURL, orgID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var admins []OrganizationAdministratorModel
	if err := httputils.ReadJSONResponse(resp, &admins); err != nil {
		return nil, err
	}

	// 设置组织ID（从请求上下文获取）
	for i := range admins {
		admins[i].OrganizationID = orgID
	}

	return admins, nil
}

// addAdministrator 调用 Skylark API 增加组织管理员（内部使用）
func (c *skylarkHTTPClient) addAdministrator(ctx context.Context, apiBaseURL, apiToken string, orgID int, req *CreateAdministratorRequest) error {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/administrators", apiBaseURL, orgID)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	return httputils.ReadJSONResponse(resp, nil)
}

// deleteAdministrator 调用 Skylark API 删除组织管理员（内部使用）
func (c *skylarkHTTPClient) deleteAdministrator(ctx context.Context, apiBaseURL, apiToken string, orgID, adminID int) error {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/administrators/%d", apiBaseURL, orgID, adminID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（DELETE请求无响应体，传入nil）
	return httputils.ReadJSONResponse(resp, nil)
}

// ========== 组织管理器内部方法（未导出） ==========

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

// deleteMemberCache 删除成员列表缓存（包含两种场景）
func (m *organizationManager) deleteMemberCache(ctx context.Context, tenantID string, remoteOrgID int) {
	// 删除不包含子孙组织的缓存
	if err := m.cache.DeleteOrgMembers(ctx, tenantID, remoteOrgID, false); err != nil {
		logx.Error("删除成员缓存失败:", err)
	}

	// 删除包含子孙组织的缓存
	if err := m.cache.DeleteOrgMembers(ctx, tenantID, remoteOrgID, true); err != nil {
		logx.Error("删除成员缓存失败:", err)
	}
}

// updateOrganizationManager 更新组织管理员（内部方法，使用分布式锁）
// 说明：管理员权限ID固定为1（"管理员"权限）
func (m *organizationManager) updateOrganizationManager(ctx context.Context, tenantID, localOrgID string, remoteOrgID int, newManagerID string, platformConfig *core.SkylarkAPIConfig) error {
	// 构建分布式锁key
	lockKey := fmt.Sprintf("skylark:lock:org_manager:%s:%s", tenantID, localOrgID)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano()) // 使用时间戳作为锁值
	lockTTL := 30                                         // 锁超时时间30秒

	// 获取分布式锁
	locked, err := m.cache.AcquireLock(ctx, lockKey, lockValue, lockTTL)
	if err != nil {
		return fmt.Errorf("获取分布式锁失败: %w", err)
	}
	if !locked {
		return core.ErrOrganizationManagerUpdateConflict // 其他请求正在更新管理员
	}

	// 确保释放锁（defer）
	defer func() {
		if delErr := m.cache.ReleaseLock(ctx, lockKey, lockValue); delErr != nil {
			logx.Error("释放分布式锁失败:", delErr)
		}
	}()

	// 1. 查询当前所有管理员
	currentAdmins, err := m.httpClient.getAdministrators(ctx, platformConfig.App, platformConfig.Token, remoteOrgID)
	if err != nil {
		return fmt.Errorf("%w: 查询当前管理员失败: %v", core.ErrOrganizationManagerUpdateFailed, err)
	}

	// 2. 转换新管理员的本地用户ID为远程用户ID
	remoteUserIDs, err := m.getRemoteUserIDs(ctx, tenantID, []string{newManagerID})
	if err != nil {
		return fmt.Errorf("%w: 查询新管理员的远程用户ID失败: %v", core.ErrOrganizationManagerUpdateFailed, err)
	}
	if len(remoteUserIDs) == 0 {
		return fmt.Errorf("%w: 新管理员的远程用户ID不存在", core.ErrUserMappingNotFound)
	}
	newRemoteUserID := remoteUserIDs[0]

	// 3. 添加新管理员（权限ID固定为1"管理员"权限）
	addReq := &CreateAdministratorRequest{
		UserID:   newRemoteUserID,
		AccessID: 1, // 固定使用1"管理员"权限
	}
	if err := m.httpClient.addAdministrator(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, addReq); err != nil {
		return fmt.Errorf("%w: 添加新管理员失败: %v", core.ErrOrganizationManagerUpdateFailed, err)
	}

	// 4. 删除所有旧管理员
	for _, admin := range currentAdmins {
		if err := m.httpClient.deleteAdministrator(ctx, platformConfig.App, platformConfig.Token, remoteOrgID, admin.ID); err != nil {
			logx.Errorf("删除旧管理员失败（管理员ID:%d）: %v", admin.ID, err)
			// 不返回错误，继续删除其他管理员
		}
	}

	// 5. 清理管理员缓存（如果有使用缓存）
	// 注意：当前实现中没有管理员缓存，所以无需清理

	return nil
}

// validateMemberMappings 校验组织成员是否都在本地映射表中
// 用途：确保执行修改操作前，组织成员的同步状态正常
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - members: 组织成员列表
//
// 返回：
//   - error: 如果有成员不在映射表中，返回错误
func (m *organizationManager) validateMemberMappings(ctx context.Context, tenantID string, members []*core.OrganizationMember) error {
	if len(members) == 0 {
		// 空成员列表，校验通过
		return nil
	}

	// 1. 提取所有远程用户ID，构建映射 map
	userIDMapping := make(map[int]string)
	for _, member := range members {
		userIDMapping[member.ID] = "" // value 将由 FillLocalUserIDMap 填充
	}

	// 2. 调用 fillLocalUserIDMap 批量查询并校验映射
	if err := m.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
		return fmt.Errorf("组织成员映射校验失败，存在未同步的用户: %w", err)
	}

	// 3. 校验通过
	return nil
}
