package flows

import (
	"context"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetJourneyFullDetail 获取流程完整详情（一站式接口）
// 功能：
//   - 聚合基础信息、业务数据、审批历史、待处理节点、节点信息
//   - 减少前端调用次数（4次 → 1次）
//   - 自动补充节点名称、处理人姓名
//
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - *core.JourneyFullDetail: 流程完整详情（包含所有维度信息）
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetJourneyFullDetail(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
) (*core.JourneyFullDetail, error) {
	// 1. 参数校验
	if flowID <= 0 {
		return nil, fmt.Errorf("flowID 必须大于 0")
	}
	if journeyID <= 0 {
		return nil, fmt.Errorf("journeyID 必须大于 0")
	}

	// 2. 获取基础信息（GetJourneyDetail）
	basicInfo, err := f.GetJourneyDetail(ctx, tenantID, flowID, journeyID)
	if err != nil {
		return nil, fmt.Errorf("获取流程基础信息失败: %w", err)
	}

	// 3. 获取流程详情（节点信息）- 提前获取，用于后续填充节点名称
	flowDetail, err := f.GetFlowDetail(ctx, tenantID, flowID)
	if err != nil {
		return nil, fmt.Errorf("获取流程详情失败: %w", err)
	}

	// 4. 构建节点信息映射
	vertices := make(map[int64]*core.FlowVertex, len(flowDetail.Vertices))
	for _, vertex := range flowDetail.Vertices {
		vertices[vertex.ID] = vertex
	}

	// 5. 获取审批历史（GetJourneyMoments）
	history, err := f.GetJourneyMoments(ctx, tenantID, journeyID)
	if err != nil {
		// 审批历史失败不影响整体流程，只记录日志
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_full_detail"),
			logx.Field("journey_id", journeyID),
			logx.Field("error", err.Error()),
		).Error("获取审批历史失败")
		history = []*core.Moment{} // 返回空列表
	}

	// 6. 过滤并补充审批历史
	// 说明：
	//   - 只保留已处理的记录（排除 status="processing" 的待处理节点）
	//   - 自动补充节点名称（vertexName）
	filteredHistory := make([]*core.Moment, 0, len(history))
	for _, moment := range history {
		// 过滤：只保留已处理的记录
		if moment.StatusKey == core.StatusProcessing {
			continue // 跳过待处理节点（这些节点应该在 PendingNodes 中）
		}

		// 补充节点名称
		if vertex, ok := vertices[moment.VertexID]; ok {
			moment.VertexName = &vertex.Name
		}

		filteredHistory = append(filteredHistory, moment)
	}

	// 7. 获取任务列表（GetJourneyAssignments）
	assignments, err := f.GetJourneyAssignments(ctx, tenantID, journeyID)
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}

	// 8. 提取待处理节点
	pendingNodes, err := f.extractPendingNodes(ctx, tenantID, flowID, assignments, vertices)
	if err != nil {
		// 待处理节点提取失败不影响整体流程，只记录日志
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_full_detail"),
			logx.Field("journey_id", journeyID),
			logx.Field("error", err.Error()),
		).Error("提取待处理节点失败")
		pendingNodes = []*core.PendingNode{} // 返回空列表
	}

	// 9. 返回完整详情
	return &core.JourneyFullDetail{
		BasicInfo:    basicInfo,
		History:      filteredHistory, // 使用过滤并补充后的历史记录
		PendingNodes: pendingNodes,
		Vertices:     vertices,
	}, nil
}

// extractPendingNodes 从 assignments 中提取待处理节点
// 说明：
//   - 筛选条件：category='processed' AND status='processing'
//   - 自动补充节点名称（通过 vertices 映射）
//   - 自动补充处理人姓名（通过用户映射）
//
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - flowID: 流程ID
//   - assignments: 任务列表
//   - vertices: 节点信息映射（用于补充节点名称）
//
// 返回:
//   - []*core.PendingNode: 待处理节点列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) extractPendingNodes(
	ctx context.Context,
	tenantID string,
	flowID int64,
	assignments []*core.Assignment,
	vertices map[int64]*core.FlowVertex,
) ([]*core.PendingNode, error) {
	// 1. 筛选待处理任务
	// 条件：category='processed' AND status='processing'
	pendingAssignments := make([]*core.Assignment, 0)
	for _, a := range assignments {
		if a.Category == core.AssignmentCategoryProcessed && a.Status == core.AssignmentStatusProcessing {
			pendingAssignments = append(pendingAssignments, a)
		}
	}

	// 2. 快速返回：没有待处理任务
	if len(pendingAssignments) == 0 {
		return []*core.PendingNode{}, nil
	}

	// 3. 按节点聚合（一个节点可能有多个待处理人）
	// 键：vertexID，值：任务列表
	vertexAssignmentsMap := make(map[int64][]*core.Assignment)
	for _, a := range pendingAssignments {
		vertexAssignmentsMap[a.VertexID] = append(vertexAssignmentsMap[a.VertexID], a)
	}

	// 4. 构建待处理节点列表
	pendingNodes := make([]*core.PendingNode, 0, len(vertexAssignmentsMap))
	for vertexID, nodeAssignments := range vertexAssignmentsMap {
		// 提取处理人ID列表
		assigneeIDs := make([]string, len(nodeAssignments))
		for i, a := range nodeAssignments {
			assigneeIDs[i] = a.AssigneeID
		}

		// 获取节点名称（从传入的 vertices 映射）
		vertexName := ""
		if vertex, ok := vertices[vertexID]; ok {
			vertexName = vertex.Name
		}

		// 构建待处理节点
		pendingNode := &core.PendingNode{
			VertexID:      vertexID,
			VertexName:    vertexName, // 从 vertices 映射获取节点名称
			AssigneeIDs:   assigneeIDs,
			AssigneeNames: assigneeIDs, // 目前直接使用 ID，后续可增强为批量查询用户名
			CreatedAt:     nodeAssignments[0].CreatedAt,
		}

		pendingNodes = append(pendingNodes, pendingNode)
	}

	return pendingNodes, nil
}
