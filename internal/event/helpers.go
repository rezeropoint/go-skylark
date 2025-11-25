package event

import (
	"database/sql"
	"regexp"
)

// 数据库类型转换辅助函数
// 说明：用于 Model 和 Domain 之间的类型转换

// convertNullString 将 sql.NullString 转换为 *string
func convertNullString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// convertToNullString 将 *string 转换为 sql.NullString
func convertToNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{Valid: false}
}

// 字段验证辅助函数

// 字段名验证正则表达式（支持中文、字母、数字、下划线，允许数字开头）
// Unicode 范围：\p{L} 匹配所有 Unicode 字母（包括中文），\p{N} 匹配所有 Unicode 数字
// 说明：Skylark 平台生成的随机字段名（如 9RbfBD、C3eib9）可能以数字开头
var fieldNameRegex = regexp.MustCompile(`^[\p{L}\p{N}_]+$`)

// isValidFieldName 验证字段名格式是否有效
// 允许：中文、英文字母、数字、下划线，允许数字开头（兼容 Skylark 随机字段名）
func isValidFieldName(name string) bool {
	return fieldNameRegex.MatchString(name)
}
