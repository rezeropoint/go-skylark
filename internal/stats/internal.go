package stats

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/logx"
)

// getAllEventConfigIDs 获取租户下所有已启用的事件配置ID
// 当 EventConfigIDs 为空时调用此方法，查询租户下所有事件
func (m *statsManager) getAllEventConfigIDs(ctx context.Context, tenantID string) ([]string, error) {
	query := `
		SELECT id FROM event_configs
		WHERE tenant_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`

	var ids []string
	err := m.dbConn.QueryRowsCtx(ctx, &ids, query, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询租户事件配置列表失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "stats_manager"),
		logx.Field("operation", "get_all_event_config_ids"),
		logx.Field("tenant_id", tenantID),
		logx.Field("count", len(ids)),
	).Info("获取租户下所有事件配置ID成功")

	return ids, nil
}

// calculateAllowedOrgValues 计算用户有权访问的远程组织值
// 通过 OrgMapping 映射：UserOrgIDs → RemoteOrgValues
//
// 返回值说明：
//   - 如果没有配置任何组织映射：返回空列表（不进行组织过滤，查询所有数据）
//   - 如果有映射但用户组织不在映射中：返回空列表（不进行组织过滤，查询所有数据）
//   - 如果有映射且用户组织在映射中：返回匹配的远程组织值列表（按映射过滤数据）
func (m *statsManager) calculateAllowedOrgValues(userOrgIDs []string, orgMappings []*core.OrgMapping) ([]string, error) {
	// 如果没有配置组织映射，返回 nil（表示不进行组织过滤）
	if len(orgMappings) == 0 {
		return nil, nil
	}

	// 采用启发式以降低常数与内存占用：
	// - 当用户组织数更少：构建 userOrgSet，遍历 orgMappings
	// - 当映射项更少：构建 orgMap（LocalOrgID -> RemoteOrgValue），遍历 userOrgIDs

	// 使用 struct{} 作为集合值以降低内存占用
	allowedValues := make(map[string]struct{})

	if len(userOrgIDs) <= len(orgMappings) {
		// 构建用户组织ID集合
		userOrgSet := make(map[string]struct{}, len(userOrgIDs))
		for _, orgID := range userOrgIDs {
			userOrgSet[orgID] = struct{}{}
		}

		// 遍历映射，匹配用户组织
		for _, mapping := range orgMappings {
			if _, ok := userOrgSet[mapping.LocalOrgID]; ok {
				allowedValues[mapping.RemoteOrgValue] = struct{}{}
			}
		}
	} else {
		// 构建 LocalOrgID -> RemoteOrgValue 的映射
		orgMap := make(map[string]string, len(orgMappings))
		for _, mapping := range orgMappings {
			orgMap[mapping.LocalOrgID] = mapping.RemoteOrgValue
		}

		// 遍历用户组织，查找映射
		for _, orgID := range userOrgIDs {
			if remote, ok := orgMap[orgID]; ok {
				allowedValues[remote] = struct{}{}
			}
		}
	}

	// 如果用户所属组织没有任何映射，返回 nil（表示不进行组织过滤）
	if len(allowedValues) == 0 {
		return nil, nil
	}

	// 转换为切片
	result := make([]string, 0, len(allowedValues))
	for value := range allowedValues {
		result = append(result, value)
	}

	return result, nil
}
