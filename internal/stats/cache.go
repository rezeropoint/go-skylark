package stats

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// batchGetUserNames 批量查询用户名（优先从 Redis 缓存获取）
func (m *statsManager) batchGetUserNames(ctx context.Context, remoteDB sqlx.SqlConn, tenantID string, userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return make(map[string]string), nil
	}

	userNames := make(map[string]string, len(userIDs))
	uncachedIDs := []string{}

	// 1. 尝试从 Redis 获取缓存（go-zero Redis 逐个获取）
	if m.rdb != nil {
		for _, userID := range userIDs {
			key := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, userID)
			val, err := m.rdb.Get(key)
			if err == nil && val != "" {
				userNames[userID] = val
			} else {
				uncachedIDs = append(uncachedIDs, userID)
			}
		}
	} else {
		uncachedIDs = userIDs
	}

	// 2. 查询未命中的用户名（从远程 users 表）
	if len(uncachedIDs) > 0 {
		type userRow struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}

		query := "SELECT id, name FROM users WHERE id = ANY($1)"
		var users []*userRow
		err := remoteDB.QueryRowsCtx(ctx, &users, query, pq.Array(uncachedIDs))
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("批量查询用户名失败: %w", err)
		}

		for _, user := range users {
			userNames[user.ID] = user.Name

			// 3. 写入 Redis 缓存
			if m.rdb != nil {
				cacheKey := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, user.ID)
				_ = m.rdb.Setex(cacheKey, user.Name, int(m.config.UserNameCacheTTL.Seconds()))
			}
		}
	}

	return userNames, nil
}

// buildStatsCacheKey 构建统计缓存键
// 说明：根据请求参数生成唯一的缓存键，支持单个或多个事件配置ID
// 参数：
//   - prefix: 缓存键前缀（如：skylark:stats:duration:）
//   - tenantID: 租户ID
//   - eventConfigIDs: 事件配置ID列表
//   - req: 统计请求参数（用于生成hash）
//
// 返回：完整的缓存键
//   - 单ID格式：prefix + tenantID + ":" + eventConfigID + ":" + params_hash（向后兼容）
//   - 多ID格式：prefix + tenantID + ":multi:" + sorted_ids_hash + ":" + params_hash
func (m *statsManager) buildStatsCacheKey(prefix, tenantID string, eventConfigIDs []string, req *core.StatsRequest) string {
	// 构建参数字符串（用于hash）
	paramsStr := fmt.Sprintf("%v|%v|%s|%s|%d",
		req.DateFrom,
		req.DateTo,
		req.Status,
		req.GroupBy,
		req.TopN,
	)

	// 计算参数hash
	paramsHash := md5.Sum([]byte(paramsStr))
	paramsHashStr := fmt.Sprintf("%x", paramsHash)

	// 单ID：使用原格式（向后兼容）
	if len(eventConfigIDs) == 1 {
		return fmt.Sprintf("%s%s:%s:%s", prefix, tenantID, eventConfigIDs[0], paramsHashStr)
	}

	// 多ID：对ID排序后计算hash（保证相同ID组合的缓存key一致）
	sortedIDs := make([]string, len(eventConfigIDs))
	copy(sortedIDs, eventConfigIDs)
	sort.Strings(sortedIDs)

	// 拼接所有ID并计算hash
	idsStr := ""
	for _, id := range sortedIDs {
		idsStr += id + ","
	}
	idsHash := md5.Sum([]byte(idsStr))
	idsHashStr := fmt.Sprintf("%x", idsHash)

	// 返回多ID格式
	return fmt.Sprintf("%s%s:multi:%s:%s", prefix, tenantID, idsHashStr, paramsHashStr)
}

// ========== 处理时长统计缓存 ==========

// getCachedDurationStats 从缓存获取处理时长统计
func (m *statsManager) getCachedDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error) {
	key := m.buildStatsCacheKey(core.CacheDurationStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.DurationStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化处理时长统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedDurationStats 缓存处理时长统计
func (m *statsManager) setCachedDurationStats(ctx context.Context, req *core.StatsRequest, stats *core.DurationStats) error {
	key := m.buildStatsCacheKey(core.CacheDurationStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化处理时长统计失败: %w", err)
	}

	return m.rdb.SetexCtx(ctx, key, string(data), int(m.config.StatsCacheTTL.Seconds()))
}

// ========== 状态统计缓存 ==========

// getCachedStatusStats 从缓存获取状态统计
func (m *statsManager) getCachedStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error) {
	key := m.buildStatsCacheKey(core.CacheStatusStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.StatusStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化状态统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedStatusStats 缓存状态统计
func (m *statsManager) setCachedStatusStats(ctx context.Context, req *core.StatsRequest, stats *core.StatusStats) error {
	key := m.buildStatsCacheKey(core.CacheStatusStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化状态统计失败: %w", err)
	}

	return m.rdb.SetexCtx(ctx, key, string(data), int(m.config.StatsCacheTTL.Seconds()))
}

// ========== 趋势统计缓存 ==========

// getCachedTrendStats 从缓存获取趋势统计
func (m *statsManager) getCachedTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error) {
	key := m.buildStatsCacheKey(core.CacheTrendStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.TrendStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化趋势统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedTrendStats 缓存趋势统计
func (m *statsManager) setCachedTrendStats(ctx context.Context, req *core.StatsRequest, stats *core.TrendStats) error {
	key := m.buildStatsCacheKey(core.CacheTrendStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化趋势统计失败: %w", err)
	}

	// 根据GroupBy设置不同的TTL
	ttl := m.config.StatsCacheTTL
	if req.GroupBy == "month" {
		// 按月聚合的数据可以缓存更久
		ttl = 24 * time.Hour
	} else if req.GroupBy == "week" {
		// 按周聚合的数据缓存1小时
		ttl = time.Hour
	}

	return m.rdb.SetexCtx(ctx, key, string(data), int(ttl.Seconds()))
}

// ========== 节点统计缓存 ==========

// getCachedNodeStats 从缓存获取节点统计
func (m *statsManager) getCachedNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error) {
	key := m.buildStatsCacheKey(core.CacheNodeStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.NodeStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化节点统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedNodeStats 缓存节点统计
func (m *statsManager) setCachedNodeStats(ctx context.Context, req *core.StatsRequest, stats *core.NodeStats) error {
	key := m.buildStatsCacheKey(core.CacheNodeStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化节点统计失败: %w", err)
	}

	// 节点统计相对稳定，可以缓存10分钟
	ttl := 10 * time.Minute
	return m.rdb.SetexCtx(ctx, key, string(data), int(ttl.Seconds()))
}

// ========== 处理人统计缓存 ==========

// getCachedUserStats 从缓存获取处理人统计
func (m *statsManager) getCachedUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error) {
	key := m.buildStatsCacheKey(core.CacheUserStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.UserStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化处理人统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedUserStats 缓存处理人统计
func (m *statsManager) setCachedUserStats(ctx context.Context, req *core.StatsRequest, stats *core.UserStats) error {
	key := m.buildStatsCacheKey(core.CacheUserStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化处理人统计失败: %w", err)
	}

	// 处理人统计可以缓存10分钟
	ttl := 10 * time.Minute
	return m.rdb.SetexCtx(ctx, key, string(data), int(ttl.Seconds()))
}

// ========== 组织统计缓存 ==========

// getCachedOrgStats 从缓存获取组织统计
func (m *statsManager) getCachedOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error) {
	key := m.buildStatsCacheKey(core.CacheOrgStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var stats core.OrgStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("反序列化组织统计缓存失败: %w", err)
	}

	return &stats, nil
}

// setCachedOrgStats 缓存组织统计
func (m *statsManager) setCachedOrgStats(ctx context.Context, req *core.StatsRequest, stats *core.OrgStats) error {
	key := m.buildStatsCacheKey(core.CacheOrgStatsKeyPrefix, req.TenantID, req.EventConfigIDs, req)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化组织统计失败: %w", err)
	}

	// 组织统计可以缓存10分钟
	ttl := 10 * time.Minute
	return m.rdb.SetexCtx(ctx, key, string(data), int(ttl.Seconds()))
}
