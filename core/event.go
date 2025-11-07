// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：事件配置管理、字段配置）
package core

import (
	"fmt"
	"time"
)

// EventConfig 事件配置领域模型（纯领域模型）
type EventConfig struct {
	ID           string    // 配置UUID
	Name         string    // 事件显示名称（自定义名称）
	FlowID       int       // 远程流程ID（关联flows.id）
	FlowTitle    string    // 远程流程名称（冗余，来自flows.title）
	OrgFieldName *string   // 组织字段名（用于权限过滤，可选）
	Description  *string   // 描述
	Enabled      bool      // 是否启用
	TenantID     string    // 租户ID
	CreatedBy    *string   // 创建者用户ID
	UpdatedBy    *string   // 最后修改者用户ID
	CreatedAt    time.Time // 创建时间
	UpdatedAt    time.Time // 更新时间
}

// GetRemoteTableName 获取远程表名
// 远程表名由FlowID动态生成：assignments_{flow_id}
func (e *EventConfig) GetRemoteTableName() string {
	return fmt.Sprintf("assignments_%d", e.FlowID)
}

// GetRemotePrimaryKey 获取远程表主键字段名
// 固定为 slp_assignment_id（Assignment是真正的主键）
func (e *EventConfig) GetRemotePrimaryKey() string {
	return "slp_assignment_id"
}

// GetJourneyIDField 获取Journey ID字段名
// Journey ID用于聚合查询（多个Assignment共享同一个Journey）
func (e *EventConfig) GetJourneyIDField() string {
	return "slp_journey_id"
}

// ========== 聚合数据结构（用于事件+字段统一管理） ==========

// CreateEventRequest 创建事件配置请求（包含字段）
// 用途：前端一次提交事件配置和字段配置，引擎使用事务保证原子性
// 说明：EventConfig.ID 和 Fields[].ID 不需要填写，由数据库自动生成
type CreateEventRequest struct {
	EventConfig EventConfig   // 事件基本配置（不需要填ID）
	Fields      []FieldConfig // 字段列表（不需要填ID和EventConfigID）
}

// UpdateEventRequest 更新事件配置请求（包含字段，完整替换）
// 用途：前端一次提交更新事件配置和字段配置
// 策略：采用完整替换策略，先删除所有旧字段，再插入新字段（避免复杂的diff逻辑）
// 说明：EventConfig.ID 必须填写，Fields[].EventConfigID 会自动填充
type UpdateEventRequest struct {
	EventConfig EventConfig   // 事件基本配置（必须包含ID）
	Fields      []FieldConfig // 字段列表（完整替换，旧字段全部删除）
}

// EventConfigWithFields 事件配置完整信息（包含字段）
// 用途：查询事件配置时，同时返回关联的字段配置
// 说明：Fields 按 display_order 排序
type EventConfigWithFields struct {
	EventConfig EventConfig    // 事件基本配置
	Fields      []*FieldConfig // 字段列表（按DisplayOrder排序）
}
