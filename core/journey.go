// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 流程记录管理
// 说明：本文件定义的类型用于表示流程记录（Journey）及相关信息
package core

// Journey 流程记录（领域模型）
// 说明：表示一个流程实例的基本信息（从 Skylark API 返回）
type Journey struct {
	ID              int64     // 流程记录ID
	SN              string    // 流程编号
	Status          string    // 流程状态（processing, completed, aborted, stashed）
	CurrentVertexID int64     // 当前节点ID
	FlowID          int64     // 流程ID
	CreatedAt       string    // 创建时间
	UpdatedAt       string    // 更新时间
	User            *FlowUser // 发起人信息（可为空）
}

// FlowUser 流程用户信息（领域模型）
// 说明：用于流程API返回的用户信息，包含完整的用户数据
// 注意：不同于 core.User（用户映射专用），此结构体用于API响应
type FlowUser struct {
	ID    int64   // 用户ID
	Name  string  // 用户名称
	Email *string // 用户邮箱（可为空）
}
