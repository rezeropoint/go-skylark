// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：统计分析、数据可视化）
package core

import (
	"database/sql"
	"time"
)

const (
	// CacheDurationStatsKeyPrefix 处理时长统计缓存键前缀
	// 完整格式：skylark:stats:duration:{tenant_id}:{event_config_id}:{params_hash}
	CacheDurationStatsKeyPrefix = "skylark:stats:duration:"

	// CacheStatusStatsKeyPrefix 状态统计缓存键前缀
	// 完整格式：skylark:stats:status:{tenant_id}:{event_config_id}:{params_hash}
	CacheStatusStatsKeyPrefix = "skylark:stats:status:"

	// CacheTrendStatsKeyPrefix 趋势统计缓存键前缀
	// 完整格式：skylark:stats:trend:{tenant_id}:{event_config_id}:{params_hash}
	CacheTrendStatsKeyPrefix = "skylark:stats:trend:"

	// CacheNodeStatsKeyPrefix 节点统计缓存键前缀
	// 完整格式：skylark:stats:node:{tenant_id}:{event_config_id}:{params_hash}
	CacheNodeStatsKeyPrefix = "skylark:stats:node:"

	// CacheUserStatsKeyPrefix 处理人统计缓存键前缀
	// 完整格式：skylark:stats:user:{tenant_id}:{event_config_id}:{params_hash}
	CacheUserStatsKeyPrefix = "skylark:stats:user:"

	// CacheOrgStatsKeyPrefix 组织统计缓存键前缀
	// 完整格式：skylark:stats:org:{tenant_id}:{event_config_id}:{params_hash}
	CacheOrgStatsKeyPrefix = "skylark:stats:org:"
)

// StatsRequest 统计请求（通用请求结构）
// 说明：所有统计接口共享此请求结构，通过不同字段组合实现不同统计功能
// 支持单个或多个事件配置ID的统计（多ID时会合并结果）
type StatsRequest struct {
	// 事件配置ID列表（引擎内部加载）
	EventConfigIDs []string // 事件配置ID列表（必填，支持单个或多个ID）

	// 权限上下文
	TenantID   string   // 租户ID（必填）
	UserOrgIDs []string // 用户可见的组织ID列表（含子组织，权限过滤）

	// 通用筛选参数
	DateFrom *time.Time // 开始日期（可选，基于slp_created_at筛选）
	DateTo   *time.Time // 结束日期（可选，基于slp_created_at筛选）
	Status   string     // 按状态筛选（可选，英文状态值如"completed"）

	// 特定统计参数
	GroupBy string // 聚合维度：day/week/month（趋势统计使用）
	TopN    int    // Top N数量（用户统计使用，默认10）
}

// DurationStats 处理时长统计响应
// 说明：统计已完成事件的处理时长（从Journey第一个Assignment创建到最后一个Assignment更新）
type DurationStats struct {
	AvgDuration    sql.NullFloat64 `db:"avg_duration"`    // 平均处理时长（秒）
	MinDuration    sql.NullFloat64 `db:"min_duration"`    // 最短处理时长（秒）
	MaxDuration    sql.NullFloat64 `db:"max_duration"`    // 最长处理时长（秒）
	TotalCount     int64           `db:"total_count"`     // 统计范围内的总事件数（Journey数量）
	CompletedCount int64           `db:"completed_count"` // 已完成事件数
}

// StatusStats 状态统计响应
// 说明：统计各状态的Journey数量和占比
type StatusStats struct {
	StatusCounts []*StatusCount // 各状态数量（按数量降序排列）
	Total        int64          // 总Journey数量
}

// StatusCount 单个状态统计项
type StatusCount struct {
	Status     string  // 状态显示名称（中文翻译后，如"处理中"）
	StatusKey  string  // 状态原始值（英文，如"processing"）
	Count      int64   // 该状态的Journey数量
	Percentage float64 // 占比（0-100）
}

// TrendStats 趋势统计响应
// 说明：按时间维度（日/周/月）统计事件数量和完成率
type TrendStats struct {
	TimePoints []*TrendPoint // 时间点数据（按时间正序排列）
}

// TrendPoint 单个时间点统计数据
type TrendPoint struct {
	Date           string  // 日期（格式：YYYY-MM-DD 或 YYYY-WW 或 YYYY-MM）
	TotalCount     int64   // 该时间段新增的总事件数
	CompletedCount int64   // 该时间段完成的事件数
	CompletionRate float64 // 完成率（0-100）
}

// NodeStats 节点统计响应
// 说明：统计各节点的事件数量和平均处理时长
type NodeStats struct {
	NodeMetrics []*NodeMetric // 节点指标列表（按事件数量降序排列）
}

// NodeMetric 单个节点统计指标
type NodeMetric struct {
	VertexID    int     // 节点ID
	VertexName  string  // 节点名称（优先别名，否则使用name）
	Count       int64   // 该节点的Journey数量（不同Journey经过该节点的次数）
	AvgDuration float64 // 该节点平均停留时长（秒）
}

// UserStats 处理人统计响应
// 说明：统计处理人的处理数量和排名（Top N）
type UserStats struct {
	UserMetrics []*UserMetric // 用户指标列表（按处理数量降序排列）
	Total       int64         // 统计范围内的总处理人数
}

// UserMetric 单个处理人统计指标
type UserMetric struct {
	UserID   string // 用户ID（对应slp_user_id）
	UserName string // 用户姓名（通过远程users表转换）
	Count    int64  // 处理的Journey数量
	Rank     int    // 排名（1开始）
}

// OrgStats 组织统计响应
// 说明：统计各组织的事件数量和平均处理时长
type OrgStats struct {
	OrgMetrics []*OrgMetric // 组织指标列表（按事件数量降序排列）
}

// OrgMetric 单个组织统计指标
type OrgMetric struct {
	OrgValue    string  // 组织字段值（远程表中的实际值）
	Count       int64   // 该组织的Journey数量
	AvgDuration float64 // 该组织的平均处理时长（秒）
}

// PendingStats 待处理事件统计响应（实时查询，不缓存）
// 说明：统计未开始和处理中的事件数量，按事件配置分组，用于大屏实时提醒
// 未开始定义：Journey只有1个不同的vertex_id（即只有发起节点的记录）
type PendingStats struct {
	EventStats      []*EventPendingStats // 按事件配置分组的统计（按事件名称排序）
	TotalPending    int64                // 所有事件的未开始总数
	TotalProcessing int64                // 所有事件的处理中总数
	Total           int64                // 总事件数（未完成的事件）
}

// EventPendingStats 单个事件的待处理统计
type EventPendingStats struct {
	EventConfigID   string `db:"event_config_id" json:"eventConfigId"`    // 事件配置ID
	EventName       string `db:"event_name" json:"eventName"`             // 事件名称
	PendingCount    int64  `db:"pending_count" json:"pendingCount"`       // 未开始事件数（只有发起节点）
	ProcessingCount int64  `db:"processing_count" json:"processingCount"` // 处理中事件数（有多个节点）
	Total           int64  `db:"total" json:"total"`                      // 该事件的总数（未完成的事件）
}
