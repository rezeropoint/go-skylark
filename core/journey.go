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

// JourneyDetail 流程记录详情（领域模型）
// 说明：包含流程记录的完整信息，包括基础数据、字段值、附件等
// 用于 GetJourneyDetail 接口返回完整的流程记录数据
// 设计参考 DetailResponse，但基于 Skylark API 返回的数据结构
type JourneyDetail struct {
	// 基础信息
	ID              int64     // 流程记录ID
	SN              string    // 流程编号
	Status          string    // 流程状态（processing, completed, aborted, stashed）
	CurrentVertexID int64     // 当前节点ID
	FlowID          int64     // 流程ID
	CreatedAt       string    // 创建时间（ISO 8601 格式）
	UpdatedAt       string    // 更新时间（ISO 8601 格式）
	JourneyURL      string    // 流程记录URL

	// 审核相关
	ReviewerVertexIDs        []int64   // 审核节点ID列表
	CurrentDurationThreshold *string   // 当前持续时间阈值

	// 发起人信息
	Initiator *FlowUser // 发起人信息

	// 业务数据
	BusinessData map[string]interface{} // 业务字段数据（field_id -> value）

	// 附件信息
	Attachments []*Attachment // 附件列表
}

// Attachment 附件信息（领域模型）
// 说明：表示流程中的附件文件信息
type Attachment struct {
	ID          int64  // 附件ID
	Name        string // 文件名
	Size        string // 文件大小（如 "1.2 MB"）
	MimeType    string // MIME 类型（如 "image/png"）
	Extension   string // 文件扩展名（如 "png"）
	DownloadURL string // 下载地址
}

// 流程状态常量
// 说明：
// - Journey.Status 使用这些常量表示流程整体状态
// - Assignment.Status 使用 StatusProcessing/StatusCompleted 表示任务处理状态
// - Assignment.moments[].status 使用操作状态表示每次变化的具体操作
// - Assignment 是 Journey 的变化记录，它们共用同一套状态体系
const (
	// Journey 整体状态（API 返回的 journey.status）
	StatusStashed    = "stashed"    // 草稿（未提交）
	StatusProcessing = "processing" // 进行中
	StatusFinished   = "finished"   // 已完成
	StatusAborted    = "aborted"    // 已终止

	// Assignment 任务处理状态（API 返回的 assignment.status）
	StatusCompleted = "completed" // 任务已完成

	// Assignment 操作状态（moments[].status，表示每次变化的具体操作）
	StatusProposed    = "proposed"    // 提议（发起）
	StatusApproved    = "approved"    // 已通过
	StatusRefused     = "refused"     // 已回退
	StatusTransferred = "transferred" // 已转交
	StatusSkipped     = "skipped"     // 已跳过
	StatusCancelled   = "cancelled"   // 已撤销
	StatusReceding    = "receding"    // 回退中
	StatusSuspended   = "suspended"   // 已暂停

	// 虚拟状态（由 SDK 计算，不是 API 返回）
	StatusPending = "pending" // 待处理（只有1个节点的未完成事件）
)

// StatusTranslationMap 流程状态中文翻译映射
// 说明：统一的状态翻译表，适用于 Journey 和 Assignment 的所有状态
var StatusTranslationMap = map[string]string{
	StatusStashed:     "编写中",
	StatusPending:     "待处理",
	StatusProcessing:  "处理中",
	StatusFinished:    "已完成",
	StatusAborted:     "已终止",
	StatusCompleted:   "已完成",
	StatusApproved:    "已通过",
	StatusRefused:     "已回退",
	StatusTransferred: "已转交",
	StatusSkipped:     "已跳过",
	StatusCancelled:   "已撤销",
	StatusReceding:    "回退中",
	StatusSuspended:   "已暂停",
}

// TranslateStatus 将流程状态翻译为中文，如果未知则返回原始状态
// 适用于 Journey.Status 和 Assignment 的 moments[].status
func TranslateStatus(status string) string {
	if translated, ok := StatusTranslationMap[status]; ok {
		return translated
	}
	return status
}
