package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// convertJourneyResponsesToDomain 批量转换 JourneyResponse 为领域模型（包含用户ID转换）
//
// 流程：
//  1. 提取所有唯一的远程用户ID
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 使用本地用户ID映射调用 ToDomain 转换
//
// 参数：
//   - ctx: 上下文
//   - journeyResponses: API 响应列表
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - []*core.Journey: 领域模型列表（用户ID已转换为本地ID）
//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
func convertJourneyResponsesToDomain(ctx context.Context, journeyResponses []JourneyResponse, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) ([]*core.Journey, error) {
	if len(journeyResponses) == 0 {
		return []*core.Journey{}, nil
	}

	// 1. 提取所有唯一的远程用户ID（使用通用函数）
	userIDMapping := core.ExtractUserIDsToMap(journeyResponses, func(jr JourneyResponse) int {
		return int(jr.User.ID)
	})

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return nil, fmt.Errorf("获取流程列表失败: 流程中存在未同步的 Skylark 用户，无法转换为本地用户ID (%w)", err)
		}
	}

	// 3. 使用映射转换为领域模型
	journeys := make([]*core.Journey, len(journeyResponses))
	for i, jr := range journeyResponses {
		journeys[i] = jr.ToDomain(userIDMapping)
	}

	return journeys, nil
}

// calculateMomentDuration 计算 Moment 的处理时长（秒）
// 处理时长 = UpdatedAt - CreatedAt
//
// 参数：
//   - createdAt: 创建时间（ISO 8601格式）
//   - updatedAt: 更新时间（ISO 8601格式）
//
// 返回：
//   - *int: 处理时长（秒），解析失败返回 nil
func calculateMomentDuration(createdAt, updatedAt string) *int {
	// 解析创建时间
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil
	}

	// 解析更新时间
	updated, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil
	}

	// 计算时间差（秒）
	duration := int(updated.Sub(created).Seconds())
	return &duration
}
