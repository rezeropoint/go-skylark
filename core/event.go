// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：事件配置管理、字段配置）
package core

import (
	"context"
	"fmt"
	"time"
)

// Skylark 远程数据库表名和字段名常量
const (
	// 远程表名前缀（动态表名格式：assignments_{flow_id}）
	RemoteTableNamePrefix = "assignments_"

	// 远程表固定字段名（assignments_{flow_id} 表）
	RemotePrimaryKeyField = "slp_assignment_id" // Assignment主键
	RemoteJourneyIDField  = "slp_journey_id"    // Journey ID（用于聚合查询）
	RemoteStatusField     = "slp_status"        // 状态字段
	RemoteVertexIDField   = "slp_vertex_id"     // 节点ID字段
	RemoteUserIDField     = "slp_user_id"       // 处理人ID字段
	RemoteCreatedAtField  = "slp_created_at"    // 创建时间字段
	RemoteUpdatedAtField  = "slp_updated_at"    // 更新时间字段

	// CacheEventConfigKeyPrefix 事件配置缓存键前缀
	// key格式: skylark:event_config:{tenant_id}:{id}
	// TTL: 10分钟（配置偶尔变更）
	CacheEventConfigKeyPrefix = "skylark:event_config:"

	// CacheEventConfigListKeyPrefix 事件配置列表缓存键前缀
	// key格式: skylark:event_list:{tenant_id}:enabled={true|false|all}
	// TTL: 5分钟（列表实时性要求较高）
	CacheEventConfigListKeyPrefix = "skylark:event_list:"

	// CacheConfiguredFlowIDsListKeyPrefix 已配置 flow_id 列表缓存键前缀
	// key格式: skylark:configured_flows:{tenant_id}:enabled={true|false|all}
	// TTL: 5分钟（与事件配置列表缓存一致）
	CacheConfiguredFlowIDsListKeyPrefix = "skylark:configured_flows:"

	// CacheFieldConfigsByFlowIDKeyPrefix 通过 flowID 查询的字段配置缓存键前缀
	// key格式: skylark:field_configs:{tenant_id}:{flow_id}
	// TTL: 10分钟（与事件配置缓存一致）
	CacheFieldConfigsByFlowIDKeyPrefix = "skylark:field_configs:"
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
	return fmt.Sprintf("%s%d", RemoteTableNamePrefix, e.FlowID)
}

// GetRemotePrimaryKey 获取远程表主键字段名
// 固定为 slp_assignment_id（Assignment是真正的主键）
func (e *EventConfig) GetRemotePrimaryKey() string {
	return RemotePrimaryKeyField
}

// GetJourneyIDField 获取Journey ID字段名
// Journey ID用于聚合查询（多个Assignment共享同一个Journey）
func (e *EventConfig) GetJourneyIDField() string {
	return RemoteJourneyIDField
}

// 聚合数据结构（用于事件+字段统一管理）

// EventCreation 事件创建聚合（包含事件配置+字段配置）
// DDD聚合根：事件配置和字段配置作为一个事务单元创建
// 说明：EventConfig.ID 和 Fields[].ID 不需要填写，由数据库自动生成
type EventCreation struct {
	EventConfig EventConfig   // 事件基本配置（不需要填ID）
	Fields      []FieldConfig // 字段列表（不需要填ID和EventConfigID）
}

// EventUpdate 事件更新聚合（包含事件配置+字段配置）
// DDD聚合根：事件配置和字段配置作为一个事务单元更新
// 策略：采用完整替换策略，先删除所有旧字段，再插入新字段（避免复杂的diff逻辑）
// 说明：EventConfig.ID 必须填写，Fields[].EventConfigID 会自动填充
type EventUpdate struct {
	EventConfig EventConfig   // 事件基本配置（必须包含ID）
	Fields      []FieldConfig // 字段列表（完整替换，旧字段全部删除）
}

// EventAggregate 事件聚合根（查询结果）
// DDD聚合根：包含事件配置及其所有关联的字段配置
// 说明：Fields 按 display_order 排序
type EventAggregate struct {
	EventConfig EventConfig    // 事件基本配置
	Fields      []*FieldConfig // 字段列表（按DisplayOrder排序）
}

// ListConfiguredFlowIDsFunc 获取已配置事件的 flow_id 列表的函数类型
// 用途：供 flows 等 Manager 筛选已配置事件监控的流程，实现 Manager 之间解耦
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID（用于查询事件配置）
//   - enabled: 筛选条件（nil=全部，true=已启用，false=已禁用）
//
// 返回：
//   - []int: 已配置的 flow_id 列表（去重）
//   - error: 错误信息（如查询失败等）
//
// 说明：
//   - 该函数会从 event_configs 表查询所有匹配的 flow_id
//   - 自动去重（同一个 flow_id 只返回一次）
//   - 用于筛选流程实例数据，确保只返回已配置事件的流程
type ListConfiguredFlowIDsFunc func(ctx context.Context, tenantID string, enabled *bool) ([]int, error)
