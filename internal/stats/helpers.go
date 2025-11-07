package stats

import (
	"database/sql"
	"fmt"
)

// ========== 数据库类型转换辅助函数 ==========
// 说明：用于 Model 和 Domain 之间的类型转换

// convertNullFloat64 将 sql.NullFloat64 转换为 *float64
func convertNullFloat64(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

// convertToNullFloat64 将 *float64 转换为 sql.NullFloat64
func convertToNullFloat64(f *float64) sql.NullFloat64 {
	if f != nil {
		return sql.NullFloat64{Float64: *f, Valid: true}
	}
	return sql.NullFloat64{Valid: false}
}

// ========== SQL 辅助函数 ==========

// quoteFieldName 为字段名添加双引号（PostgreSQL标识符转义）
// 用途：防止字段名与SQL关键字冲突
func quoteFieldName(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
