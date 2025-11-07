package stats

import (
	"sort"

	"github.com/rezeropoint/go-skylark/core"
)

// ========== 1. 处理时长统计合并 ==========

// mergeDurationStats 合并多个处理时长统计结果
// 说明：
//   - AvgDuration: 加权平均 = Σ(avg * count) / Σ(count)
//   - MinDuration: 取所有结果的最小值
//   - MaxDuration: 取所有结果的最大值
//   - TotalCount: 求和
//   - CompletedCount: 求和
func mergeDurationStats(results []*core.DurationStats) *core.DurationStats {
	if len(results) == 0 {
		return &core.DurationStats{}
	}
	if len(results) == 1 {
		return results[0]
	}

	merged := &core.DurationStats{}
	var minDuration, maxDuration float64
	var hasValidMin, hasValidMax bool

	var weightedSum float64 = 0
	var totalCount int64 = 0

	for _, stats := range results {
		// 加权求和（用于计算平均值）
		if stats.AvgDuration != nil {
			weightedSum += *stats.AvgDuration * float64(stats.TotalCount)
		}
		totalCount += stats.TotalCount

		// 更新最小值
		if stats.MinDuration != nil {
			if !hasValidMin || *stats.MinDuration < minDuration {
				minDuration = *stats.MinDuration
				hasValidMin = true
			}
		}

		// 更新最大值
		if stats.MaxDuration != nil {
			if !hasValidMax || *stats.MaxDuration > maxDuration {
				maxDuration = *stats.MaxDuration
				hasValidMax = true
			}
		}

		// 累加完成数量
		merged.CompletedCount += stats.CompletedCount
	}

	merged.TotalCount = totalCount

	// 计算加权平均
	if totalCount > 0 && weightedSum > 0 {
		avgVal := weightedSum / float64(totalCount)
		merged.AvgDuration = &avgVal
	}

	// 设置最小值和最大值
	if hasValidMin {
		merged.MinDuration = &minDuration
	}
	if hasValidMax {
		merged.MaxDuration = &maxDuration
	}

	return merged
}

// ========== 2. 状态统计合并 ==========

// mergeStatusStats 合并多个状态统计结果
// 说明：
//   - 按StatusKey分组，Count求和
//   - 重新计算Total和Percentage
//   - 按Count降序排序
func mergeStatusStats(results []*core.StatusStats) *core.StatusStats {
	if len(results) == 0 {
		return &core.StatusStats{StatusCounts: []*core.StatusCount{}}
	}
	if len(results) == 1 {
		return results[0]
	}

	// 使用map按StatusKey分组聚合
	countMap := make(map[string]*core.StatusCount)

	for _, stats := range results {
		for _, sc := range stats.StatusCounts {
			if existing, ok := countMap[sc.StatusKey]; ok {
				// 已存在，累加Count
				existing.Count += sc.Count
			} else {
				// 新状态，创建副本
				countMap[sc.StatusKey] = &core.StatusCount{
					Status:    sc.Status,    // 保留中文翻译
					StatusKey: sc.StatusKey, // 英文key
					Count:     sc.Count,
				}
			}
		}
	}

	// 转换为切片
	statusCounts := make([]*core.StatusCount, 0, len(countMap))
	var total int64 = 0
	for _, sc := range countMap {
		statusCounts = append(statusCounts, sc)
		total += sc.Count
	}

	// 按Count降序排序
	sort.Slice(statusCounts, func(i, j int) bool {
		return statusCounts[i].Count > statusCounts[j].Count
	})

	// 重新计算Percentage
	for _, sc := range statusCounts {
		if total > 0 {
			sc.Percentage = float64(sc.Count) / float64(total) * 100
		}
	}

	return &core.StatusStats{
		StatusCounts: statusCounts,
		Total:        total,
	}
}

// ========== 3. 趋势统计合并 ==========

// mergeTrendStats 合并多个趋势统计结果
// 说明：
//   - 按Date分组，TotalCount和CompletedCount求和
//   - 重新计算CompletionRate
//   - 按Date正序排序
func mergeTrendStats(results []*core.TrendStats) *core.TrendStats {
	if len(results) == 0 {
		return &core.TrendStats{TimePoints: []*core.TrendPoint{}}
	}
	if len(results) == 1 {
		return results[0]
	}

	// 使用map按Date分组聚合
	pointMap := make(map[string]*core.TrendPoint)

	for _, stats := range results {
		for _, tp := range stats.TimePoints {
			if existing, ok := pointMap[tp.Date]; ok {
				// 已存在，累加Count
				existing.TotalCount += tp.TotalCount
				existing.CompletedCount += tp.CompletedCount
			} else {
				// 新日期，创建副本
				pointMap[tp.Date] = &core.TrendPoint{
					Date:           tp.Date,
					TotalCount:     tp.TotalCount,
					CompletedCount: tp.CompletedCount,
				}
			}
		}
	}

	// 转换为切片
	timePoints := make([]*core.TrendPoint, 0, len(pointMap))
	for _, tp := range pointMap {
		// 重新计算CompletionRate
		if tp.TotalCount > 0 {
			tp.CompletionRate = float64(tp.CompletedCount) / float64(tp.TotalCount) * 100
		}
		timePoints = append(timePoints, tp)
	}

	// 按Date正序排序
	sort.Slice(timePoints, func(i, j int) bool {
		return timePoints[i].Date < timePoints[j].Date
	})

	return &core.TrendStats{
		TimePoints: timePoints,
	}
}

// ========== 4. 节点统计合并 ==========

// mergeNodeStats 合并多个节点统计结果
// 说明：
//   - 按VertexID分组，Count求和
//   - AvgDuration加权平均 = Σ(avg * count) / Σ(count)
//   - 按Count降序排序
func mergeNodeStats(results []*core.NodeStats) *core.NodeStats {
	if len(results) == 0 {
		return &core.NodeStats{NodeMetrics: []*core.NodeMetric{}}
	}
	if len(results) == 1 {
		return results[0]
	}

	// 使用map按VertexID分组聚合
	// 注意：不同事件配置可能有相同的VertexID但VertexName不同，这里优先使用第一个非空名称
	type nodeAgg struct {
		vertexID    int
		vertexName  string
		count       int64
		weightedSum float64 // 用于计算加权平均
	}

	nodeMap := make(map[int]*nodeAgg)

	for _, stats := range results {
		for _, nm := range stats.NodeMetrics {
			if existing, ok := nodeMap[nm.VertexID]; ok {
				// 已存在，累加Count和加权和
				existing.count += nm.Count
				existing.weightedSum += nm.AvgDuration * float64(nm.Count)
				// 如果当前名称为空但新名称非空，更新名称
				if existing.vertexName == "" && nm.VertexName != "" {
					existing.vertexName = nm.VertexName
				}
			} else {
				// 新节点
				nodeMap[nm.VertexID] = &nodeAgg{
					vertexID:    nm.VertexID,
					vertexName:  nm.VertexName,
					count:       nm.Count,
					weightedSum: nm.AvgDuration * float64(nm.Count),
				}
			}
		}
	}

	// 转换为切片
	nodeMetrics := make([]*core.NodeMetric, 0, len(nodeMap))
	for _, agg := range nodeMap {
		avgDuration := float64(0)
		if agg.count > 0 {
			avgDuration = agg.weightedSum / float64(agg.count)
		}

		nodeMetrics = append(nodeMetrics, &core.NodeMetric{
			VertexID:    agg.vertexID,
			VertexName:  agg.vertexName,
			Count:       agg.count,
			AvgDuration: avgDuration,
		})
	}

	// 按Count降序排序
	sort.Slice(nodeMetrics, func(i, j int) bool {
		return nodeMetrics[i].Count > nodeMetrics[j].Count
	})

	return &core.NodeStats{
		NodeMetrics: nodeMetrics,
	}
}

// ========== 5. 处理人统计合并 ==========

// mergeUserStats 合并多个处理人统计结果
// 说明：
//   - 按UserID分组，Count求和
//   - 按Count降序排序
//   - 重新计算Rank
//   - 取Top N
func mergeUserStats(results []*core.UserStats, topN int) *core.UserStats {
	if len(results) == 0 {
		return &core.UserStats{UserMetrics: []*core.UserMetric{}}
	}
	if len(results) == 1 {
		// 单个结果也要检查TopN
		if topN > 0 && len(results[0].UserMetrics) > topN {
			return &core.UserStats{
				UserMetrics: results[0].UserMetrics[:topN],
				Total:       int64(topN),
			}
		}
		return results[0]
	}

	// 使用map按UserID分组聚合
	type userAgg struct {
		userID   string
		userName string
		count    int64
	}

	userMap := make(map[string]*userAgg)

	for _, stats := range results {
		for _, um := range stats.UserMetrics {
			if existing, ok := userMap[um.UserID]; ok {
				// 已存在，累加Count
				existing.count += um.Count
				// 如果当前名称为空但新名称非空，更新名称
				if existing.userName == "" && um.UserName != "" {
					existing.userName = um.UserName
				}
			} else {
				// 新用户
				userMap[um.UserID] = &userAgg{
					userID:   um.UserID,
					userName: um.UserName,
					count:    um.Count,
				}
			}
		}
	}

	// 转换为切片
	userMetrics := make([]*core.UserMetric, 0, len(userMap))
	for _, agg := range userMap {
		userMetrics = append(userMetrics, &core.UserMetric{
			UserID:   agg.userID,
			UserName: agg.userName,
			Count:    agg.count,
		})
	}

	// 按Count降序排序
	sort.Slice(userMetrics, func(i, j int) bool {
		return userMetrics[i].Count > userMetrics[j].Count
	})

	// 取Top N
	if topN > 0 && len(userMetrics) > topN {
		userMetrics = userMetrics[:topN]
	}

	// 重新计算Rank
	for i := range userMetrics {
		userMetrics[i].Rank = i + 1
	}

	return &core.UserStats{
		UserMetrics: userMetrics,
		Total:       int64(len(userMetrics)),
	}
}

// ========== 6. 组织统计合并 ==========

// mergeOrgStats 合并多个组织统计结果
// 说明：
//   - 按OrgValue分组，Count求和
//   - AvgDuration加权平均 = Σ(avg * count) / Σ(count)
//   - 按Count降序排序
func mergeOrgStats(results []*core.OrgStats) *core.OrgStats {
	if len(results) == 0 {
		return &core.OrgStats{OrgMetrics: []*core.OrgMetric{}}
	}
	if len(results) == 1 {
		return results[0]
	}

	// 使用map按OrgValue分组聚合
	type orgAgg struct {
		orgValue    string
		count       int64
		weightedSum float64 // 用于计算加权平均
	}

	orgMap := make(map[string]*orgAgg)

	for _, stats := range results {
		for _, om := range stats.OrgMetrics {
			if existing, ok := orgMap[om.OrgValue]; ok {
				// 已存在，累加Count和加权和
				existing.count += om.Count
				existing.weightedSum += om.AvgDuration * float64(om.Count)
			} else {
				// 新组织
				orgMap[om.OrgValue] = &orgAgg{
					orgValue:    om.OrgValue,
					count:       om.Count,
					weightedSum: om.AvgDuration * float64(om.Count),
				}
			}
		}
	}

	// 转换为切片
	orgMetrics := make([]*core.OrgMetric, 0, len(orgMap))
	for _, agg := range orgMap {
		avgDuration := float64(0)
		if agg.count > 0 {
			avgDuration = agg.weightedSum / float64(agg.count)
		}

		orgMetrics = append(orgMetrics, &core.OrgMetric{
			OrgValue:    agg.orgValue,
			Count:       agg.count,
			AvgDuration: avgDuration,
		})
	}

	// 按Count降序排序
	sort.Slice(orgMetrics, func(i, j int) bool {
		return orgMetrics[i].Count > orgMetrics[j].Count
	})

	return &core.OrgStats{
		OrgMetrics: orgMetrics,
	}
}

// ========== 7. 待处理事件统计合并 ==========

// mergePendingStats 合并多个待处理事件统计结果
// 说明：
//   - 保留所有事件的EventStats列表
//   - 计算TotalPending、TotalProcessing、Total的总和
//   - 按事件名称排序
func mergePendingStats(results []*core.PendingStats) *core.PendingStats {
	if len(results) == 0 {
		return &core.PendingStats{EventStats: []*core.EventPendingStats{}}
	}
	if len(results) == 1 {
		return results[0]
	}

	merged := &core.PendingStats{
		EventStats: make([]*core.EventPendingStats, 0),
	}

	// 合并所有事件的统计数据
	for _, stats := range results {
		// 累加总数
		merged.TotalPending += stats.TotalPending
		merged.TotalProcessing += stats.TotalProcessing
		merged.Total += stats.Total

		// 收集每个事件的统计
		merged.EventStats = append(merged.EventStats, stats.EventStats...)
	}

	// 按事件名称排序
	sort.Slice(merged.EventStats, func(i, j int) bool {
		return merged.EventStats[i].EventName < merged.EventStats[j].EventName
	})

	return merged
}
