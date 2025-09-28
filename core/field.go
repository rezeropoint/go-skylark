package core

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

// ————以下是Skylark的类型————

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
