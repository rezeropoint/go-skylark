package stats

import (
	"context"
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

// 处理时长统计

// GetDurationStats 获取事件处理时长统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetDurationStats(ctx context.Context, req *core.StatsCriteria) (*core.DurationStats, error) {
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

// ========== 2. 状态统计 ==========

// GetStatusStats 获取事件状态统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetStatusStats(ctx context.Context, req *core.StatsCriteria) (*core.StatusStats, error) {
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

// ========== 3. 趋势统计 ==========

// GetTrendStats 获取事件趋势统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetTrendStats(ctx context.Context, req *core.StatsCriteria) (*core.TrendStats, error) {
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

// ========== 4. 节点统计 ==========

// GetNodeStats 获取节点统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetNodeStats(ctx context.Context, req *core.StatsCriteria) (*core.NodeStats, error) {
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

// ========== 5. 处理人统计 ==========

// GetUserStats 获取处理人统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetUserStats(ctx context.Context, req *core.StatsCriteria) (*core.UserStats, error) {
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

// ========== 6. 组织统计 ==========

// GetOrgStats 获取组织统计（支持单个或多个事件配置ID，支持空ID查询所有事件）
func (m *statsManager) GetOrgStats(ctx context.Context, req *core.StatsCriteria) (*core.OrgStats, error) {
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

// ========== 7. 待处理事件统计（实时查询，不缓存） ==========

// GetPendingStats 获取待处理事件统计（实时查询，不使用缓存，支持单个或多个事件配置ID，支持空ID查询所有事件）
// 说明：用于大屏实时提醒，统计未开始的事件数量
// 未开始定义：Journey只有1个不同的vertex_id（即只有发起节点的记录）
func (m *statsManager) GetPendingStats(ctx context.Context, req *core.StatsCriteria) (*core.PendingStats, error) {
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
