package event

import (
	"testing"
)

func TestIsValidFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		// 有效的字段名
		{"中文字段名", "文本", true},
		{"中文+数字", "文本123", true},
		{"中文+下划线", "文本_字段", true},
		{"英文字段名", "field", true},
		{"英文+数字", "field123", true},
		{"英文+下划线", "field_name", true},
		{"下划线开头", "_field", true},
		{"下划线开头+中文", "_文本", true},
		{"混合中英文", "用户user_name", true},

		// 无效的字段名
		{"数字开头", "123field", false},
		{"空字符串", "", false},
		{"只有空格", "   ", false},
		{"包含空格", "field name", false},
		{"包含特殊字符", "field-name", false},
		{"包含点号", "field.name", false},
		{"包含@符号", "field@name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFieldName(tt.fieldName)
			if got != tt.want {
				t.Errorf("isValidFieldName(%q) = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}
