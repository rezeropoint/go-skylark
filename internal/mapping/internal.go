package mapping

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// validateOrgExists 验证组织是否存在
// 查询 organizations 表，验证组织ID是否存在且状态为 active，未被软删除
// 参数：
//   - ctx: 上下文
//   - db: 数据库连接
//   - orgID: 组织ID
//   - tenantID: 租户ID
//
// 返回：
//   - error: 如果组织不存在或验证失败
func (m *mappingManager) validateOrgExists(ctx context.Context, orgID, tenantID string) error {
	query := `
		SELECT id FROM organizations
		WHERE id = $1 AND tenant_id = $2
		  AND status = 'active'
		  AND deleted_at IS NULL
	`

	var existingID string
	err := m.dbConn.QueryRowCtx(ctx, &existingID, query, orgID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: 组织ID %s 不存在或状态不可用", core.ErrInvalidLocalOrgID, orgID)
		}
		return fmt.Errorf("查询组织失败: %w", err)
	}

	return nil
}

// checkMappingExists 检查映射是否已存在
// 验证 (tenant_id, remote_org_value) 唯一约束
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - remoteOrgValue: 远程组织值
//   - excludeID: 排除的映射ID（用于更新操作，为空表示创建操作）
//
// 返回：
//   - error: 如果映射已存在
func (m *mappingManager) checkMappingExists(ctx context.Context, tenantID, remoteOrgValue, excludeID string) error {
	var query string
	var args []interface{}

	if excludeID == "" {
		// 创建操作：检查是否存在
		query = `
			SELECT id FROM event_org_mappings
			WHERE tenant_id = $1 AND remote_org_value = $2
		`
		args = []interface{}{tenantID, remoteOrgValue}
	} else {
		// 更新操作：检查除了当前记录外是否存在
		query = `
			SELECT id FROM event_org_mappings
			WHERE tenant_id = $1 AND remote_org_value = $2 AND id != $3
		`
		args = []interface{}{tenantID, remoteOrgValue, excludeID}
	}

	var existingID string
	err := m.dbConn.QueryRowCtx(ctx, &existingID, query, args...)
	if err == nil {
		// 已存在映射
		return fmt.Errorf("%w: 远程组织值 '%s' 已映射到本地组织 (映射ID: %s)", core.ErrOrgMappingExists, remoteOrgValue, existingID)
	} else if err != sql.ErrNoRows {
		// 查询错误
		return fmt.Errorf("检查映射是否存在失败: %w", err)
	}

	// 不存在，返回nil
	return nil
}

// checkLocalOrgIDExists 检查本地组织ID是否已被映射
// 验证 (tenant_id, local_org_id) 唯一约束
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - localOrgID: 本地组织ID
//   - excludeID: 排除的映射ID（用于更新操作，为空表示创建操作）
//
// 返回：
//   - error: 如果映射已存在
func (m *mappingManager) checkLocalOrgIDExists(ctx context.Context, tenantID, localOrgID, excludeID string) error {
	var query string
	var args []interface{}

	if excludeID == "" {
		// 创建操作
		query = `
			SELECT remote_org_value FROM event_org_mappings
			WHERE tenant_id = $1 AND local_org_id = $2
		`
		args = []interface{}{tenantID, localOrgID}
	} else {
		// 更新操作
		query = `
			SELECT remote_org_value FROM event_org_mappings
			WHERE tenant_id = $1 AND local_org_id = $2 AND id != $3
		`
		args = []interface{}{tenantID, localOrgID, excludeID}
	}

	var existingRemoteValue sql.NullString
	err := m.dbConn.QueryRowCtx(ctx, &existingRemoteValue, query, args...)
	if err == nil {
		// 已存在映射
		return fmt.Errorf("%w: 该本地组织已被远程组织值 '%s' 映射", core.ErrLocalOrgIDInUse, existingRemoteValue.String)
	} else if err != sql.ErrNoRows {
		// 查询错误
		return fmt.Errorf("检查本地组织ID映射是否存在失败: %w", err)
	}

	// 不存在，返回nil
	return nil
}
