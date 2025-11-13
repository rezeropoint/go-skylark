package query

import (
	"fmt"
	"regexp"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// validateFieldName 验证字段名（防SQL注入）
// 正则：只允许字母、数字、下划线，且必须以字母或下划线开头
func validateFieldName(fieldName string) error {
	if fieldName == "" {
		return core.ErrInvalidFieldName
	}

	// 字段名正则验证：^[a-zA-Z_][a-zA-Z0-9_]*$
	pattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !pattern.MatchString(fieldName) {
		return core.ErrInvalidFieldName
	}

	return nil
}

// quoteFieldName 给字段名添加双引号（支持PostgreSQL大小写敏感字段）
// 例如：C3eib9 → "C3eib9"，防止PostgreSQL自动转小写
func quoteFieldName(fieldName string) string {
	return fmt.Sprintf("\"%s\"", fieldName)
}

// extractUserIDs 提取用户 ID 列表（去重）
func extractUserIDs(assignments []*assignmentRow) []string {
	if len(assignments) == 0 {
		return nil
	}

	// 快路径：当最终只有 0 或 1 个唯一用户ID时，避免构建哈希集合
	var firstID string
	haveFirst := false
	var distinctSet map[string]struct{}

	for _, assignment := range assignments {
		if assignment.UserID.Valid && assignment.UserID.String != "" {
			id := assignment.UserID.String
			if !haveFirst {
				firstID = id
				haveFirst = true
				continue
			}
			if id == firstID {
				continue
			}
			if distinctSet == nil {
				distinctSet = make(map[string]struct{})
				distinctSet[firstID] = struct{}{}
			}
			distinctSet[id] = struct{}{}
		}
	}

	if !haveFirst {
		return nil
	}
	if distinctSet == nil {
		return []string{firstID}
	}

	userIDs := make([]string, 0, len(distinctSet))
	for id := range distinctSet {
		userIDs = append(userIDs, id)
	}
	return userIDs
}

// buildColumnInfo 构建列定义
func buildColumnInfo(fieldConfigs []*core.FieldConfig) []*core.ColumnInfo {
	// 系统字段（只返回3个核心字段）
	columns := []*core.ColumnInfo{
		{Field: core.RemoteJourneyIDField, DisplayName: "Journey ID", Type: "number"},
		{Field: core.RemoteStatusField, DisplayName: "状态", Type: "string"},
		{Field: core.VertexNameColumn, DisplayName: "当前节点", Type: "string"},
	}

	// 业务字段（只包含可见字段）
	for _, field := range fieldConfigs {
		if field.IsVisible {
			columns = append(columns, &core.ColumnInfo{
				Field:       field.FieldName,
				DisplayName: field.DisplayName,
				Type:        field.FieldType,
			})
		}
	}

	return columns
}
