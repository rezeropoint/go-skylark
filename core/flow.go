// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：Flow 信息查询、字段元数据）
package core

import "fmt"

// FlowInfo 远程流程信息（纯领域模型）
type FlowInfo struct {
	ID          int    // 流程ID
	Title       string // 流程名称
	NamespaceID int    // 命名空间ID
}

// FieldMetadata 远程表字段元数据（纯领域模型）
type FieldMetadata struct {
	FieldName string // 字段名
	DataType  string // 数据类型（PostgreSQL类型）
	IsSystem  bool   // 是否为系统字段（slp_前缀），通过IsSystemField()计算
}

// IsSystemField 判断字段是否为系统字段
// 系统字段以 slp_ 前缀标识
func IsSystemField(fieldName string) bool {
	return len(fieldName) >= 4 && fieldName[:4] == "slp_"
}

// API 请求常量（调用 Skylark REST API 使用）

// HTTP 协议常量
const (
	SchemeHTTPS = "https://" // HTTPS协议
)

// API 路径常量
const (
	APIFlowsPath       = "/api/v4/yaw/flows/"          // 流程API路径
	APIFormsPath       = "/api/v4/forms/"              // 表单API路径
	APIJourneysPath    = "/api/v4/yaw/journeys/"       // 流程记录API路径
	APIAttachmentsPath = "/api/v4/attachments/uptoken" // 附件API路径
	APIUploadPath      = "https://up.qbox.me/"         // 七牛云上传API路径
)

// JourneyOperation 流程操作类型
type JourneyOperation string

// 操作类型常量
const (
	OperationRoute    JourneyOperation = "route"    // 路由操作（修改数据）
	OperationPropose  JourneyOperation = "propose"  // 提议操作（创建流程）
	OperationApprove  JourneyOperation = "approve"  // 通过操作
	OperationRefuse   JourneyOperation = "refuse"   // 回退操作
	OperationTransfer JourneyOperation = "transfer" // 转交操作
	OperationCancel   JourneyOperation = "cancel"   // 撤销操作
)

// 事件类型常量
const (
	EventJourneyStatus = "JourneyStatusEvent" // 旅程状态事件
)

// 节点类型常量
const (
	VertexTypeInitial = "Initial" // 初始节点（流程开始）
	VertexTypeNormal  = "Normal"  // 普通节点（审批节点）
	VertexTypeFinal   = "Final"   // 最终节点（流程结束）
)

// 七牛云上传常量
const (
	QiniuXKeyValue        = "1593586993541" // 七牛云上传x:key字段的值
	DefaultUserIDForToken = "1"             // 获取上传令牌时使用的默认用户ID
)

// BuildJourneyAPIURL 构建 Journey 相关 API URL
// 格式：https://{app}/api/v4/yaw/journeys/{journey_id}/{action}
// 参数：
//   - apiCtx: API 调用上下文
//   - journeyID: 流程记录ID
//   - action: 操作路径（如 "assignments"）
// 返回：完整的 API URL
func BuildJourneyAPIURL(apiCtx SkylarkAPIContext, journeyID int64, action string) string {
	return fmt.Sprintf("%s%s%s%d/%s", SchemeHTTPS, apiCtx.App, APIJourneysPath, journeyID, action)
}

// FlowDetail 流程详情（领域模型）
// 说明：包含流程的完整结构信息，包括字段、节点、边
// 用于 GetFlowDetail 接口返回流程的元数据
type FlowDetail struct {
	ID       int64         // 流程ID
	Title    string        // 流程名称
	Fields   []*FlowField  // 字段列表
	Vertices []*FlowVertex // 节点列表
	Edges    []*FlowEdge   // 边列表
}

// FlowField 流程字段（领域模型）
// 说明：表示流程中的一个字段定义
type FlowField struct {
	ID    int64  // 字段ID
	Title string // 字段标题
}

// FlowVertex 流程节点（领域模型）
// 说明：表示流程中的一个节点（审批步骤）
type FlowVertex struct {
	ID   int64  // 节点ID
	Name string // 节点名称
	Type string // 节点类型（Initial/Normal/Final）
}

// FlowEdge 流程边（领域模型）
// 说明：表示流程节点之间的连接关系
type FlowEdge struct {
	FromVertexID int64 // 起始节点ID
	ToVertexID   int64 // 目标节点ID
}
