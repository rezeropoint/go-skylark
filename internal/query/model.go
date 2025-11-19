package query

import (
	"database/sql"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// assignmentRow 用于扫描 Assignment 查询结果（数据库模型）
// 对应 Skylark 远程表：assignments_{flow_id}
type assignmentRow struct {
	AssignmentID int            `db:"slp_assignment_id"` // Assignment ID
	JourneyID    int            `db:"slp_journey_id"`    // Journey ID（流程实例）
	Status       string         `db:"slp_status"`        // 状态
	VertexID     int            `db:"slp_vertex_id"`     // 节点ID
	VertexName   sql.NullString `db:"vertex_name"`       // 节点名称（JOIN vertices表）
	VertexAlias  sql.NullString `db:"vertex_alias"`      // 节点别名（JOIN vertices表）
	UserID       sql.NullString `db:"slp_user_id"`       // 处理人ID
	CreatedAt    time.Time      `db:"slp_created_at"`    // 创建时间
	UpdatedAt    time.Time      `db:"slp_updated_at"`    // 更新时间
	BusinessData string         `db:"business_data"`     // 业务数据（JSON字符串）
}

// fieldMetadataModel 用于扫描字段元数据查询结果（数据库模型）
// 对应 PostgreSQL information_schema.columns 表
type fieldMetadataModel struct {
	ColumnName string `db:"column_name"` // 列名
	DataType   string `db:"data_type"`   // 数据类型
}

// ToDomain 转换为领域模型
func (m *fieldMetadataModel) ToDomain() *core.FieldMetadata {
	return &core.FieldMetadata{
		FieldName: m.ColumnName,
		DataType:  m.DataType,
		IsSystem:  core.IsSystemField(m.ColumnName), // 自动判断是否为系统字段
	}
}

// flowInfoModel 用于扫描 Flow 信息查询结果（数据库模型）
// 对应远程 Skylark 数据库 flows 表
type flowInfoModel struct {
	ID          int `db:"id"`           // 流程ID
	Title       string `db:"title"`      // 流程名称
	NamespaceID int `db:"namespace_id"` // 命名空间ID
}

// ToDomain 转换为领域模型
func (m *flowInfoModel) ToDomain() *core.FlowInfo {
	return &core.FlowInfo{
		ID:          m.ID,
		Title:       m.Title,
		NamespaceID: m.NamespaceID,
	}
}
