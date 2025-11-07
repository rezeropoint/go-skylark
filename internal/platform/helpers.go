package platform

import "database/sql"

// ========== 数据库类型转换辅助函数 ==========
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
