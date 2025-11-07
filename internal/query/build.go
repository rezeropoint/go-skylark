package query

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/logx"
)

// buildDetailResponse 组装 DetailResponse
func (m *queryManager) buildDetailResponse(assignments []*assignmentRow, userNames map[string]string, visibleFields []*core.FieldConfig) *core.DetailResponse {
	if len(assignments) == 0 {
		return nil
	}

	// 第一个 Assignment（发起人信息）
	firstAssignment := assignments[0]
	initiatorUserID := ""
	if firstAssignment.UserID.Valid {
		initiatorUserID = firstAssignment.UserID.String
	}
	initiatorUserName := userNames[initiatorUserID]

	// 最后一个 Assignment（当前状态）
	lastAssignment := assignments[len(assignments)-1]
	currentStatus := lastAssignment.Status

	// 查找包含有效业务数据的 Assignment（优先从第一个开始查找）
	// Skylark 特性：业务数据通常在开始节点，后续节点可能只包含处理信息
	allBusinessData := make(map[string]interface{})
	for _, assignment := range assignments {
		if assignment.BusinessData != "" {
			tempData := make(map[string]interface{})
			if err := json.Unmarshal([]byte(assignment.BusinessData), &tempData); err != nil {
				logx.Error("解析业务数据JSON失败:", err)
				continue
			}
			// 合并业务数据（后面的覆盖前面的，保留最新非空值）
			for key, value := range tempData {
				if value != nil {
					allBusinessData[key] = value
				}
			}
		}
	}

	// 调试日志：输出业务数据解析情况
	businessDataKeys := make([]string, 0, len(allBusinessData))
	for key := range allBusinessData {
		businessDataKeys = append(businessDataKeys, key)
	}
	logx.Infof("[调试] allBusinessData keys: %v, 共%d个字段", businessDataKeys, len(allBusinessData))
	logx.Infof("[调试] visibleFields 数量: %d", len(visibleFields))

	// 过滤业务数据：只保留配置的可见字段（排除系统字段）
	latestBusinessData := make(map[string]interface{})
	for _, field := range visibleFields {
		logx.Infof("[调试] 检查字段: %s, IsVisible=%v", field.FieldName, field.IsVisible)
		if field.IsVisible {
			if value, exists := allBusinessData[field.FieldName]; exists {
				latestBusinessData[field.FieldName] = value
				logx.Infof("[调试] 添加可见字段: %s", field.FieldName)
			} else {
				logx.Infof("[调试] 字段 %s 在 allBusinessData 中不存在", field.FieldName)
			}
		}
	}
	logx.Infof("[调试] 最终 latestBusinessData 字段数: %d", len(latestBusinessData))

	// 构建 FlowHistory（按 VertexID 合并节点）
	// 1. 按 VertexID 分组 assignments
	vertexMap := make(map[int][]*assignmentRow)
	for _, assignment := range assignments {
		vertexMap[assignment.VertexID] = append(vertexMap[assignment.VertexID], assignment)
	}

	// 2. 合并相同 VertexID 的节点，收集所有处理人
	flowHistory := make([]*core.FlowNode, 0, len(vertexMap))
	for vertexID, vertexAssignments := range vertexMap {
		// 收集该节点的所有用户（去重）
		userIDSet := make(map[string]bool)
		userIDList := []string{}
		userNameList := []string{}

		// 找最早创建时间和最晚更新时间
		var createdAt, updatedAt time.Time
		var vertexName, vertexAlias string

		for i, assignment := range vertexAssignments {
			// 收集用户信息（去重）
			if assignment.UserID.Valid && assignment.UserID.String != "" {
				userID := assignment.UserID.String
				if !userIDSet[userID] {
					userIDSet[userID] = true
					userIDList = append(userIDList, userID)
					userNameList = append(userNameList, userNames[userID])
				}
			}

			// 第一个assignment获取节点信息和初始时间
			if i == 0 {
				createdAt = assignment.CreatedAt
				updatedAt = assignment.UpdatedAt
				if assignment.VertexName.Valid {
					vertexName = assignment.VertexName.String
				}
				if assignment.VertexAlias.Valid {
					vertexAlias = assignment.VertexAlias.String
				}
			} else {
				// 更新时间范围
				if assignment.CreatedAt.Before(createdAt) {
					createdAt = assignment.CreatedAt
				}
				if assignment.UpdatedAt.After(updatedAt) {
					updatedAt = assignment.UpdatedAt
				}
			}
		}

		flowHistory = append(flowHistory, &core.FlowNode{
			VertexID:    vertexID,
			VertexName:  vertexName,
			VertexAlias: vertexAlias,
			UserIDs:     userIDList,
			UserNames:   userNameList,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}

	// 3. 按创建时间排序（保持时间线顺序）
	sort.Slice(flowHistory, func(i, j int) bool {
		return flowHistory[i].CreatedAt.Before(flowHistory[j].CreatedAt)
	})

	return &core.DetailResponse{
		JourneyID:          firstAssignment.JourneyID,
		CurrentStatus:      currentStatus,
		InitiatorUserID:    initiatorUserID,
		InitiatorUserName:  initiatorUserName,
		InitiatedAt:        firstAssignment.CreatedAt,
		LatestBusinessData: latestBusinessData,
		FlowHistory:        flowHistory,
	}
}
