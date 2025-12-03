package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/rezeropoint/go-skylark/v2/core"

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

// ========== 单个事件统计内部方法 ==========

// getSingleDurationStats 查询单个事件配置的处理时长统计（内部方法）
func (m *statsManager) getSingleDurationStats(ctx context.Context, req *core.StatsCriteria) (*core.DurationStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedDurationStats(ctx, req)
	if err == nil && cached != nil {
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置（含字段）
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 计算组织权限（UserOrgIDs → RemoteOrgValues）
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 5. 构建SQL
	query, args := buildDurationStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询（使用 Model 类型处理 NULL 值）
	var model DurationStatsModel
	err = remoteDB.QueryRowCtx(ctx, &model, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询处理时长统计失败: %w", err)
	}

	// 8. 转换为领域模型
	stats := model.ToDomain()

	// 9. 写入缓存
	_ = m.setCachedDurationStats(ctx, req, stats)

	return stats, nil
}

// getSingleStatusStats 查询单个事件配置的状态统计（内部方法）
func (m *statsManager) getSingleStatusStats(ctx context.Context, req *core.StatsCriteria) (*core.StatusStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedStatusStats(ctx, req)
	if err == nil && cached != nil {
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 5. 构建SQL
	query, args := buildStatusStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询
	type statusRow struct {
		Status string `db:"slp_status"`
		Count  int64  `db:"count"`
	}
	var rows []*statusRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询状态统计失败: %w", err)
	}

	// 8. 处理结果（状态翻译 + 计算占比）
	statusCounts := make([]*core.StatusCount, 0, len(rows))
	var total int64 = 0
	for _, row := range rows {
		total += row.Count
	}

	for _, row := range rows {
		percentage := float64(0)
		if total > 0 {
			percentage = float64(row.Count) / float64(total) * 100
		}

		statusCounts = append(statusCounts, &core.StatusCount{
			Status:     core.TranslateStatus(row.Status), // 翻译为中文
			StatusKey:  row.Status,                       // 保留英文key
			Count:      row.Count,
			Percentage: percentage,
		})
	}

	stats := &core.StatusStats{
		StatusCounts: statusCounts,
		Total:        total,
	}

	// 9. 写入缓存
	_ = m.setCachedStatusStats(ctx, req, stats)

	return stats, nil
}

// getSingleTrendStats 查询单个事件配置的趋势统计（内部方法）
func (m *statsManager) getSingleTrendStats(ctx context.Context, req *core.StatsCriteria) (*core.TrendStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedTrendStats(ctx, req)
	if err == nil && cached != nil {
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 5. 构建SQL
	query, args := buildTrendStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询
	type trendRow struct {
		Date           sql.NullString `db:"date"` // 使用NullString防止NULL值
		TotalCount     int64          `db:"total_count"`
		CompletedCount int64          `db:"completed_count"`
	}
	var rows []*trendRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询趋势统计失败: %w", err)
	}

	// 8. 处理结果（计算完成率）
	timePoints := make([]*core.TrendPoint, 0, len(rows))
	for _, row := range rows {
		completionRate := float64(0)
		if row.TotalCount > 0 {
			completionRate = float64(row.CompletedCount) / float64(row.TotalCount) * 100
		}

		timePoints = append(timePoints, &core.TrendPoint{
			Date:           row.Date.String, // 如果为NULL则为空字符串
			TotalCount:     row.TotalCount,
			CompletedCount: row.CompletedCount,
			CompletionRate: completionRate,
		})
	}

	stats := &core.TrendStats{
		TimePoints: timePoints,
	}

	// 9. 写入缓存
	_ = m.setCachedTrendStats(ctx, req, stats)

	return stats, nil
}

// getSingleNodeStats 查询单个事件配置的节点统计（内部方法）
func (m *statsManager) getSingleNodeStats(ctx context.Context, req *core.StatsCriteria) (*core.NodeStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedNodeStats(ctx, req)
	if err == nil && cached != nil {
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 5. 构建SQL
	query, args := buildNodeStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询
	type nodeRow struct {
		VertexID    int             `db:"slp_vertex_id"`
		VertexName  sql.NullString  `db:"vertex_name"` // 使用NullString防止NULL值
		Count       int64           `db:"count"`
		AvgDuration sql.NullFloat64 `db:"avg_duration"` // 使用NullFloat64防止NULL值
	}
	var rows []*nodeRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询节点统计失败: %w", err)
	}

	// 8. 处理结果
	nodeMetrics := make([]*core.NodeMetric, 0, len(rows))
	for _, row := range rows {
		nodeMetrics = append(nodeMetrics, &core.NodeMetric{
			VertexID:    row.VertexID,
			VertexName:  row.VertexName.String, // 如果为NULL则为空字符串
			Count:       row.Count,
			AvgDuration: row.AvgDuration.Float64, // 如果为NULL则为0.0
		})
	}

	stats := &core.NodeStats{
		NodeMetrics: nodeMetrics,
	}

	// 9. 写入缓存
	_ = m.setCachedNodeStats(ctx, req, stats)

	return stats, nil
}

// getSingleUserStats 查询单个事件配置的处理人统计（内部方法）
func (m *statsManager) getSingleUserStats(ctx context.Context, req *core.StatsCriteria) (*core.UserStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedUserStats(ctx, req)
	if err == nil && cached != nil {
		// 批量转换用户ID（远程ID → 本地ID）
		if err := convertUserStatsUserIDs(ctx, cached, m.fillLocalUserIDMap, req.TenantID); err != nil {
			return nil, fmt.Errorf("转换用户ID失败: %w", err)
		}
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 5. 构建SQL
	query, args := buildUserStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询
	type userRow struct {
		UserID int64 `db:"slp_user_id"`
		Count  int64 `db:"count"`
	}
	var rows []*userRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询处理人统计失败: %w", err)
	}

	// 8. 提取用户ID列表（转换为字符串）
	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, strconv.FormatInt(row.UserID, 10))
	}

	// 9. 批量查询用户名（复用现有逻辑）
	userNames, err := m.batchGetUserNames(ctx, remoteDB, req.TenantID, userIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询用户名失败: %w", err)
	}

	// 10. 处理结果（添加排名和用户名）
	userMetrics := make([]*core.UserMetric, 0, len(rows))
	for i, row := range rows {
		userIDStr := strconv.FormatInt(row.UserID, 10)
		userName := userNames[userIDStr]
		if userName == "" {
			userName = userIDStr // 如果查不到用户名，显示用户ID
		}

		userMetrics = append(userMetrics, &core.UserMetric{
			UserID:   userIDStr,
			UserName: userName,
			Count:    row.Count,
			Rank:     i + 1, // 排名从1开始
		})
	}

	stats := &core.UserStats{
		UserMetrics: userMetrics,
		Total:       int64(len(userMetrics)),
	}

	// 11. 批量转换用户ID（远程ID → 本地ID）
	if err := convertUserStatsUserIDs(ctx, stats, m.fillLocalUserIDMap, req.TenantID); err != nil {
		return nil, fmt.Errorf("转换用户ID失败: %w", err)
	}

	// 12. 写入缓存
	_ = m.setCachedUserStats(ctx, req, stats)

	return stats, nil
}

// getSingleOrgStats 查询单个事件配置的组织统计（内部方法）
func (m *statsManager) getSingleOrgStats(ctx context.Context, req *core.StatsCriteria) (*core.OrgStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedOrgStats(ctx, req)
	if err == nil && cached != nil {
		return cached, nil
	}

	eventConfigID := req.EventConfigIDs[0]

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 检查是否配置了组织字段（未配置时返回空结果，前端应避免调用）
	if eventConfigWithFields.EventConfig.OrgFieldName == nil {
		// 返回空统计（提示前端该事件不支持组织统计）
		return &core.OrgStats{OrgMetrics: []*core.OrgMetric{}}, nil
	}

	// 4. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 5. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 6. 构建SQL
	query, args := buildOrgStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 7. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 8. 执行查询
	type orgRow struct {
		OrgValue    string          `db:"org_value"`
		Count       int64           `db:"count"`
		AvgDuration sql.NullFloat64 `db:"avg_duration"` // 使用NullFloat64防止NULL值
	}
	var rows []*orgRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询组织统计失败: %w", err)
	}

	// 9. 处理结果
	orgMetrics := make([]*core.OrgMetric, 0, len(rows))
	for _, row := range rows {
		orgMetrics = append(orgMetrics, &core.OrgMetric{
			OrgValue:    row.OrgValue,
			Count:       row.Count,
			AvgDuration: row.AvgDuration.Float64, // 如果为NULL则为0.0
		})
	}

	stats := &core.OrgStats{
		OrgMetrics: orgMetrics,
	}

	// 10. 写入缓存
	_ = m.setCachedOrgStats(ctx, req, stats)

	return stats, nil
}

// getSinglePendingStats 查询单个事件配置的待处理统计（内部方法，实时查询）
func (m *statsManager) getSinglePendingStats(ctx context.Context, req *core.StatsCriteria) (*core.PendingStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	eventConfigID := req.EventConfigIDs[0]

	// 1. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, eventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 2. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 3. 计算组织权限
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 4. 构建SQL（现在返回包含事件信息的结果，支持日期范围筛选）
	query, args := buildPendingStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 5. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 6. 执行查询（返回单个事件的统计）
	var eventStats core.EventPendingStats
	err = remoteDB.QueryRowCtx(ctx, &eventStats, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询待处理事件统计失败: %w", err)
	}

	// 7. 封装到PendingStats结构
	stats := &core.PendingStats{
		EventStats:      []*core.EventPendingStats{&eventStats},
		TotalPending:    eventStats.PendingCount,
		TotalProcessing: eventStats.ProcessingCount,
		Total:           eventStats.Total,
	}

	return stats, nil
}
