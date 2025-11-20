// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 任务管理
// 说明：本文件定义的类型用于表示流程任务（Assignment）相关信息
package core

// Assignment 任务信息（领域模型）
// 说明：表示流程中的一个任务节点处理信息（从 Skylark API 返回）
type Assignment struct {
	ID         int64   // 任务ID
	AssigneeID string  // 处理人ID（本地用户ID）
	Status     string  // 任务状态（processing）
	Category   string  // 任务类型（proposed, processed, cc）
	VertexID   int64   // 节点ID
	JourneyID  int64   // 流程记录ID
	CreatedAt  string  // 创建时间（ISO 8601 格式）
	UpdatedAt  string  // 更新时间（ISO 8601 格式）
	FlowID     *int64  // 流程ID（可选，性能优化时自动补充）
	FlowTitle  *string // 流程名称（可选，性能优化时自动补充）
}

// Assignment 类别常量（slp_category）
const (
	AssignmentCategoryProposed  = "proposed"  // 我发起的任务
	AssignmentCategoryProcessed = "processed" // 由我处理的任务
	AssignmentCategoryCC        = "cc"        // 抄送我的任务
)

// Assignment 状态常量（slp_status）
// 说明：
//   - category='proposed' 的状态表示整个流程的状态（与 journey.go 中的 Status* 常量值相同）
//   - category='processed' 的状态表示单个节点的处理状态
//
// 注意：
//   - AssignmentStatusProcessing/Finished/Aborted 与 journey.go 中的 StatusProcessing/StatusFinished/StatusAborted 值相同
//   - 为了代码清晰度，分别定义但值相同，代表同一个概念（Journey 状态 = proposed assignment 状态）
const (
	// 流程状态（category='proposed' 时，与 Journey 状态相同）
	AssignmentStatusProcessing = "processing" // 流程进行中
	AssignmentStatusFinished   = "finished"   // 流程已完成
	AssignmentStatusAborted    = "aborted"    // 流程已终止

	// 节点状态（category='processed' 时）
	AssignmentStatusApproved     = "approved"      // 审批同意
	AssignmentStatusRefused      = "refused"       // 审批拒绝
	AssignmentStatusTransferred  = "transferred"   // 转交
	AssignmentStatusWithdrawn    = "withdrawn"     // 撤回
	AssignmentStatusResubmitted  = "resubmitted"   // 重新提交
	AssignmentStatusAutoApproved = "auto_approved" // 自动审批通过
)
