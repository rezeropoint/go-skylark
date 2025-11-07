package stats

import (
	"fmt"
)

// quoteFieldName 给字段名添加双引号（支持PostgreSQL大小写敏感字段）
// 例如：C3eib9 → "C3eib9"，防止PostgreSQL自动转小写
func quoteFieldName(fieldName string) string {
	return fmt.Sprintf("\"%s\"", fieldName)
}
