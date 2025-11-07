package stats

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// statsManager 统计分析管理器实现
type statsManager struct {
	config          Config                            // 配置参数
	dbConn          sqlx.SqlConn                      // 本地数据库连接（查询事件配置、字段配置、组织映射）
	getRemoteDB     core.GetRemoteDBFunc              // 获取远程数据库连接的函数（由 Platform Manager 提供）
	getEventConfig  core.GetEventConfigWithFieldsFunc // 获取事件配置（含字段）的函数（由 Event Manager 提供）
	listOrgMappings core.ListOrgMappingsFunc          // 获取组织映射列表的函数（由 Mapping Manager 提供）
	cache           core.CacheInterface               // 缓存接口（统一缓存管理）
}

// newStatsManager 创建统计分析管理器
func newStatsManager(
	config Config,
	db sqlx.SqlConn,
	getRemoteDB core.GetRemoteDBFunc,
	getEventConfig core.GetEventConfigWithFieldsFunc,
	listOrgMappings core.ListOrgMappingsFunc,
	cache core.CacheInterface,
) (*statsManager, error) {
	// 验证必填参数
	if getRemoteDB == nil {
		return nil, fmt.Errorf("getRemoteDB 函数不能为空")
	}
	if getEventConfig == nil {
		return nil, fmt.Errorf("getEventConfig 函数不能为空")
	}
	if listOrgMappings == nil {
		return nil, fmt.Errorf("listOrgMappings 函数不能为空")
	}

	// 缓存接口必须提供
	if cache == nil {
		return nil, fmt.Errorf("缓存接口不能为空")
	}

	if config.UserNameCacheTTL <= 0 {
		config.UserNameCacheTTL = 24 * time.Hour
	}
	if config.StatsCacheTTL <= 0 {
		config.StatsCacheTTL = 5 * time.Minute
	}

	manager := &statsManager{
		config:          config,
		dbConn:          db,
		getRemoteDB:     getRemoteDB,
		getEventConfig:  getEventConfig,
		listOrgMappings: listOrgMappings,
		cache:           cache,
	}

	return manager, nil
}

// ========== 1. 处理时长统计 ==========

// GetDurationStats 获取事件处理时长统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.DurationStats{
				AvgDuration:    nil,
				MinDuration:    nil,
				MaxDuration:    nil,
				TotalCount:     0,
				CompletedCount: 0,
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_duration_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleDurationStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedDurationStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_duration_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件处理时长统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.DurationStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleDurationStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的处理时长统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果
	merged := mergeDurationStats(results)

	// 6. 缓存合并结果
	_ = m.setCachedDurationStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_duration_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("avg_duration", merged.AvgDuration),
		logx.Field("total_count", merged.TotalCount),
	).Info("查询多事件处理时长统计成功（已合并）")

	return merged, nil
}

// getSingleDurationStats 查询单个事件配置的处理时长统计（内部方法）
func (m *statsManager) getSingleDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error) {
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

	// 7. 执行查询
	var stats core.DurationStats
	err = remoteDB.QueryRowCtx(ctx, &stats, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询处理时长统计失败: %w", err)
	}

	// 8. 写入缓存
	_ = m.setCachedDurationStats(ctx, req, &stats)

	return &stats, nil
}

// ========== 2. 状态统计 ==========

// GetStatusStats 获取事件状态统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.StatusStats{
				StatusCounts: []*core.StatusCount{},
				Total:        0,
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_status_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleStatusStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedStatusStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_status_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件状态统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.StatusStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleStatusStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的状态统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果
	merged := mergeStatusStats(results)

	// 6. 缓存合并结果
	_ = m.setCachedStatusStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_status_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("total", merged.Total),
		logx.Field("status_count", len(merged.StatusCounts)),
	).Info("查询多事件状态统计成功（已合并）")

	return merged, nil
}

// getSingleStatusStats 查询单个事件配置的状态统计（内部方法）
func (m *statsManager) getSingleStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error) {
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

// ========== 3. 趋势统计 ==========

// GetTrendStats 获取事件趋势统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.TrendStats{
				TimePoints: []*core.TrendPoint{},
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_trend_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 默认GroupBy为day
	if req.GroupBy == "" {
		req.GroupBy = "day"
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleTrendStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedTrendStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_trend_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件趋势统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.TrendStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleTrendStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的趋势统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果
	merged := mergeTrendStats(results)

	// 6. 缓存合并结果
	_ = m.setCachedTrendStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_trend_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("group_by", req.GroupBy),
		logx.Field("point_count", len(merged.TimePoints)),
	).Info("查询多事件趋势统计成功（已合并）")

	return merged, nil
}

// getSingleTrendStats 查询单个事件配置的趋势统计（内部方法）
func (m *statsManager) getSingleTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error) {
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

// ========== 4. 节点统计 ==========

// GetNodeStats 获取节点统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.NodeStats{
				NodeMetrics: []*core.NodeMetric{},
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_node_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleNodeStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedNodeStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_node_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件节点统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.NodeStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleNodeStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的节点统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果
	merged := mergeNodeStats(results)

	// 6. 缓存合并结果
	_ = m.setCachedNodeStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_node_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("node_count", len(merged.NodeMetrics)),
	).Info("查询多事件节点统计成功（已合并）")

	return merged, nil
}

// getSingleNodeStats 查询单个事件配置的节点统计（内部方法）
func (m *statsManager) getSingleNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error) {
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

// ========== 5. 处理人统计 ==========

// GetUserStats 获取处理人统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.UserStats{
				UserMetrics: []*core.UserMetric{},
				Total:       0,
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_user_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 默认Top 10
	if req.TopN <= 0 {
		req.TopN = 10
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleUserStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedUserStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_user_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件处理人统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.UserStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleUserStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的处理人统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果（传入TopN参数）
	merged := mergeUserStats(results, req.TopN)

	// 6. 缓存合并结果
	_ = m.setCachedUserStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_user_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("user_count", len(merged.UserMetrics)),
	).Info("查询多事件处理人统计成功（已合并）")

	return merged, nil
}

// getSingleUserStats 查询单个事件配置的处理人统计（内部方法）
func (m *statsManager) getSingleUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error) {
	// 注意：此时req.EventConfigIDs应该只有1个元素

	// 1. 尝试从缓存获取
	cached, err := m.getCachedUserStats(ctx, req)
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
	query, args := buildUserStatsSQL(&eventConfigWithFields.EventConfig, allowedOrgValues, req)

	// 6. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 7. 执行查询
	type userRow struct {
		UserID string `db:"slp_user_id"`
		Count  int64  `db:"count"`
	}
	var rows []*userRow
	err = remoteDB.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询处理人统计失败: %w", err)
	}

	// 8. 提取用户ID列表
	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	// 9. 批量查询用户名（复用现有逻辑）
	userNames, err := m.batchGetUserNames(ctx, remoteDB, req.TenantID, userIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询用户名失败: %w", err)
	}

	// 10. 处理结果（添加排名和用户名）
	userMetrics := make([]*core.UserMetric, 0, len(rows))
	for i, row := range rows {
		userName := userNames[row.UserID]
		if userName == "" {
			userName = row.UserID // 如果查不到用户名，显示用户ID
		}

		userMetrics = append(userMetrics, &core.UserMetric{
			UserID:   row.UserID,
			UserName: userName,
			Count:    row.Count,
			Rank:     i + 1, // 排名从1开始
		})
	}

	stats := &core.UserStats{
		UserMetrics: userMetrics,
		Total:       int64(len(userMetrics)),
	}

	// 11. 写入缓存
	_ = m.setCachedUserStats(ctx, req, stats)

	return stats, nil
}

// ========== 6. 组织统计 ==========

// GetOrgStats 获取组织统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.OrgStats{
				OrgMetrics: []*core.OrgMetric{},
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_org_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSingleOrgStats(ctx, req)
	}

	// 3. 多ID：尝试从缓存获取合并结果
	cached, err := m.getCachedOrgStats(ctx, req)
	if err == nil && cached != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_org_stats"),
			logx.Field("event_config_ids", req.EventConfigIDs),
			logx.Field("source", "cache"),
		).Info("从缓存获取多事件组织统计成功")
		return cached, nil
	}

	// 4. 缓存未命中，循环查询每个ID并合并结果
	results := make([]*core.OrgStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（利用单ID缓存）
		singleStats, err := m.getSingleOrgStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的组织统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 5. 合并结果
	merged := mergeOrgStats(results)

	// 6. 缓存合并结果
	_ = m.setCachedOrgStats(ctx, req, merged)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_org_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_merged"),
		logx.Field("org_count", len(merged.OrgMetrics)),
	).Info("查询多事件组织统计成功（已合并）")

	return merged, nil
}

// getSingleOrgStats 查询单个事件配置的组织统计（内部方法）
func (m *statsManager) getSingleOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error) {
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

	// 3. 检查是否配置了组织字段
	if eventConfigWithFields.EventConfig.OrgFieldName == nil {
		return nil, fmt.Errorf("该事件未配置组织字段，无法进行组织统计")
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
	if query == "" {
		return nil, fmt.Errorf("构建组织统计SQL失败")
	}

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

// ========== 7. 待处理事件统计（实时查询，不缓存） ==========

// GetPendingStats 获取待处理事件统计（实时查询，不使用缓存，支持单个或多个事件配置ID，支持空ID查询所有事件）
// 说明：用于大屏实时提醒，统计未开始的事件数量
// 未开始定义：Journey只有1个不同的vertex_id（即只有发起节点的记录）
func (m *statsManager) GetPendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error) {
	// 1. 如果 EventConfigIDs 为空，查询租户下所有事件配置
	if len(req.EventConfigIDs) == 0 {
		allIDs, err := m.getAllEventConfigIDs(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("获取租户事件配置列表失败: %w", err)
		}

		// 如果租户下没有任何事件配置，返回空统计结果
		if len(allIDs) == 0 {
			return &core.PendingStats{
				EventStats:      []*core.EventPendingStats{},
				TotalPending:    0,
				TotalProcessing: 0,
				Total:           0,
			}, nil
		}

		// 将查询到的所有ID设置到请求中
		req.EventConfigIDs = allIDs
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "stats_manager"),
			logx.Field("operation", "get_pending_stats"),
			logx.Field("tenant_id", req.TenantID),
			logx.Field("event_count", len(allIDs)),
		).Info("EventConfigIDs 为空，已自动查询租户下所有事件配置")
	}

	// ⚠️ 注意：此接口不使用缓存，每次都实时查询

	// 2. 单ID：使用原有逻辑（向后兼容）
	if len(req.EventConfigIDs) == 1 {
		return m.getSinglePendingStats(ctx, req)
	}

	// 3. 多ID：循环查询每个ID并合并结果（不使用缓存）
	results := make([]*core.PendingStats, 0, len(req.EventConfigIDs))
	for _, eventConfigID := range req.EventConfigIDs {
		// 为每个ID创建单独的请求
		singleReq := *req
		singleReq.EventConfigIDs = []string{eventConfigID}

		// 查询单个ID的统计（实时查询）
		singleStats, err := m.getSinglePendingStats(ctx, &singleReq)
		if err != nil {
			return nil, fmt.Errorf("查询事件配置[%s]的待处理统计失败: %w", eventConfigID, err)
		}

		results = append(results, singleStats)
	}

	// 4. 合并结果
	merged := mergePendingStats(results)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_pending_stats"),
		logx.Field("event_config_ids", req.EventConfigIDs),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("source", "database_realtime_merged"),
		logx.Field("event_count", len(merged.EventStats)),
		logx.Field("total_pending", merged.TotalPending),
		logx.Field("total_processing", merged.TotalProcessing),
		logx.Field("total", merged.Total),
	).Info("查询多事件待处理统计成功（实时合并）")

	return merged, nil
}

// getSinglePendingStats 查询单个事件配置的待处理统计（内部方法，实时查询）
func (m *statsManager) getSinglePendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error) {
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
