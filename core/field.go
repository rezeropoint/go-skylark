// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求 + 🟢 数据库查询
// 说明：本文件包含两部分内容：
// - 🔵 API 请求相关：TypedValue, FieldMapping, FieldOption（用于调用 API 时的字段值封装）
// - 🟢 数据库查询相关：FieldConfig（用于配置查询时的字段映射）
package core

import "time"

// ================================================================
// 🔵 API 请求相关类型（用于调用 Skylark REST API）
// ================================================================

// FieldType 定义字段类型枚举
type FieldType string

const (
	FieldString      FieldType = "string"
	FieldImage       FieldType = "imageURL"
	FieldImageBase64 FieldType = "imageBase64"
	// 以后还可以加更多类型
)

type TypedValue struct {
	Type  string
	Value any
}

// isOptionField 判断字段类型是否为选项类型
// 参数:
//   - fieldType: 字段类型
func IsOptionField(fieldType string) bool {
	return fieldType == FieldTypeRadioButton ||
		fieldType == FieldTypeCheckbox ||
		fieldType == FieldTypeSelectField ||
		fieldType == FieldTypeMultipleSelectField
}

// FieldMapping 结构用于存储字段映射
type FieldMapping struct {
	ID          int           `json:"id"`                // 字段ID
	IdentityKey string        `json:"identity_key"`      // 字段唯一标识键
	Type        string        `json:"type"`              // 字段类型
	Options     []FieldOption `json:"options,omitempty"` // 字段可选项
}

// FieldOption 表示字段的可选选项
type FieldOption struct {
	ID       int            `json:"id"`                 // 选项ID
	Value    string         `json:"value"`              // 选项值
	Settings map[string]any `json:"settings,omitempty"` // 选项设置
	Position int            `json:"position"`           // 选项位置
}

// Field 表示字段
type Field struct {
	ID     int            `json:"id"`     // 流程ID
	Title  string         `json:"title"`  // 流程标题
	Fields []FieldMapping `json:"fields"` // 流程包含的字段
}

// ================================================================
// 🟢 数据库查询相关类型（用于配置 Skylark 查询引擎）
// ================================================================

// FieldConfig 字段配置领域模型
type FieldConfig struct {
	ID            string    `db:"id"`              // 配置UUID
	EventConfigID string    `db:"event_config_id"` // 关联事件配置ID
	FieldName     string    `db:"field_name"`      // 远程表字段名
	DisplayName   string    `db:"display_name"`    // 展示名称
	FieldType     string    `db:"field_type"`      // 字段类型：string/number/date/datetime/boolean
	IsVisible     bool      `db:"is_visible"`      // 是否在列表页展示
	DisplayOrder  int       `db:"display_order"`   // 展示顺序
	IsSearchable  bool      `db:"is_searchable"`   // 是否可搜索
	CreatedAt     time.Time `db:"created_at"`      // 创建时间
	UpdatedAt     time.Time `db:"updated_at"`      // 更新时间
}

// FieldType 字段类型常量
const (
	FieldTypeString   = "string"
	FieldTypeNumber   = "number"
	FieldTypeDate     = "date"
	FieldTypeDatetime = "datetime"
	FieldTypeBoolean  = "boolean"
)

// IsValidFieldType 验证字段类型是否有效
func IsValidFieldType(fieldType string) bool {
	switch fieldType {
	case FieldTypeString, FieldTypeNumber, FieldTypeDate, FieldTypeDatetime, FieldTypeBoolean:
		return true
	default:
		return false
	}
}
