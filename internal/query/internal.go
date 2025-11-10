package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 权限计算

// calculateAllowedOrgValues 计算用户有权访问的远程组织值
// 通过 OrgMapping 映射：UserOrgIDs → RemoteOrgValues
//
// 返回值说明：
//   - 如果没有配置任何组织映射：返回空列表（不进行组织过滤，查询所有数据）
//   - 如果有映射但用户组织不在映射中：返回空列表（不进行组织过滤，查询所有数据）
//   - 如果有映射且用户组织在映射中：返回匹配的远程组织值列表（按映射过滤数据）
func (m *queryManager) calculateAllowedOrgValues(userOrgIDs []string, orgMappings []*core.OrgMapping) ([]string, error) {
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

// 查询结果解析

// queryRow 用于接收单行 JSON 数据
type queryRow struct {
	Data string `db:"row_data"`
}

// executeQueryAndParse 执行查询并解析 JSON 结果
func (m *queryManager) executeQueryAndParse(ctx context.Context, remoteDB sqlx.SqlConn, querySQL string, queryArgs []interface{}) ([]map[string]interface{}, error) {
	// 包装查询，使用 row_to_json 将结果转换为 JSON
	wrappedSQL := fmt.Sprintf("SELECT row_to_json(t)::text as row_data FROM (%s) t", querySQL)

	// 执行查询
	var rows []*queryRow
	err := remoteDB.QueryRowsCtx(ctx, &rows, wrappedSQL, queryArgs...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}

	// 解析 JSON 为 map 并处理字段转换
	records := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(row.Data), &record); err != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}

		// 翻译状态字段
		if status, ok := record["slp_status"].(string); ok {
			record["slp_status"] = core.TranslateStatus(status)
		}

		// 移除不需要返回给前端的字段
		delete(record, "slp_assignment_id") // Assignment ID 不返回给前端
		delete(record, "slp_created_at")    // 创建时间仅用于排序
		delete(record, "slp_vertex_id")     // 节点ID已被 vertex_name 替代

		records = append(records, record)
	}

	return records, nil
}
