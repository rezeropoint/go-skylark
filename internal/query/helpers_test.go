package query

import (
	"testing"

	"github.com/rezeropoint/go-skylark/v2/core"
)

func TestValidateFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		wantErr   bool
	}{
		// 有效的字段名
		{"中文字段名", "文本", false},
		{"中文+数字", "文本123", false},
		{"中文+下划线", "文本_字段", false},
		{"英文字段名", "field", false},
		{"英文+数字", "field123", false},
		{"英文+下划线", "field_name", false},
		{"下划线开头", "_field", false},
		{"下划线开头+中文", "_文本", false},
		{"混合中英文", "用户user_name", false},
		{"Skylark系统字段", "slp_journey_id", false},
		{"数字开头（Skylark随机字段名）", "9RbfBD", false},
		{"数字+字母混合", "123field", false},
		{"字母+数字混合", "C3eib9", false},

		// 无效的字段名
		{"空字符串", "", true},
		{"包含空格", "field name", true},
		{"包含特殊字符", "field-name", true},
		{"包含点号", "field.name", true},
		{"SQL注入尝试1", "field; DROP TABLE", true},
		{"SQL注入尝试2", "field' OR '1'='1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldName(tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFieldName(%q) error = %v, wantErr %v", tt.fieldName, err, tt.wantErr)
			}
			if err != nil && err != core.ErrInvalidFieldName {
				t.Errorf("validateFieldName(%q) = %v, want %v", tt.fieldName, err, core.ErrInvalidFieldName)
			}
		})
	}
}

func TestQuoteFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      string
	}{
		{"普通字段", "field", `"field"`},
		{"中文字段", "文本", `"文本"`},
		{"字母+数字混合", "C3eib9", `"C3eib9"`},
		{"数字开头（Skylark随机字段名）", "9RbfBD", `"9RbfBD"`},
		{"系统字段", "slp_journey_id", `"slp_journey_id"`},
		{"下划线字段", "_field_name", `"_field_name"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteFieldName(tt.fieldName)
			if got != tt.want {
				t.Errorf("quoteFieldName(%q) = %q, want %q", tt.fieldName, got, tt.want)
			}
		})
	}
}
