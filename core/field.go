// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求 + 🟢 数据库查询
// 说明：本文件包含两部分内容：
// - 🔵 API 请求相关：TypedValue, FieldMapping, FieldOption（用于调用 Skylark REST API 时的字段值封装和选项处理）
// - 🟢 数据库查询相关：FieldConfig（用于配置 Skylark PostgreSQL 查询引擎的字段映射）
package core

import "time"

// ==================== 🔵 API 请求相关类型（用于调用 Skylark REST API） ====================

// TypedValue 带类型的值（用于 API 请求中的字段值封装）
// 用途：在调用 Flows/Forms API 时，封装字段值及其类型（如图片URL、Base64等）
type TypedValue struct {
	Type  string // 值类型：string/imageURL/imageBase64
	Value any    // 实际值
}

// FieldType 定义字段值类型枚举（用于 TypedValue.Type）
type FieldType string

const (
	FieldString      FieldType = "string"      // 普通字符串
	FieldImage       FieldType = "imageURL"    // 图片URL
	FieldImageBase64 FieldType = "imageBase64" // Base64编码的图片
	// 以后还可以加更多类型
)

// FieldMapping 字段映射（用于 API 请求中的字段元数据）
// 用途：存储从 Skylark API 获取的字段信息，包括字段类型、可选项等
type FieldMapping struct {
	ID          int           `json:"id"`                // 字段ID
	IdentityKey string        `json:"identity_key"`      // 字段唯一标识键
	Type        string        `json:"type"`              // 字段类型（如 Field::RadioButton）
	Options     []FieldOption `json:"options,omitempty"` // 字段可选项（仅选项类型字段有值）
}

// FieldOption 字段选项（用于 API 请求中的选项类型字段）
// 用途：表示 RadioButton、Checkbox、SelectField 等选项字段的可选值
type FieldOption struct {
	ID       int            `json:"id"`                 // 选项ID
	Value    string         `json:"value"`              // 选项值
	Settings map[string]any `json:"settings,omitempty"` // 选项设置
	Position int            `json:"position"`           // 选项位置
}

// Field 流程字段集合（用于 API 响应）
// 用途：表示一个流程包含的所有字段信息
type Field struct {
	ID     int            `json:"id"`     // 流程ID
	Title  string         `json:"title"`  // 流程标题
	Fields []FieldMapping `json:"fields"` // 流程包含的字段映射列表
}

// IsOptionField 判断字段类型是否为选项类型（用于 API 字段处理）
// 用途：判断字段是否为 RadioButton、Checkbox、SelectField、MultipleSelectField
// 参数:
//   - fieldType: 字段类型（如 "Field::RadioButton"）
func IsOptionField(fieldType string) bool {
	return fieldType == FieldTypeRadioButton ||
		fieldType == FieldTypeCheckbox ||
		fieldType == FieldTypeSelectField ||
		fieldType == FieldTypeMultipleSelectField
}

// ==================== 🟢 数据库查询相关类型（用于配置 Skylark 查询引擎） ====================

// FieldConfig 字段配置领域模型（用于 Skylark PostgreSQL 查询）
// 用途：配置查询引擎的字段映射，定义如何从 assignments_{flow_id} 表中查询和展示字段
// 特点：纯领域模型（无 db 标签、无 sql.Null* 类型）
type FieldConfig struct {
	ID            string    // 配置UUID
	EventConfigID string    // 关联事件配置ID
	FieldName     string    // 远程表字段名（对应 assignments_{flow_id} 表的列名）
	DisplayName   string    // 展示名称（给用户看的名称）
	FieldType     string    // 字段类型：string/number/date/datetime/boolean
	IsVisible     bool      // 是否在列表页展示
	DisplayOrder  int       // 展示顺序（升序）
	IsSearchable  bool      // 是否可搜索（用于构建 WHERE 子句）
	CreatedAt     time.Time // 创建时间
	UpdatedAt     time.Time // 更新时间
}

// FieldConfig.FieldType 字段类型常量（用于数据库查询配置）
// 说明：定义字段的数据类型，用于查询时的类型转换和校验
const (
	FieldTypeString   = "string"   // 字符串类型
	FieldTypeNumber   = "number"   // 数字类型
	FieldTypeDate     = "date"     // 日期类型
	FieldTypeDatetime = "datetime" // 日期时间类型
	FieldTypeBoolean  = "boolean"  // 布尔类型
)

// IsValidFieldType 验证字段类型是否有效（用于创建/更新字段配置时校验）
// 参数:
//   - fieldType: 字段类型字符串
//
// 返回:
//   - bool: 是否为有效的字段类型
func IsValidFieldType(fieldType string) bool {
	switch fieldType {
	case FieldTypeString, FieldTypeNumber, FieldTypeDate, FieldTypeDatetime, FieldTypeBoolean:
		return true
	default:
		return false
	}
}

// ==================== 🔵 API 相关常量 ====================

// Skylark API 返回的字段类型常量（用于 FieldMapping.Type）
// 说明：这些是 Skylark API 返回的 Field::XXX 格式的字段类型
const (
	FieldTypeRadioButton         = "Field::RadioButton"         // 单选按钮字段
	FieldTypeCheckbox            = "Field::Checkbox"            // 复选框字段
	FieldTypeSelectField         = "Field::SelectField"         // 下拉选择字段
	FieldTypeMultipleSelectField = "Field::MultipleSelectField" // 多选下拉字段
)

// 字段后缀常量（用于识别 API 中的图片字段）
// 说明：Skylark API 约定以特定后缀标识图片字段
const (
	SuffixImage                = "_Img"       // 图片URL字段后缀（如：avatar_Img）
	SuffixBase64Image          = "_Base64Img" // Base64图片字段后缀（如：avatar_Base64Img）
	MinSuffixImageLength       = 4            // 图片后缀最小长度
	MinSuffixBase64ImageLength = 10           // Base64图片后缀最小长度
)

// 缓存键前缀常量（用于 API 字段映射缓存）
const (
	// CacheFieldMappingKeyPrefix 字段映射缓存键前缀（flows/forms 模块使用）
	// 完整格式：skylark:field_mapping:{app}:{flowID}
	CacheFieldMappingKeyPrefix = "skylark:field_mapping:"
)
