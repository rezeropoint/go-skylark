package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

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
	if eventConfig.OrgFieldName.Valid && len(allowedOrgValues) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("a.%s = ANY($%d)", quoteFieldName(eventConfig.OrgFieldName.String), argIndex))
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
			if eventConfig.OrgFieldName.Valid && len(allowedOrgValues) > 0 {
				subWhereParts = append(subWhereParts, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(eventConfig.OrgFieldName.String), argIndex))
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
	if eventConfig.OrgFieldName.Valid && len(allowedOrgValues) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(eventConfig.OrgFieldName.String), argIndex))
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
			if eventConfig.OrgFieldName.Valid && len(allowedOrgValues) > 0 {
				subWhereParts = append(subWhereParts, fmt.Sprintf("%s = ANY($%d)", quoteFieldName(eventConfig.OrgFieldName.String), argIndex))
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

// assignmentRow 用于扫描 Assignment 查询结果
type assignmentRow struct {
	AssignmentID int            `db:"slp_assignment_id"`
	JourneyID    int            `db:"slp_journey_id"`
	Status       string         `db:"slp_status"`
	VertexID     int            `db:"slp_vertex_id"`
	VertexName   sql.NullString `db:"vertex_name"`
	VertexAlias  sql.NullString `db:"vertex_alias"`
	UserID       sql.NullString `db:"slp_user_id"`
	CreatedAt    time.Time      `db:"slp_created_at"`
	UpdatedAt    time.Time      `db:"slp_updated_at"`
	BusinessData string         `db:"business_data"` // JSON 字符串
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
