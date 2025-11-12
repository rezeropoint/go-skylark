package flows

import (
	"github.com/rezeropoint/go-skylark/core"
)

// FlowInfoModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射
type FlowInfoModel struct {
	ID          int    `db:"id"`           // 流程ID
	Title       string `db:"title"`        // 流程名称
	NamespaceID int    `db:"namespace_id"` // 命名空间ID
}

// FieldMetadataModel 是数据库查询专用结构体（基础设施层）
type FieldMetadataModel struct {
	FieldName string `db:"column_name"` // 字段名
	DataType  string `db:"data_type"`   // 数据类型（PostgreSQL类型）
}

// ToDomain 将数据库模型转换为领域模型
func (m *FlowInfoModel) ToDomain() *core.FlowInfo {
	return &core.FlowInfo{
		ID:          m.ID,
		Title:       m.Title,
		NamespaceID: m.NamespaceID,
	}
}

// ToDomain 将数据库模型转换为领域模型
func (m *FieldMetadataModel) ToDomain() *core.FieldMetadata {
	return &core.FieldMetadata{
		FieldName: m.FieldName,
		DataType:  m.DataType,
		IsSystem:  core.IsSystemField(m.FieldName),
	}
}

// JourneyResponse Skylark API 返回的流程记录结构体
// 职责：处理 API 响应的 JSON 反序列化
type JourneyResponse struct {
	ID                       int64    `json:"id"`                          // 流程记录ID
	SN                       string   `json:"sn"`                          // 流程编号
	Status                   string   `json:"status"`                      // 流程状态
	CurrentVertexID          int64    `json:"current_vertex_id"`           // 当前节点ID
	FlowID                   int64    `json:"flow_id"`                     // 流程ID
	CurrentDurationThreshold *string  `json:"current_duration_threshold"`  // 当前持续时间阈值
	CreatedAt                string   `json:"created_at"`                  // 创建时间
	UpdatedAt                string   `json:"updated_at"`                  // 更新时间
	ReviewerVertexIDs        []int64  `json:"reviewer_vertex_ids"`         // 审核节点ID列表
	JourneyURL               string   `json:"journey_url"`                 // 流程记录URL
	User                     UserInfo `json:"user"`                        // 发起人信息
	Response                 ResponseData `json:"response"`                // 响应数据
}

// UserInfo API 返回的用户信息结构体
type UserInfo struct {
	ID    int64   `json:"id"`    // 用户ID
	Name  string  `json:"name"`  // 用户名称
	Email *string `json:"email"` // 用户邮箱（可为空）
}

// ResponseData API 返回的响应数据结构体
// 说明：包含流程的字段数据和附件信息（初版简化，后续扩展）
type ResponseData struct {
	ID int64 `json:"id"` // 响应ID
	// 其他字段按需扩展（cached_values, mapped_values, entries 等）
}

// ToDomain 将 API 响应转换为领域模型
func (j *JourneyResponse) ToDomain() *core.Journey {
	return &core.Journey{
		ID:              j.ID,
		SN:              j.SN,
		Status:          j.Status,
		CurrentVertexID: j.CurrentVertexID,
		FlowID:          j.FlowID,
		CreatedAt:       j.CreatedAt,
		UpdatedAt:       j.UpdatedAt,
		User:            j.User.ToDomain(),
	}
}

// ToDomain 将 API 用户信息转换为领域模型
func (u *UserInfo) ToDomain() *core.FlowUser {
	return &core.FlowUser{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}
}

// AssignmentResponse Skylark API 返回的任务结构体
// 职责：处理 API 响应的 JSON 反序列化
type AssignmentResponse struct {
	ID                       int64                  `json:"id"`                          // 任务ID
	AssigneeID               int64                  `json:"assignee_id"`                 // 处理人ID
	Status                   string                 `json:"status"`                      // 任务状态（processing, completed）
	Category                 string                 `json:"category"`                    // 任务类型（proposed, processed, cc）
	Read                     bool                   `json:"read"`                        // 是否已读
	CurrentDurationThreshold *string                `json:"current_duration_threshold"`  // 当前持续时间阈值
	VertexID                 int64                  `json:"vertex_id"`                   // 节点ID
	JourneyID                int64                  `json:"journey_id"`                  // 流程记录ID
	CreatedAt                string                 `json:"created_at"`                  // 创建时间
	UpdatedAt                string                 `json:"updated_at"`                  // 更新时间
	OperationData            map[string]interface{} `json:"operation_data"`              // 操作数据
	Response                 ResponseData           `json:"response"`                    // 响应数据
}

// ToDomain 将 API 响应转换为领域模型
func (a *AssignmentResponse) ToDomain() *core.Assignment {
	return &core.Assignment{
		ID:         a.ID,
		AssigneeID: a.AssigneeID,
		Status:     a.Status,
		Category:   a.Category,
		VertexID:   a.VertexID,
		JourneyID:  a.JourneyID,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}
