package stats

import (
	"database/sql"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// DurationStatsModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射，允许使用框架类型
type DurationStatsModel struct {
	AvgDuration    sql.NullFloat64 `db:"avg_duration"`    // 平均处理时长（秒）
	MinDuration    sql.NullFloat64 `db:"min_duration"`    // 最短处理时长（秒）
	MaxDuration    sql.NullFloat64 `db:"max_duration"`    // 最长处理时长（秒）
	TotalCount     int64           `db:"total_count"`     // 统计范围内的总事件数
	CompletedCount int64           `db:"completed_count"` // 已完成事件数
}

// EventPendingStatsModel 是数据库查询专用结构体（基础设施层）
type EventPendingStatsModel struct {
	EventConfigID   string `db:"event_config_id"`  // 事件配置ID
	EventName       string `db:"event_name"`       // 事件名称
	PendingCount    int64  `db:"pending_count"`    // 未开始事件数
	ProcessingCount int64  `db:"processing_count"` // 处理中事件数
	Total           int64  `db:"total"`            // 该事件的总数
}

// ToDomain 将数据库模型转换为领域模型
func (m *DurationStatsModel) ToDomain() *core.DurationStats {
	return &core.DurationStats{
		AvgDuration:    convertNullFloat64(m.AvgDuration),
		MinDuration:    convertNullFloat64(m.MinDuration),
		MaxDuration:    convertNullFloat64(m.MaxDuration),
		TotalCount:     m.TotalCount,
		CompletedCount: m.CompletedCount,
	}
}

// ToDomain 将数据库模型转换为领域模型
func (m *EventPendingStatsModel) ToDomain() *core.EventPendingStats {
	return &core.EventPendingStats{
		EventConfigID:   m.EventConfigID,
		EventName:       m.EventName,
		PendingCount:    m.PendingCount,
		ProcessingCount: m.ProcessingCount,
		Total:           m.Total,
	}
}
