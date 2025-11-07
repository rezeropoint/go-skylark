package flows

import (
	"github.com/rezeropoint/go-skylark/core"
)

// FlowInfoModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射
type FlowInfoModel struct {
	ID          int    `db:"id"`           // 流程ID
	Title       string `db:"title"`        // 流程名称
	NamespaceID int    `db:"namespace_id"` // 命名空间ID
}

// FieldMetadataModel 是数据库查询专用结构体（基础设施层）
type FieldMetadataModel struct {
	FieldName string `db:"column_name"` // 字段名
	DataType  string `db:"data_type"`   // 数据类型（PostgreSQL类型）
}

// ToDomain 将数据库模型转换为领域模型
func (m *FlowInfoModel) ToDomain() *core.FlowInfo {
	return &core.FlowInfo{
		ID:          m.ID,
		Title:       m.Title,
		NamespaceID: m.NamespaceID,
	}
}

// ToDomain 将数据库模型转换为领域模型
func (m *FieldMetadataModel) ToDomain() *core.FieldMetadata {
	return &core.FieldMetadata{
		FieldName: m.FieldName,
		DataType:  m.DataType,
		IsSystem:  core.IsSystemField(m.FieldName),
	}
}
