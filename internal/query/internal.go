package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
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

// SQL 构建与查询

// buildQuerySQLWithConfig 构建事件列表查询SQL（DISTINCT ON Journey聚合 + vertices表关联）- 新版本，接收 EventConfig 参数
func (m *queryManager) buildQuerySQLWithConfig(req *core.QueryRequest, eventConfig *core.EventConfig, visibleFields []*core.FieldConfig, allowedOrgValues []string) (string, []interface{}, error) {
	var sqlBuilder strings.Builder
	var args []interface{}
	argIndex := 1

	// 1. 构建 SELECT 子句（系统字段 + 节点名称 + 可见业务字段）
	selectFields := []string{
		"a." + quoteFieldName("slp_journey_id"),
		"a." + quoteFieldName("slp_assignment_id"), // 用于排序去重，但不返回给前端
		"a." + quoteFieldName("slp_status"),
		"a." + quoteFieldName("slp_vertex_id"),
		"COALESCE(v.alias_name, v.name, '') as vertex_name", // 节点名称（优先别名）
		"a." + quoteFieldName("slp_created_at"),             // 用于排序，但不返回给前端
	}
	for _, field := range visibleFields {
		selectFields = append(selectFields, "a."+quoteFieldName(field.FieldName))
	}

	// 2. 开始构建内层查询（DISTINCT ON + LEFT JOIN vertices）
	sqlBuilder.WriteString("SELECT * FROM (\n")
	sqlBuilder.WriteString(fmt.Sprintf("    SELECT DISTINCT ON (a.%s)\n", quoteFieldName("slp_journey_id")))
	sqlBuilder.WriteString("        ")
	sqlBuilder.WriteString(strings.Join(selectFields, ",\n        "))
	sqlBuilder.WriteString("\n    FROM ")
	sqlBuilder.WriteString(eventConfig.GetRemoteTableName())
	sqlBuilder.WriteString(" a\n")
	sqlBuilder.WriteString("    LEFT JOIN vertices v ON a.slp_vertex_id = v.id\n")

	// 3. 构建 WHERE 子句
	whereClauses := []string{}

	// 3.1 组织权限过滤（如果配置了组织字段且有映射）
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("a.%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex))
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 3.2 状态筛选（支持虚拟状态 pending/processing 和实际状态）
	if len(req.Status) > 0 {
		// 检测是否包含虚拟状态 'pending' 或 'processing'
		hasPending := false
		hasProcessing := false
		realStatuses := []string{}

		for _, status := range req.Status {
			if status == "pending" {
				hasPending = true
			} else if status == "processing" {
				hasProcessing = true
			} else {
				realStatuses = append(realStatuses, status)
			}
		}

		// 如果包含虚拟状态，使用子查询（基于 node_count）
		if hasPending || hasProcessing {
			// 构建子查询的WHERE条件
			subWhereParts := []string{"slp_status NOT IN ('completed', 'rejected')"}

			// 子查询也需要应用组织权限过滤
			if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
				subWhereParts = append(subWhereParts, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex))
				args = append(args, pq.Array(allowedOrgValues))
				argIndex++
			}

			subWhereClause := strings.Join(subWhereParts, " AND ")

			// 构建 HAVING 子句
			var havingClause string
			if hasPending && hasProcessing {
				// 查询所有未完成的（不需要 HAVING）
				havingClause = ""
			} else if hasPending {
				havingClause = "HAVING COUNT(DISTINCT slp_vertex_id) = 1"
			} else { // hasProcessing
				havingClause = "HAVING COUNT(DISTINCT slp_vertex_id) > 1"
			}

			// 构建子查询
			if havingClause != "" {
				subQuery := fmt.Sprintf(`a.%s IN (
					SELECT slp_journey_id
					FROM %s
					WHERE %s
					GROUP BY slp_journey_id
					%s
				)`, quoteFieldName("slp_journey_id"), eventConfig.GetRemoteTableName(), subWhereClause, havingClause)
				whereClauses = append(whereClauses, subQuery)
			} else {
				// pending + processing 的情况，等价于所有未完成的事件
				subQuery := fmt.Sprintf(`a.%s IN (
					SELECT DISTINCT slp_journey_id
					FROM %s
					WHERE %s
				)`, quoteFieldName("slp_journey_id"), eventConfig.GetRemoteTableName(), subWhereClause)
				whereClauses = append(whereClauses, subQuery)
			}
		}

		// 如果还有实际的 Skylark 状态，添加状态过滤
		if len(realStatuses) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("a.%s = ANY($%d)", quoteFieldName("slp_status"), argIndex))
			args = append(args, pq.Array(realStatuses))
			argIndex++
		}
	}

	// 3.3 关键词搜索（对所有可搜索字段使用 ILIKE）
	if req.Keyword != "" {
		searchClauses := []string{}
		keyword := "%" + req.Keyword + "%"
		for _, field := range visibleFields {
			if field.IsSearchable {
				searchClauses = append(searchClauses, fmt.Sprintf("a.%s::text ILIKE $%d", quoteFieldName(field.FieldName), argIndex))
			}
		}
		if len(searchClauses) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(searchClauses, " OR ")+")")
			args = append(args, keyword)
			argIndex++
		}
	}

	// 添加 WHERE 子句
	if len(whereClauses) > 0 {
		sqlBuilder.WriteString("    WHERE ")
		sqlBuilder.WriteString(strings.Join(whereClauses, " AND "))
		sqlBuilder.WriteString("\n")
	}

	// 4. 内层 ORDER BY（DISTINCT ON 的关键）
	sqlBuilder.WriteString(fmt.Sprintf("    ORDER BY a.%s, a.%s DESC\n", quoteFieldName("slp_journey_id"), quoteFieldName("slp_created_at")))
	sqlBuilder.WriteString(") t\n")

	// 5. 外层 ORDER BY（最终排序）
	sortField := "slp_created_at"
	sortOrder := "DESC"
	if req.SortField != "" {
		sortField = req.SortField
	}
	if req.SortOrder != "" && strings.ToUpper(req.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}
	sqlBuilder.WriteString(fmt.Sprintf("ORDER BY %s %s\n", quoteFieldName(sortField), sortOrder))

	// 6. 分页（LIMIT + OFFSET）
	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize
	sqlBuilder.WriteString(fmt.Sprintf("LIMIT $%d OFFSET $%d", argIndex, argIndex+1))
	args = append(args, limit, offset)

	return sqlBuilder.String(), args, nil
}

// countJourneysWithConfig 查询总记录数（Journey数量）- 新版本，接收 EventConfig 参数
func (m *queryManager) countJourneysWithConfig(ctx context.Context, remoteDB sqlx.SqlConn, req *core.QueryRequest, eventConfig *core.EventConfig, allowedOrgValues []string) (int64, error) {
	var sqlBuilder strings.Builder
	var args []interface{}
	argIndex := 1

	// 构建 COUNT 查询（与主查询的 WHERE 条件相同）
	sqlBuilder.WriteString(fmt.Sprintf("SELECT COUNT(DISTINCT %s) FROM ", quoteFieldName("slp_journey_id")))
	sqlBuilder.WriteString(eventConfig.GetRemoteTableName())

	whereClauses := []string{}

	// 组织权限过滤
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex))
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 状态筛选（支持虚拟状态 pending/processing 和实际状态）
	if len(req.Status) > 0 {
		// 检测是否包含虚拟状态 'pending' 或 'processing'
		hasPending := false
		hasProcessing := false
		realStatuses := []string{}

		for _, status := range req.Status {
			if status == "pending" {
				hasPending = true
			} else if status == "processing" {
				hasProcessing = true
			} else {
				realStatuses = append(realStatuses, status)
			}
		}

		// 如果包含虚拟状态，使用子查询（基于 node_count）
		if hasPending || hasProcessing {
			// 构建子查询的WHERE条件
			subWhereParts := []string{"slp_status NOT IN ('completed', 'rejected')"}

			// 子查询也需要应用组织权限过滤
			if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
				subWhereParts = append(subWhereParts, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex))
				args = append(args, pq.Array(allowedOrgValues))
				argIndex++
			}

			subWhereClause := strings.Join(subWhereParts, " AND ")

			// 构建 HAVING 子句
			var havingClause string
			if hasPending && hasProcessing {
				// 查询所有未完成的（不需要 HAVING）
				havingClause = ""
			} else if hasPending {
				havingClause = "HAVING COUNT(DISTINCT slp_vertex_id) = 1"
			} else { // hasProcessing
				havingClause = "HAVING COUNT(DISTINCT slp_vertex_id) > 1"
			}

			// 构建子查询
			if havingClause != "" {
				subQuery := fmt.Sprintf(`%s IN (
					SELECT slp_journey_id
					FROM %s
					WHERE %s
					GROUP BY slp_journey_id
					%s
				)`, quoteFieldName("slp_journey_id"), eventConfig.GetRemoteTableName(), subWhereClause, havingClause)
				whereClauses = append(whereClauses, subQuery)
			} else {
				// pending + processing 的情况，等价于所有未完成的事件
				subQuery := fmt.Sprintf(`%s IN (
					SELECT DISTINCT slp_journey_id
					FROM %s
					WHERE %s
				)`, quoteFieldName("slp_journey_id"), eventConfig.GetRemoteTableName(), subWhereClause)
				whereClauses = append(whereClauses, subQuery)
			}
		}

		// 如果还有实际的 Skylark 状态，添加状态过滤
		if len(realStatuses) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ANY($%d)", quoteFieldName("slp_status"), argIndex))
			args = append(args, pq.Array(realStatuses))
			argIndex++
		}
	}

	// 关键词搜索（注意：这里没有传入 visibleFields，无法精确匹配，但为了向后兼容，暂时保留空实现）
	if req.Keyword != "" {
		// TODO: 需要传入 visibleFields 才能正确构建搜索条件
		// 当前版本暂时不支持精确的关键词搜索计数
	}

	if len(whereClauses) > 0 {
		sqlBuilder.WriteString(" WHERE ")
		sqlBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	// 执行查询
	var total int64
	err := remoteDB.QueryRowCtx(ctx, &total, sqlBuilder.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("查询总记录数失败: %w", err)
	}

	return total, nil
}

// queryJourneyAssignments 查询 Journey 的所有 Assignment
func (m *queryManager) queryJourneyAssignments(ctx context.Context, remoteDB sqlx.SqlConn, eventConfig *core.EventConfig, journeyID int) ([]*assignmentRow, error) {
	// 构建查询SQL（返回所有Assignment + vertices信息 + 业务数据JSON）
	query := fmt.Sprintf(`
		SELECT
			a.slp_assignment_id,
			a.slp_journey_id,
			a.slp_status,
			a.slp_vertex_id,
			v.name as vertex_name,
			v.alias_name as vertex_alias,
			a.slp_user_id,
			a.slp_created_at,
			a.slp_updated_at,
			row_to_json(a)::text as business_data
		FROM %s a
		LEFT JOIN vertices v ON a.slp_vertex_id = v.id
		WHERE a.slp_journey_id = $1
		ORDER BY a.slp_created_at ASC
	`, eventConfig.GetRemoteTableName())

	var assignments []*assignmentRow
	err := remoteDB.QueryRowsCtx(ctx, &assignments, query, journeyID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询 Journey Assignment 失败: %w", err)
	}

	return assignments, nil
}
