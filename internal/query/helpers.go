package query

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

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
		if assignment.UserID.Valid {
			id := strconv.FormatInt(assignment.UserID.Int64, 10)
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

// convertDetailResponseUserIDs 转换 DetailResponse 中的用户ID（字符串格式的远程ID → 本地ID）
// 说明：
// - GetEventDetail 返回的用户ID是字符串格式的远程用户ID（来自 slp_user_id）
// - 此函数将所有用户ID（InitiatorUserID 和 FlowHistory[].UserIDs）批量转换为本地用户ID
// - 转换策略：提取 → 去重 → 批量查询映射 → 替换
func convertDetailResponseUserIDs(
	ctx context.Context,
	detail *core.DetailResponse,
	fillLocalUserIDMap core.FillLocalUserIDMapFunc,
	tenantID string,
) error {
	if detail == nil {
		return nil
	}

	// 1. 提取所有唯一的远程用户ID（字符串格式）
	remoteIDMapping := make(map[int]string)

	// 发起人ID
	if detail.InitiatorUserID != "" {
		if id, err := strconv.Atoi(detail.InitiatorUserID); err == nil && id > 0 {
			remoteIDMapping[id] = ""
		}
	}

	// FlowHistory 中的 UserIDs
	for _, node := range detail.FlowHistory {
		for _, userIDStr := range node.UserIDs {
			if userIDStr != "" {
				if id, err := strconv.Atoi(userIDStr); err == nil && id > 0 {
					remoteIDMapping[id] = ""
				}
			}
		}
	}

	if len(remoteIDMapping) == 0 {
		return nil // 没有需要转换的用户ID
	}

	// 2. 批量转换（填充映射）
	if err := fillLocalUserIDMap(ctx, tenantID, &remoteIDMapping); err != nil {
		return fmt.Errorf("批量转换用户ID失败: %w", err)
	}

	// 3. 替换所有用户ID为本地ID
	// 3.1 替换发起人ID
	if detail.InitiatorUserID != "" {
		if remoteID, err := strconv.Atoi(detail.InitiatorUserID); err == nil {
			if localID, ok := remoteIDMapping[remoteID]; ok && localID != "" {
				detail.InitiatorUserID = localID
			} else {
				return fmt.Errorf("发起人ID %s 无本地映射: %w", detail.InitiatorUserID, core.ErrUserMappingNotFound)
			}
		}
	}

	// 3.2 替换 FlowHistory 中的用户ID
	for _, node := range detail.FlowHistory {
		for i, userIDStr := range node.UserIDs {
			if userIDStr != "" {
				if remoteID, err := strconv.Atoi(userIDStr); err == nil {
					if localID, ok := remoteIDMapping[remoteID]; ok && localID != "" {
						node.UserIDs[i] = localID
					} else {
						return fmt.Errorf("用户ID %s 无本地映射: %w", userIDStr, core.ErrUserMappingNotFound)
					}
				}
			}
		}
	}

	return nil
}
