// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：事件数据查询、Journey 聚合）
package core

import (
	"context"
	"time"
)

// 查询缓存键常量
const (
	// CacheFlowListKeyPrefix flows 列表缓存键前缀
	// 完整格式：skylark:flows:{tenant_id}:{namespace_id}
	CacheFlowListKeyPrefix = "skylark:flows:"

	// CacheFlowFieldsKeyPrefix flow 字段列表缓存键前缀
	// 完整格式：skylark:flow_fields:{tenant_id}:{flow_id}
	CacheFlowFieldsKeyPrefix = "skylark:flow_fields:"

	// CacheUserNameKeyPrefix 用户名缓存键前缀
	// 完整格式：skylark:users:{tenant_id}:{user_id}
	CacheUserNameKeyPrefix = "skylark:users:"
)

// 查询结果列名常量（JOIN 后的别名字段）
const (
	// VertexNameColumn 节点名称列（COALESCE(v.alias_name, v.name, '')）
	VertexNameColumn = "vertex_name"

	// VertexAliasColumn 节点别名列（v.alias_name）
	VertexAliasColumn = "vertex_alias"

	// BusinessDataColumn 业务数据列（row_to_json(a)::text）
	BusinessDataColumn = "business_data"

	// RowDataColumn 行数据列（row_to_json(t)::text）
	RowDataColumn = "row_data"
)

// 排序方向常量
const (
	// SortOrderAsc 升序排序
	SortOrderAsc = "asc"

	// SortOrderDesc 降序排序
	SortOrderDesc = "desc"
)

// 函数类型定义（避免循环依赖）

// GetEventConfigWithFieldsFunc 获取事件配置（含字段）的函数类型
type GetEventConfigWithFieldsFunc func(ctx context.Context, id, tenantID string) (*EventAggregate, error)

// ListOrgMappingsFunc 获取组织映射列表的函数类型
type ListOrgMappingsFunc func(ctx context.Context, tenantID string) ([]*OrgMapping, error)

// 查询请求/响应定义

// QueryRequest 事件数据查询请求
type QueryRequest struct {
	// 事件配置ID（引擎内部加载）
	EventConfigID string // 事件配置ID

	// 权限上下文
	TenantID   string   // 租户ID
	UserOrgIDs []string // 用户可见的组织ID列表（含子组织）

	// 查询参数
	Page      int      // 页码（从1开始）
	PageSize  int      // 每页数量
	Keyword   string   // 关键词搜索
	SortField string   // 排序字段
	SortOrder string   // 排序方向：asc/desc
	Status    []string // 状态筛选（可选，支持多个状态：processing/finished/cancelled等）
}

// QueryResponse 事件数据查询响应
type QueryResponse struct {
	Columns []*ColumnInfo            // 列定义
	Records []map[string]interface{} // 数据记录（每条是一个Journey的最新Assignment）
	Total   int64                    // 总记录数（Journey数量）
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Field       string // 字段名
	DisplayName string // 展示名称
	Type        string // 字段类型
}

// DetailRequest 事件详情查询请求（返回完整流转历史）
type DetailRequest struct {
	EventConfigID string   // 事件配置ID（引擎内部加载）
	JourneyID     int      // Journey ID（事件实例ID）
	TenantID      string   // 租户ID
	UserOrgIDs    []string // 用户可见的组织ID列表（用于权限验证）
}

// DetailResponse 事件详情响应
type DetailResponse struct {
	JourneyID          int                    // Journey ID
	CurrentStatus      string                 // 当前流程状态
	InitiatorUserID    string                 // 发起人ID（第一个Assignment的UserID）
	InitiatorUserName  string                 // 发起人姓名（通过远程users表转换）
	InitiatedAt        time.Time              // 发起时间（第一个Assignment的创建时间）
	LatestBusinessData map[string]interface{} // 最新的业务数据（当前快照，只包含最后一个Assignment的业务字段）
	FlowHistory        []*FlowNode            // 流转历史（所有Assignment的节点信息，按时间正序排列）
}

// FlowNode 流程节点信息（用于展示流转历史时间线）
// 说明：
// - 相同vertexId的多个assignment会被合并为一个节点（Skylark特性：同一节点多人处理）
// - Status字段已移除（历史节点的状态无意义，只有DetailResponse.CurrentStatus有意义）
// - 支持一个节点多人处理（UserIDs和UserNames为数组）
type FlowNode struct {
	VertexID    int       // 节点ID
	VertexName  string    // 节点名称
	VertexAlias string    // 节点别名
	UserIDs     []string  // 处理人ID列表（支持同一节点多人处理，对应slp_user_id）
	UserNames   []string  // 处理人姓名列表（与UserIDs一一对应，通过远程users表查询转换）
	CreatedAt   time.Time // 该节点第一次创建时间
	UpdatedAt   time.Time // 该节点最后更新时间
}
