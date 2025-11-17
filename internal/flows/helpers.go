package flows

import (
	"context"
	"fmt"

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
			return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 3. 使用映射转换为领域模型
	journeys := make([]*core.Journey, len(journeyResponses))
	for i, jr := range journeyResponses {
		journeys[i] = jr.ToDomain(userIDMapping)
	}

	return journeys, nil
}

// convertJourneyUserID 转换单个 Journey 的用户ID（用于 GetJourneyBySN）
//
// 流程：
//  1. 提取远程用户ID
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 使用本地用户ID映射调用 ToDomain 转换
//
// 参数：
//   - ctx: 上下文
//   - journeyResponse: API 响应
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - *core.Journey: 领域模型（用户ID已转换为本地ID）
//   - error: 如果远程ID无映射，返回 core.ErrUserMappingNotFound
func convertJourneyUserID(ctx context.Context, journeyResponse *JourneyResponse, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) (*core.Journey, error) {
	// 1. 提取远程用户ID
	userIDMapping := map[int]string{
		int(journeyResponse.User.ID): "",
	}

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
		return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
	}

	// 3. 使用映射转换为领域模型
	return journeyResponse.ToDomain(userIDMapping), nil
}

// convertJourneyDetailUserID 转换 JourneyDetail 的发起人用户ID（用于 GetJourneyDetail）
//
// 流程：
//  1. 提取远程用户ID（发起人）
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 替换 Initiator.ID 为本地用户ID
//
// 参数：
//   - ctx: 上下文
//   - journeyDetail: 领域模型（Initiator.ID 为远程ID）
//   - remoteUserID: 远程用户ID
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - error: 如果远程ID无映射，返回 core.ErrUserMappingNotFound
func convertJourneyDetailUserID(ctx context.Context, journeyDetail *core.JourneyDetail, remoteUserID int64, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) error {
	// 1. 提取远程用户ID
	userIDMapping := map[int]string{
		int(remoteUserID): "",
	}

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
		return fmt.Errorf("批量转换用户ID失败: %w", err)
	}

	// 3. 替换 Initiator.ID 为本地用户ID
	if journeyDetail.Initiator != nil {
		journeyDetail.Initiator.ID = userIDMapping[int(remoteUserID)]
	}

	return nil
}

// convertAssignmentsUserIDs 批量转换 Assignment 列表的用户ID（用于 GetJourneyAssignments/GetUserAssignments）
//
// 流程：
//  1. 提取所有唯一的远程用户ID（AssigneeID）
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 替换每个 Assignment.AssigneeID 为本地用户ID
//
// 参数：
//   - ctx: 上下文
//   - assignments: Assignment 列表（AssigneeID 为字符串形式的远程ID）
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
func convertAssignmentsUserIDs(ctx context.Context, assignments []*core.Assignment, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) error {
	if len(assignments) == 0 {
		return nil
	}

	// 1. 提取所有唯一的远程用户ID（使用通用函数）
	// Assignment.AssigneeID 是字符串形式的远程ID，需要转为 int
	userIDMapping := core.ExtractUserIDsToMap(assignments, func(a *core.Assignment) int {
		// AssigneeID 格式为 "123"（字符串形式的远程ID）
		var remoteID int
		fmt.Sscanf(a.AssigneeID, "%d", &remoteID)
		return remoteID
	})

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 3. 替换每个 Assignment.AssigneeID 为本地用户ID
	for _, assignment := range assignments {
		var remoteID int
		fmt.Sscanf(assignment.AssigneeID, "%d", &remoteID)
		assignment.AssigneeID = userIDMapping[remoteID]
	}

	return nil
}

// convertMomentsUserIDs 批量转换 Moment 列表的用户ID（用于 GetJourneyMoments）
//
// 流程：
//  1. 提取所有唯一的远程用户ID（OperatorID）
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 替换每个 Moment.OperatorID 为本地用户ID
//
// 参数：
//   - ctx: 上下文
//   - moments: Moment 列表（OperatorID 为字符串形式的远程ID）
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
func convertMomentsUserIDs(ctx context.Context, moments []*core.Moment, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) error {
	if len(moments) == 0 {
		return nil
	}

	// 1. 提取所有唯一的远程用户ID（使用通用函数）
	// Moment.OperatorID 是字符串形式的远程ID，需要转为 int
	userIDMapping := core.ExtractUserIDsToMap(moments, func(m *core.Moment) int {
		// OperatorID 格式为 "123"（字符串形式的远程ID）
		var remoteID int
		fmt.Sscanf(m.OperatorID, "%d", &remoteID)
		return remoteID
	})

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 3. 替换每个 Moment.OperatorID 为本地用户ID
	for _, moment := range moments {
		var remoteID int
		fmt.Sscanf(moment.OperatorID, "%d", &remoteID)
		moment.OperatorID = userIDMapping[remoteID]
	}

	return nil
}

// convertProcessingUsersIDs 批量转换 ProcessingUser 列表的用户ID（用于 GetCurrentProcessingUsers）
//
// 流程：
//  1. 提取所有唯一的远程用户ID
//  2. 批量查询本地用户ID（调用 fillLocalUserIDMap 填充映射）
//  3. 替换每个 ProcessingUser.ID 为本地用户ID
//
// 参数：
//   - ctx: 上下文
//   - users: ProcessingUser 列表（ID 为字符串形式的远程ID）
//   - fillLocalUserIDMap: 批量反向转换函数
//   - tenantID: 租户ID
//
// 返回：
//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
func convertProcessingUsersIDs(ctx context.Context, users []*core.ProcessingUser, fillLocalUserIDMap core.FillLocalUserIDMapFunc, tenantID string) error {
	if len(users) == 0 {
		return nil
	}

	// 1. 提取所有唯一的远程用户ID（使用通用函数）
	// ProcessingUser.ID 是字符串形式的远程ID，需要转为 int
	userIDMapping := core.ExtractUserIDsToMap(users, func(u *core.ProcessingUser) int {
		// ID 格式为 "123"（字符串形式的远程ID）
		var remoteID int
		fmt.Sscanf(u.ID, "%d", &remoteID)
		return remoteID
	})

	// 2. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 3. 替换每个 ProcessingUser.ID 为本地用户ID
	for _, user := range users {
		var remoteID int
		fmt.Sscanf(user.ID, "%d", &remoteID)
		user.ID = userIDMapping[remoteID]
	}

	return nil
}
