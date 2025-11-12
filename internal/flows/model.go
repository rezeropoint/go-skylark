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

// JourneyDetailResponse Skylark API 返回的流程记录详情结构体
// 职责：处理 GetJourneyDetail API 响应的 JSON 反序列化
// 说明：继承 JourneyResponse 基础字段，扩展 Response 为 ResponseDetailData
type JourneyDetailResponse struct {
	ID                       int64              `json:"id"`                          // 流程记录ID
	SN                       string             `json:"sn"`                          // 流程编号
	Status                   string             `json:"status"`                      // 流程状态
	CurrentVertexID          int64              `json:"current_vertex_id"`           // 当前节点ID
	FlowID                   int64              `json:"flow_id"`                     // 流程ID
	CurrentDurationThreshold *string            `json:"current_duration_threshold"`  // 当前持续时间阈值
	CreatedAt                string             `json:"created_at"`                  // 创建时间
	UpdatedAt                string             `json:"updated_at"`                  // 更新时间
	ReviewerVertexIDs        []int64            `json:"reviewer_vertex_ids"`         // 审核节点ID列表
	JourneyURL               string             `json:"journey_url"`                 // 流程记录URL
	User                     UserInfo           `json:"user"`                        // 发起人信息
	Response                 ResponseDetailData `json:"response"`                    // 响应数据（详情版）
}

// ResponseDetailData API 返回的响应详情数据结构体
// 说明：包含完整的字段数据和附件信息（用于详情接口）
type ResponseDetailData struct {
	ID           int64                        `json:"id"`            // 响应ID
	CachedValues map[string]FieldValueDetail  `json:"cached_values"` // 字段ID -> 值详情
	MappedValues map[string]FieldValueDetail  `json:"mapped_values"` // 字段别名 -> 值详情
	Entries      []EntryDetail                `json:"entries"`       // 字段条目列表
}

// FieldValueDetail 字段值详情结构体
// 说明：包含字段的多种表示形式
type FieldValueDetail struct {
	Value         []interface{} `json:"value"`          // 原始值
	TextValue     []string      `json:"text_value"`     // 文本值
	ExportedValue []string      `json:"exported_value"` // 导出值
}

// EntryDetail 字段条目详情结构体
// 说明：表示一个字段的具体条目（可能包含附件）
type EntryDetail struct {
	ID         int64             `json:"id"`                   // 条目ID
	FieldID    int64             `json:"field_id"`             // 字段ID
	OptionID   *int64            `json:"option_id"`            // 选项ID（可为空）
	Value      string            `json:"value"`                // 值
	ValueID    *int64            `json:"value_id"`             // 值ID（可为空）
	Attachment *AttachmentDetail `json:"attachment,omitempty"` // 附件（可为空）
}

// AttachmentDetail 附件详情结构体
// 说明：包含附件的完整信息
type AttachmentDetail struct {
	ID          int64  `json:"id"`           // 附件ID
	Name        string `json:"name"`         // 文件名
	Size        string `json:"size"`         // 文件大小
	MimeType    string `json:"mime_type"`    // MIME 类型
	Extension   string `json:"extension"`    // 文件扩展名
	DownloadURL string `json:"download_url"` // 下载地址
}

// ToDomain 将 JourneyDetailResponse 转换为领域模型
func (j *JourneyDetailResponse) ToDomain() *core.JourneyDetail {
	// 构建业务数据（优先级：ExportedValue > TextValue > Value）
	businessData := make(map[string]interface{})
	for fieldID, fieldValue := range j.Response.CachedValues {
		if len(fieldValue.ExportedValue) > 0 {
			if len(fieldValue.ExportedValue) == 1 {
				businessData[fieldID] = fieldValue.ExportedValue[0]
			} else {
				businessData[fieldID] = fieldValue.ExportedValue
			}
		} else if len(fieldValue.TextValue) > 0 {
			if len(fieldValue.TextValue) == 1 {
				businessData[fieldID] = fieldValue.TextValue[0]
			} else {
				businessData[fieldID] = fieldValue.TextValue
			}
		} else if len(fieldValue.Value) > 0 {
			if len(fieldValue.Value) == 1 {
				businessData[fieldID] = fieldValue.Value[0]
			} else {
				businessData[fieldID] = fieldValue.Value
			}
		}
	}

	// 提取附件列表
	var attachments []*core.Attachment
	for _, entry := range j.Response.Entries {
		if entry.Attachment != nil {
			attachments = append(attachments, &core.Attachment{
				ID:          entry.Attachment.ID,
				Name:        entry.Attachment.Name,
				Size:        entry.Attachment.Size,
				MimeType:    entry.Attachment.MimeType,
				Extension:   entry.Attachment.Extension,
				DownloadURL: entry.Attachment.DownloadURL,
			})
		}
	}

	return &core.JourneyDetail{
		// 基础信息
		ID:              j.ID,
		SN:              j.SN,
		Status:          j.Status,
		CurrentVertexID: j.CurrentVertexID,
		FlowID:          j.FlowID,
		CreatedAt:       j.CreatedAt,
		UpdatedAt:       j.UpdatedAt,
		JourneyURL:      j.JourneyURL,

		// 审核相关
		ReviewerVertexIDs:        j.ReviewerVertexIDs,
		CurrentDurationThreshold: j.CurrentDurationThreshold,

		// 发起人信息
		Initiator: j.User.ToDomain(),

		// 业务数据
		BusinessData: businessData,

		// 附件信息
		Attachments: attachments,
	}
}

// FlowDetailResponse Skylark API 返回的流程详情结构体
// 职责：处理 GetFlowDetail API 响应的 JSON 反序列化
type FlowDetailResponse struct {
	ID       int64                `json:"id"`       // 流程ID
	Title    string               `json:"title"`    // 流程名称
	Fields   []FlowFieldResponse  `json:"fields"`   // 字段列表
	Vertices []FlowVertexResponse `json:"vertices"` // 节点列表
	Edges    []FlowEdgeResponse   `json:"edges"`    // 边列表
}

// FlowFieldResponse 流程字段响应结构体
type FlowFieldResponse struct {
	ID          int64   `json:"id"`          // 字段ID
	Title       string  `json:"title"`       // 字段标题
	Description *string `json:"description"` // 字段描述（可为空）
}

// FlowVertexResponse 流程节点响应结构体
type FlowVertexResponse struct {
	ID   int64  `json:"id"`   // 节点ID
	Name string `json:"name"` // 节点名称
	Type string `json:"type"` // 节点类型（Initial/Normal/Final）
}

// FlowEdgeResponse 流程边响应结构体
type FlowEdgeResponse struct {
	ID           int64 `json:"id"`             // 边ID
	FromVertexID int64 `json:"from_vertex_id"` // 起始节点ID
	ToVertexID   int64 `json:"to_vertex_id"`   // 目标节点ID
}

// ToDomain 将 FlowDetailResponse 转换为领域模型
func (f *FlowDetailResponse) ToDomain() *core.FlowDetail {
	// 转换字段列表
	fields := make([]*core.FlowField, len(f.Fields))
	for i, field := range f.Fields {
		fields[i] = field.ToDomain()
	}

	// 转换节点列表
	vertices := make([]*core.FlowVertex, len(f.Vertices))
	for i, vertex := range f.Vertices {
		vertices[i] = vertex.ToDomain()
	}

	// 转换边列表
	edges := make([]*core.FlowEdge, len(f.Edges))
	for i, edge := range f.Edges {
		edges[i] = edge.ToDomain()
	}

	return &core.FlowDetail{
		ID:       f.ID,
		Title:    f.Title,
		Fields:   fields,
		Vertices: vertices,
		Edges:    edges,
	}
}

// ToDomain 将 FlowFieldResponse 转换为领域模型
func (f *FlowFieldResponse) ToDomain() *core.FlowField {
	return &core.FlowField{
		ID:    f.ID,
		Title: f.Title,
	}
}

// ToDomain 将 FlowVertexResponse 转换为领域模型
func (v *FlowVertexResponse) ToDomain() *core.FlowVertex {
	return &core.FlowVertex{
		ID:   v.ID,
		Name: v.Name,
		Type: v.Type,
	}
}

// ToDomain 将 FlowEdgeResponse 转换为领域模型
func (e *FlowEdgeResponse) ToDomain() *core.FlowEdge {
	return &core.FlowEdge{
		FromVertexID: e.FromVertexID,
		ToVertexID:   e.ToVertexID,
	}
}

// UserAssignmentsResponse Skylark API 返回的用户任务列表结构体
// 职责：处理 GET /api/v4/yaw/flows/user_assignments.json 响应的 JSON 反序列化
type UserAssignmentsResponse struct {
	Assignments []AssignmentResponse `json:"assignments"` // 任务列表（复用已有类型）
}

// ProposedJourneysResponse Skylark API 返回的用户发起的流程列表结构体
// 职责：处理 GET /api/v4/yaw/flows/proposed_journeys.json 响应的 JSON 反序列化
type ProposedJourneysResponse struct {
	Journeys []JourneyResponse `json:"journeys"` // 流程列表（复用已有类型）
}
