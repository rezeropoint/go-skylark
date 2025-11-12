// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 任务管理
// 说明：本文件定义的类型用于表示流程任务（Assignment）相关信息
package core

// Assignment 任务信息（领域模型）
// 说明：表示流程中的一个任务节点处理信息（从 Skylark API 返回）
type Assignment struct {
	ID         int64  // 任务ID
	AssigneeID int64  // 处理人ID
	Status     string // 任务状态（processing, completed）
	Category   string // 任务类型（proposed, processed, cc）
	VertexID   int64  // 节点ID
	JourneyID  int64  // 流程记录ID
	CreatedAt  string // 创建时间（ISO 8601 格式）
	UpdatedAt  string // 更新时间（ISO 8601 格式）
}
