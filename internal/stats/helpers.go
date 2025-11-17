package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// 数据库类型转换辅助函数
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

// SQL 辅助函数

// quoteFieldName 为字段名添加双引号（PostgreSQL标识符转义）
// 用途：防止字段名与SQL关键字冲突
func quoteFieldName(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// convertUserStatsUserIDs 转换 UserStats 中的用户ID（字符串格式的远程ID → 本地ID）
// 说明：
// - GetUserStats 返回的用户ID是字符串格式的远程用户ID（来自 slp_user_id）
// - 此函数将所有用户ID（UserMetrics[].UserID）批量转换为本地用户ID
// - 转换策略：提取 → 去重 → 批量查询映射 → 替换
func convertUserStatsUserIDs(
	ctx context.Context,
	stats *core.UserStats,
	fillLocalUserIDMap core.FillLocalUserIDMapFunc,
	tenantID string,
) error {
	if stats == nil || len(stats.UserMetrics) == 0 {
		return nil
	}

	// 1. 提取所有唯一的远程用户ID（字符串格式）
	remoteIDMapping := make(map[int]string)

	for _, metric := range stats.UserMetrics {
		if metric.UserID != "" {
			if id, err := strconv.Atoi(metric.UserID); err == nil && id > 0 {
				remoteIDMapping[id] = ""
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
	for _, metric := range stats.UserMetrics {
		if metric.UserID != "" {
			if remoteID, err := strconv.Atoi(metric.UserID); err == nil {
				if localID, ok := remoteIDMapping[remoteID]; ok && localID != "" {
					metric.UserID = localID
				} else {
					return fmt.Errorf("用户ID %s 无本地映射: %w", metric.UserID, core.ErrUserMappingNotFound)
				}
			}
		}
	}

	return nil
}
