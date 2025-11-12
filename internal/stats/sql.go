package stats

import (
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
)

// buildDurationStatsSQL 构建处理时长统计SQL
// 说明：统计Journey维度的处理时长（从第一个Assignment创建到最后一个Assignment更新）
// 返回：SQL语句、参数列表
func buildDurationStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（使用CTE先按Journey聚合，再统计）
	query := fmt.Sprintf(`
		WITH journey_stats AS (
			SELECT
				slp_journey_id,
				MIN(slp_created_at) as start_time,
				MAX(slp_updated_at) as end_time,
				MAX(slp_status) as final_status
			FROM %s
			WHERE %s
			GROUP BY slp_journey_id
		)
		SELECT
			AVG(EXTRACT(EPOCH FROM (end_time - start_time))) as avg_duration,
			MIN(EXTRACT(EPOCH FROM (end_time - start_time))) as min_duration,
			MAX(EXTRACT(EPOCH FROM (end_time - start_time))) as max_duration,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE final_status = 'completed') as completed_count
		FROM journey_stats
	`, tableName, whereConditions)

	return query, args
}

// buildStatusStatsSQL 构建状态统计SQL
// 说明：统计各状态的Journey数量，对于未完成事件按节点数区分待处理和处理中的虚拟状态
// 虚拟状态定义：
//   - pending（待处理）：只有1个节点的未完成事件
//   - processing（处理中）：有多个节点的未完成事件
//
// 返回：SQL语句、参数列表
func buildStatusStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（先聚合Journey，计算节点数和最新状态，再根据节点数区分虚拟状态）
	query := fmt.Sprintf(`
		WITH journey_aggregated AS (
			SELECT
				slp_journey_id,
				COUNT(DISTINCT slp_vertex_id) as node_count,
				MAX(slp_status) as final_status
			FROM %s
			WHERE %s
			GROUP BY slp_journey_id
		),
		journey_virtual_status AS (
			SELECT
				slp_journey_id,
				node_count,
				final_status,
				CASE
					-- 未完成事件：根据节点数区分虚拟状态
					WHEN final_status NOT IN ('completed', 'rejected')
						AND node_count = 1 THEN 'pending'
					WHEN final_status NOT IN ('completed', 'rejected')
						AND node_count > 1 THEN 'processing'
					-- 已完成事件：保持原状态
					ELSE final_status
				END as virtual_status
			FROM journey_aggregated
		)
		SELECT
			virtual_status as slp_status,
			COUNT(*) as count
		FROM journey_virtual_status
		GROUP BY virtual_status
		ORDER BY count DESC
	`, tableName, whereConditions)

	return query, args
}

// buildTrendStatsSQL 构建趋势统计SQL
// 说明：按日/周/月统计Journey数量和完成率
// 返回：SQL语句、参数列表
func buildTrendStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 确定时间聚合格式
	var dateFormat string
	switch req.GroupBy {
	case "week":
		dateFormat = "TO_CHAR(slp_created_at, 'IYYY-IW')" // ISO周格式：2025-W03
	case "month":
		dateFormat = "TO_CHAR(slp_created_at, 'YYYY-MM')" // 年-月格式：2025-01
	default: // "day" 或默认
		dateFormat = "DATE(slp_created_at)" // 日期格式：2025-01-15
	}

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选（必须有）
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（先聚合Journey，再按时间统计）
	query := fmt.Sprintf(`
		WITH journey_latest AS (
			SELECT DISTINCT ON (slp_journey_id)
				slp_journey_id,
				slp_status,
				slp_created_at
			FROM %s
			WHERE %s
			ORDER BY slp_journey_id, slp_created_at DESC
		),
		daily_stats AS (
			SELECT
				%s as date,
				slp_journey_id,
				slp_status
			FROM journey_latest
		)
		SELECT
			date::TEXT,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE slp_status = 'completed') as completed_count
		FROM daily_stats
		GROUP BY date
		ORDER BY date
	`, tableName, whereConditions, dateFormat)

	return query, args
}

// buildNodeStatsSQL 构建节点统计SQL
// 说明：统计各节点的Journey数量和平均处理时长
// 返回：SQL语句、参数列表
func buildNodeStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("a.%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("a.slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("a.slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（关联vertices表获取节点名称）
	query := fmt.Sprintf(`
		SELECT
			a.slp_vertex_id,
			COALESCE(v.alias_name, v.name, '') as vertex_name,
			COUNT(DISTINCT a.slp_journey_id) as count,
			AVG(EXTRACT(EPOCH FROM (a.slp_updated_at - a.slp_created_at))) as avg_duration
		FROM %s a
		LEFT JOIN vertices v ON a.slp_vertex_id = v.id
		WHERE %s
		GROUP BY a.slp_vertex_id, v.alias_name, v.name
		ORDER BY count DESC
	`, tableName, whereConditions)

	return query, args
}

// buildUserStatsSQL 构建处理人统计SQL
// 说明：统计处理人的Journey数量（Top N）
// 返回：SQL语句、参数列表
func buildUserStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 确定Top N数量（默认10）
	topN := req.TopN
	if topN <= 0 {
		topN = 10
	}

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 过滤掉NULL的用户ID
	if whereConditions != "" {
		whereConditions += " AND "
	}
	whereConditions += "slp_user_id IS NOT NULL"

	// 构建SQL
	query := fmt.Sprintf(`
		SELECT
			slp_user_id,
			COUNT(DISTINCT slp_journey_id) as count
		FROM %s
		WHERE %s
		GROUP BY slp_user_id
		ORDER BY count DESC
		LIMIT %d
	`, tableName, whereConditions, topN)

	return query, args
}

// buildOrgStatsSQL 构建组织统计SQL
// 说明：统计各组织的Journey数量和平均处理时长
// 返回：SQL语句、参数列表
func buildOrgStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 必须配置组织字段才能统计
	if eventConfig.OrgFieldName == nil {
		return "", nil
	}

	orgFieldName := quoteFieldName(*eventConfig.OrgFieldName)

	// 构建权限WHERE条件
	var whereConditions string
	if len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", orgFieldName, argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（先按Journey聚合，再按组织统计）
	query := fmt.Sprintf(`
		WITH org_journey_stats AS (
			SELECT
				%s as org_value,
				slp_journey_id,
				MIN(slp_created_at) as start_time,
				MAX(slp_updated_at) as end_time
			FROM %s
			WHERE %s
			GROUP BY %s, slp_journey_id
		)
		SELECT
			org_value,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (end_time - start_time))) as avg_duration
		FROM org_journey_stats
		GROUP BY org_value
		ORDER BY count DESC
	`, orgFieldName, tableName, whereConditions, orgFieldName)

	return query, args
}

// buildPendingStatsSQL 构建待处理事件统计SQL（实时查询）
// 说明：按Journey聚合，统计只有1个节点的事件（未开始）和多个节点的事件（处理中）
// 未开始定义：Journey只有1个不同的vertex_id（即只有发起节点）
// 返回：SQL语句、参数列表
func buildPendingStatsSQL(eventConfig *core.EventConfig, allowedOrgValues []string, req *core.StatsCriteria) (string, []interface{}) {
	tableName := eventConfig.GetRemoteTableName()
	args := []interface{}{}
	argIndex := 1

	// 构建权限WHERE条件
	var whereConditions string
	if eventConfig.OrgFieldName != nil && len(allowedOrgValues) > 0 {
		whereConditions = fmt.Sprintf("%s = ANY($%d)", quoteFieldName(*eventConfig.OrgFieldName), argIndex)
		args = append(args, pq.Array(allowedOrgValues))
		argIndex++
	}

	// 添加日期范围筛选
	if req.DateFrom != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at >= $%d", argIndex)
		args = append(args, *req.DateFrom)
		argIndex++
	}
	if req.DateTo != nil {
		if whereConditions != "" {
			whereConditions += " AND "
		}
		whereConditions += fmt.Sprintf("slp_created_at <= $%d", argIndex)
		args = append(args, *req.DateTo)
		argIndex++
	}

	// 如果没有任何筛选条件，添加默认条件
	if whereConditions == "" {
		whereConditions = "1=1"
	}

	// 构建SQL（使用CTE先按Journey聚合节点数，再统计）
	// 排除已完成和已拒绝的事件，只统计待处理的
	// 返回包含事件配置ID和事件名称的结果
	query := fmt.Sprintf(`
		WITH journey_nodes AS (
			SELECT
				slp_journey_id,
				COUNT(DISTINCT slp_vertex_id) as node_count,
				MAX(slp_status) as final_status
			FROM %s
			WHERE %s
			  AND slp_status NOT IN ('completed', 'rejected')
			GROUP BY slp_journey_id
		)
		SELECT
			'%s' as event_config_id,
			'%s' as event_name,
			COUNT(*) FILTER (WHERE node_count = 1) as pending_count,
			COUNT(*) FILTER (WHERE node_count > 1) as processing_count,
			COUNT(*) as total
		FROM journey_nodes
	`, tableName, whereConditions, eventConfig.ID, eventConfig.Name)

	return query, args
}
