package event

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/core"
)

// EventConfigModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射，允许使用框架类型
type EventConfigModel struct {
	ID           string         `db:"id"`             // 配置UUID
	Name         string         `db:"name"`           // 事件显示名称（自定义名称）
	FlowID       int            `db:"flow_id"`        // 远程流程ID（关联flows.id）
	FlowTitle    string         `db:"flow_title"`     // 远程流程名称（冗余，来自flows.title）
	OrgFieldName sql.NullString `db:"org_field_name"` // 组织字段名（用于权限过滤，可选）
	Description  sql.NullString `db:"description"`    // 描述
	Enabled      bool           `db:"enabled"`        // 是否启用
	TenantID     string         `db:"tenant_id"`      // 租户ID
	CreatedBy    sql.NullString `db:"created_by"`     // 创建者用户ID
	UpdatedBy    sql.NullString `db:"updated_by"`     // 最后修改者用户ID
	CreatedAt    time.Time      `db:"created_at"`     // 创建时间
	UpdatedAt    time.Time      `db:"updated_at"`     // 更新时间
}

// FieldConfigModel 是数据库查询专用结构体（基础设施层）
type FieldConfigModel struct {
	ID            string    `db:"id"`              // 配置UUID
	EventConfigID string    `db:"event_config_id"` // 关联事件配置ID
	FieldName     string    `db:"field_name"`      // 远程表字段名
	DisplayName   string    `db:"display_name"`    // 展示名称
	FieldType     string    `db:"field_type"`      // 字段类型
	IsVisible     bool      `db:"is_visible"`      // 是否在列表页展示
	DisplayOrder  int       `db:"display_order"`   // 展示顺序
	IsSearchable  bool      `db:"is_searchable"`   // 是否可搜索
	CreatedAt     time.Time `db:"created_at"`      // 创建时间
	UpdatedAt     time.Time `db:"updated_at"`      // 更新时间
}

// ToDomain 将数据库模型转换为领域模型
func (m *EventConfigModel) ToDomain() *core.EventConfig {
	return &core.EventConfig{
		ID:           m.ID,
		Name:         m.Name,
		FlowID:       m.FlowID,
		FlowTitle:    m.FlowTitle,
		OrgFieldName: convertNullString(m.OrgFieldName),
		Description:  convertNullString(m.Description),
		Enabled:      m.Enabled,
		TenantID:     m.TenantID,
		CreatedBy:    convertNullString(m.CreatedBy),
		UpdatedBy:    convertNullString(m.UpdatedBy),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// ToDomain 将数据库模型转换为领域模型
func (m *FieldConfigModel) ToDomain() *core.FieldConfig {
	return &core.FieldConfig{
		ID:            m.ID,
		EventConfigID: m.EventConfigID,
		FieldName:     m.FieldName,
		DisplayName:   m.DisplayName,
		FieldType:     m.FieldType,
		IsVisible:     m.IsVisible,
		DisplayOrder:  m.DisplayOrder,
		IsSearchable:  m.IsSearchable,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// FromDomainEvent 将领域模型转换为数据库模型
func FromDomainEvent(config *core.EventConfig) *EventConfigModel {
	return &EventConfigModel{
		ID:           config.ID,
		Name:         config.Name,
		FlowID:       config.FlowID,
		FlowTitle:    config.FlowTitle,
		OrgFieldName: convertToNullString(config.OrgFieldName),
		Description:  convertToNullString(config.Description),
		Enabled:      config.Enabled,
		TenantID:     config.TenantID,
		CreatedBy:    convertToNullString(config.CreatedBy),
		UpdatedBy:    convertToNullString(config.UpdatedBy),
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	}
}

// FromDomainField 将领域模型转换为数据库模型
func FromDomainField(field *core.FieldConfig) *FieldConfigModel {
	return &FieldConfigModel{
		ID:            field.ID,
		EventConfigID: field.EventConfigID,
		FieldName:     field.FieldName,
		DisplayName:   field.DisplayName,
		FieldType:     field.FieldType,
		IsVisible:     field.IsVisible,
		DisplayOrder:  field.DisplayOrder,
		IsSearchable:  field.IsSearchable,
		CreatedAt:     field.CreatedAt,
		UpdatedAt:     field.UpdatedAt,
	}
}

// GetRemoteTableName 获取远程表名
func (m *EventConfigModel) GetRemoteTableName() string {
	return fmt.Sprintf("assignments_%d", m.FlowID)
}
