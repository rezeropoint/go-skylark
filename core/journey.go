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
	ID    string  // 用户ID（本地用户ID）
	Name  string  // 用户名称
	Email *string // 用户邮箱（可为空）
}

// JourneyDetail 流程记录详情（领域模型）
// 说明：包含流程记录的完整信息，包括基础数据、字段值、附件等
// 用于 GetJourneyDetail 接口返回完整的流程记录数据
// 设计参考 DetailResponse，但基于 Skylark API 返回的数据结构
type JourneyDetail struct {
	// 基础信息
	ID              int64  // 流程记录ID
	SN              string // 流程编号
	Status          string // 流程状态（processing, completed, aborted, stashed）
	CurrentVertexID int64  // 当前节点ID
	FlowID          int64  // 流程ID
	CreatedAt       string // 创建时间（ISO 8601 格式）
	UpdatedAt       string // 更新时间（ISO 8601 格式）
	JourneyURL      string // 流程记录URL

	// 审核相关
	ReviewerVertexIDs        []int64 // 审核节点ID列表
	CurrentDurationThreshold *string // 当前持续时间阈值

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
	StatusStepIn      = "step_in"     // 介入
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
	StatusProposed:    "已发起",
	StatusStepIn:      "已介入",
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

// JourneySearchRequest 流程记录搜索请求（领域模型）
// 说明：用于搜索流程记录，支持多种过滤条件
// 用途：调用 POST /api/v4/yaw/flows/:id/journeys/search 接口
// 示例：
//
//	req := &core.JourneySearchRequest{
//	    FlowID:   123,
//	    Status:   core.StatusProcessing, // 使用常量：StatusProcessing, StatusFinished, StatusAborted, StatusStashed
//	    Page:     1,
//	    PageSize: 20,
//	}
type JourneySearchRequest struct {
	FlowID int64 // 流程ID（必填）

	// 搜索条件（可选，空字符串表示不限制）
	Status      string // 流程状态（使用常量：StatusProcessing, StatusFinished, StatusAborted, StatusStashed；空字符串表示不筛选）
	Keyword     string // 关键词搜索（搜索SN或字段值；空字符串表示不搜索）
	InitiatorID string // 发起人ID（本地用户ID；空字符串表示不限制发起人）

	// 时间范围（可选，空字符串表示不限制）
	CreatedFrom string // 创建时间开始（ISO 8601格式，如 "2025-01-01T00:00:00Z"；空字符串表示不限制开始时间）
	CreatedTo   string // 创建时间结束（ISO 8601格式；空字符串表示不限制结束时间）

	// 分页参数
	Page     int // 页码（从1开始）
	PageSize int // 每页数量
}

// JourneySearchResponse 流程记录搜索响应（领域模型）
// 说明：搜索接口返回的结果，包含流程列表和总数
type JourneySearchResponse struct {
	Journeys   []*Journey // 流程记录列表
	TotalCount int        // 总数
}

// Moment 流程审批历史记录（领域模型）
// 说明：表示流程中的一次节点处理记录（审批、回退、转交等操作）
// 用途：GetJourneyMoments 接口返回流程的审批时间线
// 注意：与 Assignment 的区别：
//   - Assignment 表示任务（可能有多个待处理人）
//   - Moment 表示历史记录（某个人在某个时间点执行的操作）
type Moment struct {
	ID           int64   // 记录ID
	AssignmentID int64   // 任务ID
	JourneyID    int64   // 流程记录ID
	VertexID     int64   // 节点ID
	VertexName   *string // 节点名称（可为空）
	Status       string  // 操作状态中文显示名称（如 "处理中"）
	StatusKey    string  // 操作状态英文键值（如 "processing"，对应 StatusProcessing/StatusApproved 等常量）
	OperatorID   string  // 操作人ID（本地用户ID）
	OperatorName *string // 操作人姓名（可为空）
	Comment      *string // 处理意见（可为空）
	CreatedAt    string  // 创建时间（ISO 8601格式）
	UpdatedAt    string  // 更新时间（ISO 8601格式）
	Duration     *int    // 处理时长（秒，可为空）
}

// ProcessingUser 当前处理人信息（领域模型）
// 说明：表示流程当前待处理人的完整信息（从 Skylark API 返回）
// 用途：GetCurrentProcessingUsers 接口返回当前流程任务的处理者列表
// 注意：部分字段可能为空（nickname、phone、identifier、headimgurl）
type ProcessingUser struct {
	ID         string   // 用户ID（本地用户ID）
	Name       string   // 用户名称
	Nickname   *string  // 昵称（可为空）
	Phone      *string  // 手机号（可为空）
	Identifier *string  // 标识符（可为空）
	Headimgurl *string  // 头像URL（可为空）
	Tags       []string // 标签列表
}

// UpdateJourneyStatusOptions 更新流程状态选项（领域模型）
// 说明：用于更新流程任务状态时提供的可选参数
// 用途：UpdateJourneyStatus 接口的参数，支持审批意见、下一节点、抄送等
type UpdateJourneyStatusOptions struct {
	Comment           string                // 处理意见
	NextVertexID      int                   // 下一个节点ID
	CarbonCopyUserIDs []string              // 抄送者ID列表（本地用户ID）
	Data              map[string]TypedValue // 字段数据
}
