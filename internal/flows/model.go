package flows

import (
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// SearchJourneysRequest 搜索流程记录请求结构体
// 说明：用于 POST /api/v4/yaw/flows/:id/journeys/search 请求体
type SearchJourneysRequest struct {
	Page        int    `json:"page"`                   // 页码
	PerPage     int    `json:"per_page"`               // 每页数量
	Status      string `json:"status,omitempty"`       // 状态过滤（可选）
	Keyword     string `json:"keyword,omitempty"`      // 关键词搜索（可选）
	InitiatorID *int64 `json:"initiator_id,omitempty"` // 发起人ID（可选）
	CreatedFrom string `json:"created_from,omitempty"` // 创建时间起始（可选）
	CreatedTo   string `json:"created_to,omitempty"`   // 创建时间结束（可选）
	Token       string `header:"Authorization"`        // 认证令牌
}

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
	ID                       int64        `json:"id"`                         // 流程记录ID
	SN                       string       `json:"sn"`                         // 流程编号
	Status                   string       `json:"status"`                     // 流程状态
	CurrentVertexID          int64        `json:"current_vertex_id"`          // 当前节点ID
	FlowID                   int64        `json:"flow_id"`                    // 流程ID
	CurrentDurationThreshold *string      `json:"current_duration_threshold"` // 当前持续时间阈值
	CreatedAt                string       `json:"created_at"`                 // 创建时间
	UpdatedAt                string       `json:"updated_at"`                 // 更新时间
	ReviewerVertexIDs        []int64      `json:"reviewer_vertex_ids"`        // 审核节点ID列表
	JourneyURL               string       `json:"journey_url"`                // 流程记录URL
	User                     UserInfo     `json:"user"`                       // 发起人信息
	Response                 ResponseData `json:"response"`                   // 响应数据
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
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
func (j *JourneyResponse) ToDomain(userIDMapping map[int]string) *core.Journey {
	return &core.Journey{
		ID:              j.ID,
		SN:              j.SN,
		Status:          j.Status,
		CurrentVertexID: j.CurrentVertexID,
		FlowID:          j.FlowID,
		CreatedAt:       j.CreatedAt,
		UpdatedAt:       j.UpdatedAt,
		User:            j.User.ToDomain(userIDMapping),
	}
}

// ToDomain 将 API 用户信息转换为领域模型
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
//
// 说明：
//   - 如果映射中找不到对应的本地用户ID，返回空字符串
//   - 一般情况下不会找不到，因为调用前已经通过 fillLocalUserIDMap 验证
func (u *UserInfo) ToDomain(userIDMapping map[int]string) *core.FlowUser {

	return &core.FlowUser{
		ID:    userIDMapping[int(u.ID)], // 直接使用本地用户ID（找不到为空字符串）
		Name:  u.Name,
		Email: u.Email,
	}
}

// AssignmentResponse Skylark API 返回的任务结构体
// 职责：处理 API 响应的 JSON 反序列化
type AssignmentResponse struct {
	ID                       int64                  `json:"id"`                         // 任务ID
	AssigneeID               int64                  `json:"assignee_id"`                // 处理人ID
	Status                   string                 `json:"status"`                     // 任务状态（processing, finished, aborted, approved 等）
	Category                 string                 `json:"category"`                   // 任务类型（proposed, processed, cc）
	Read                     bool                   `json:"read"`                       // 是否已读
	CurrentDurationThreshold *string                `json:"current_duration_threshold"` // 当前持续时间阈值
	VertexID                 int64                  `json:"vertex_id"`                  // 节点ID
	JourneyID                int64                  `json:"journey_id"`                 // 流程记录ID
	CreatedAt                string                 `json:"created_at"`                 // 创建时间
	UpdatedAt                string                 `json:"updated_at"`                 // 更新时间
	OperationData            map[string]interface{} `json:"operation_data"`             // 操作数据
	Response                 ResponseData           `json:"response"`                   // 响应数据
}

// ToDomain 将 API 响应转换为领域模型
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
func (a *AssignmentResponse) ToDomain(userIDMapping map[int]string) *core.Assignment {
	return &core.Assignment{
		ID:         a.ID,
		AssigneeID: userIDMapping[int(a.AssigneeID)], // 直接使用本地用户ID（找不到为空字符串）
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
	ID                       int64              `json:"id"`                         // 流程记录ID
	SN                       string             `json:"sn"`                         // 流程编号
	Status                   string             `json:"status"`                     // 流程状态
	CurrentVertexID          int64              `json:"current_vertex_id"`          // 当前节点ID
	FlowID                   int64              `json:"flow_id"`                    // 流程ID
	CurrentDurationThreshold *string            `json:"current_duration_threshold"` // 当前持续时间阈值
	CreatedAt                string             `json:"created_at"`                 // 创建时间
	UpdatedAt                string             `json:"updated_at"`                 // 更新时间
	ReviewerVertexIDs        []int64            `json:"reviewer_vertex_ids"`        // 审核节点ID列表
	JourneyURL               string             `json:"journey_url"`                // 流程记录URL
	User                     UserInfo           `json:"user"`                       // 发起人信息
	Response                 ResponseDetailData `json:"response"`                   // 响应数据（详情版）
}

// ResponseDetailData API 返回的响应详情数据结构体
// 说明：包含完整的字段数据和附件信息（用于详情接口）
type ResponseDetailData struct {
	ID           int64                       `json:"id"`            // 响应ID
	CachedValues map[string]FieldValueDetail `json:"cached_values"` // 字段ID -> 值详情
	MappedValues map[string]FieldValueDetail `json:"mapped_values"` // 字段别名 -> 值详情
	Entries      []EntryDetail               `json:"entries"`       // 字段条目列表
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
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
func (j *JourneyDetailResponse) ToDomain(userIDMapping map[int]string) *core.JourneyDetail {
	return j.ToDomainWithFieldNames(userIDMapping, nil)
}

// ToDomainWithFieldNames 将 API 响应转换为领域模型（使用字段名作为键）
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
//   - fieldMappings: 字段映射（字段名 → 字段信息），用于将字段ID转换为字段名
func (j *JourneyDetailResponse) ToDomainWithFieldNames(userIDMapping map[int]string, fieldMappings map[string]core.FieldMapping) *core.JourneyDetail {
	// 构建字段ID到字段名的反向映射
	fieldIDToName := make(map[string]string)
	if fieldMappings != nil {
		for fieldName, fieldInfo := range fieldMappings {
			fieldIDToName[fmt.Sprintf("%d", fieldInfo.ID)] = fieldName
		}
	}

	// 构建业务数据（优先级：ExportedValue > TextValue > Value）
	businessData := make(map[string]interface{})
	for fieldID, fieldValue := range j.Response.CachedValues {
		// 确定使用的键（优先使用字段名，如果没有映射则使用字段ID）
		key := fieldID
		if fieldName, ok := fieldIDToName[fieldID]; ok && fieldName != "" {
			key = fieldName
		}

		// 提取值
		if len(fieldValue.ExportedValue) > 0 {
			if len(fieldValue.ExportedValue) == 1 {
				businessData[key] = fieldValue.ExportedValue[0]
			} else {
				businessData[key] = fieldValue.ExportedValue
			}
		} else if len(fieldValue.TextValue) > 0 {
			if len(fieldValue.TextValue) == 1 {
				businessData[key] = fieldValue.TextValue[0]
			} else {
				businessData[key] = fieldValue.TextValue
			}
		} else if len(fieldValue.Value) > 0 {
			if len(fieldValue.Value) == 1 {
				businessData[key] = fieldValue.Value[0]
			} else {
				businessData[key] = fieldValue.Value
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

		// 发起人信息（直接使用本地用户ID）
		Initiator: j.User.ToDomain(userIDMapping),

		// 业务数据
		BusinessData: businessData,

		// 附件信息
		Attachments: attachments,
	}
}

// ToDomainWithFieldConfigs 将 API 响应转换为领域模型（使用事件配置的 DisplayName 作为键）
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
//   - fieldConfigs: 事件配置的字段列表，用于将 identity_key 转换为 DisplayName
//   - fieldMappings: 流程字段映射（identity_key → FieldMapping），用于获取字段ID和identity_key的关系
//
// 说明：
//   - 转换链：CachedValues[字段ID] → fieldMappings 获取 identity_key → fieldConfigs[field_name] → display_name
//   - 如果 fieldMappings 或 fieldConfigs 为空，则回退使用字段ID作为键
//   - 字段ID在 CachedValues 中是字符串类型的数字（如 "110"）
//   - FieldConfig.FieldName 存储的是 identity_key（如 "DateTime"）
func (j *JourneyDetailResponse) ToDomainWithFieldConfigs(userIDMapping map[int]string, fieldConfigs []*core.FieldConfig, fieldMappings map[string]core.FieldMapping) *core.JourneyDetail {
	// 1. 构建 字段ID → identity_key 的映射（从 fieldMappings 反向构建）
	// fieldMappings 的结构是 identity_key → FieldMapping，需要反向
	fieldIDToIdentityKey := make(map[string]string)
	if fieldMappings != nil {
		for identityKey, fm := range fieldMappings {
			fieldIDToIdentityKey[fmt.Sprintf("%d", fm.ID)] = identityKey
		}
	}

	// 2. 构建 identity_key → display_name 的映射（从 fieldConfigs）
	// fieldConfigs.FieldName 存储的就是 identity_key
	identityKeyToDisplayName := make(map[string]string)
	if len(fieldConfigs) > 0 {
		for _, fc := range fieldConfigs {
			identityKeyToDisplayName[fc.FieldName] = fc.DisplayName
		}
	}

	// 3. 构建业务数据（优先级：ExportedValue > TextValue > Value）
	businessData := make(map[string]interface{})
	for fieldID, fieldValue := range j.Response.CachedValues {
		// 确定使用的键（转换链：字段ID → identity_key → display_name）
		key := fieldID // 默认使用字段ID
		if identityKey, ok := fieldIDToIdentityKey[fieldID]; ok && identityKey != "" {
			// 找到了 identity_key，尝试获取 display_name
			if displayName, ok := identityKeyToDisplayName[identityKey]; ok && displayName != "" {
				key = displayName
			} else {
				// 没有配置 display_name，使用 identity_key
				key = identityKey
			}
		}

		// 提取值
		if len(fieldValue.ExportedValue) > 0 {
			if len(fieldValue.ExportedValue) == 1 {
				businessData[key] = fieldValue.ExportedValue[0]
			} else {
				businessData[key] = fieldValue.ExportedValue
			}
		} else if len(fieldValue.TextValue) > 0 {
			if len(fieldValue.TextValue) == 1 {
				businessData[key] = fieldValue.TextValue[0]
			} else {
				businessData[key] = fieldValue.TextValue
			}
		} else if len(fieldValue.Value) > 0 {
			if len(fieldValue.Value) == 1 {
				businessData[key] = fieldValue.Value[0]
			} else {
				businessData[key] = fieldValue.Value
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

		// 发起人信息（直接使用本地用户ID）
		Initiator: j.User.ToDomain(userIDMapping),

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
// 说明：该 API 直接返回数组，而非包装对象
type UserAssignmentsResponse []AssignmentResponse

// ProposedJourneysResponse Skylark API 返回的用户发起的流程列表结构体
// 职责：处理 GET /api/v4/yaw/flows/:flow_id/journeys/proposed_journeys 响应的 JSON 反序列化
// 说明：该 API 直接返回数组，而非包装对象
type ProposedJourneysResponse []JourneyResponse

// JourneySearchAPIResponse Skylark API 返回的搜索响应结构体
// 职责：处理 POST /api/v4/yaw/flows/:id/journeys/search 响应的 JSON 反序列化
// 说明：该 API 直接返回数组，而非包装对象；总数从响应头 X-SLP-Total-Count 获取
type JourneySearchAPIResponse []JourneyResponse

// MomentResponse Skylark API 返回的审批历史记录结构体
// 职责：处理 GET /api/v4/yaw/journeys/:id/moments 响应的 JSON 反序列化
// 说明：实际 API 返回的数据结构（与文档不一致）
type MomentResponse struct {
	ID         int64                 `json:"id"`         // 记录ID
	JourneyID  int64                 `json:"journey_id"` // 流程记录ID
	VertexID   int64                 `json:"vertex_id"`  // 节点ID
	Status     string                `json:"status"`     // 操作状态（proposed/approved/step_in/refused等）
	Comment    *string               `json:"comment"`    // 处理意见（可为空）
	CreatedAt  string                `json:"created_at"` // 创建时间（ISO 8601格式）
	UpdatedAt  string                `json:"updated_at"` // 更新时间（ISO 8601格式）
	Assignment *MomentAssignmentInfo `json:"assignment"` // 关联的任务信息（可为空）
	User       *MomentUserInfo       `json:"user"`       // 操作人信息（可为空）
}

// MomentAssignmentInfo API 返回的 Moment 关联任务信息结构体
// 说明：用于 MomentResponse 的嵌套对象
type MomentAssignmentInfo struct {
	ID       int64 `json:"id"`        // 任务ID
	VertexID int64 `json:"vertex_id"` // 节点ID
}

// MomentUserInfo API 返回的 Moment 操作人信息结构体
// 说明：用于 MomentResponse 的嵌套对象
type MomentUserInfo struct {
	ID   int64  `json:"id"`   // 用户ID
	Name string `json:"name"` // 用户名称
}

// ToDomain 将 API 响应转换为领域模型
// 参数：
//   - userIDMapping: 远程用户ID到本地用户ID的映射（int → string）
func (m *MomentResponse) ToDomain(userIDMapping map[int]string) *core.Moment {
	moment := &core.Moment{
		ID:        m.ID,
		JourneyID: m.JourneyID,
		VertexID:  m.VertexID,
		Status:    core.TranslateStatus(m.Status), // 中文显示名称
		StatusKey: m.Status,                       // 英文键值
		Comment:   m.Comment,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	// 从 assignment 中提取 AssignmentID
	if m.Assignment != nil {
		moment.AssignmentID = m.Assignment.ID
	}

	// 从 user 中提取操作人信息
	if m.User != nil {
		moment.OperatorID = userIDMapping[int(m.User.ID)] // 转换为本地用户ID
		moment.OperatorName = &m.User.Name
	}

	// 注意：实际 API 不返回 VertexName 和 Duration，保持为空/零值
	return moment
}

// AbortJourneyRequest 终止流程请求结构体
// 说明：用于 PUT /api/v4/yaw/flows/:flow_id/journeys/:id 请求体
type AbortJourneyRequest struct {
	Status string `json:"status"`          // 固定为 "aborted"
	Token  string `header:"Authorization"` // 认证令牌
}

// AbortJourneyResponse Skylark API 返回的终止流程响应结构体
// 说明：响应包含更新后的 journey 基本信息（简化版）
type AbortJourneyResponse struct {
	ID     int64  `json:"id"`     // 流程记录ID
	Status string `json:"status"` // 流程状态（应为 "aborted"）
}

// FlowRouteRequest 表示路由流程请求
type FlowRouteRequest struct {
	Assignment RouteAssignment `json:"assignment"`      // 分配信息
	UserID     int             `json:"user_id"`         // 用户ID
	Webhook    Webhook         `json:"Webhook"`         // Webhook配置
	Token      string          `header:"Authorization"` // 认证令牌
}

// RouteAssignment 表示路由分配信息
type RouteAssignment struct {
	Operation          string         `json:"operation"`           // 操作类型
	ResponseAttributes map[string]any `json:"response_attributes"` // 响应属性
}

// Webhook 表示Webhook配置
type Webhook struct {
	PayloadURL       string   `json:"payload_url"`       // 回调URL
	SubscribedEvents []string `json:"subscribed_events"` // 订阅事件
}

// FlowProposeRequest 表示提议流程请求
type FlowProposeRequest struct {
	Assignment ProposeAssignment `json:"assignment"`      // 分配信息
	UserID     int               `json:"user_id"`         // 用户ID
	Webhook    Webhook           `json:"Webhook"`         // Webhook配置
	Token      string            `header:"Authorization"` // 认证令牌
}

// ProposeAssignment 表示提议分配信息
type ProposeAssignment struct {
	Operation          string              `json:"operation"`           // 操作类型
	NextVertexID       int                 `json:"next_vertex_id"`      // 下一个节点ID
	DurationThresholds []map[string]string `json:"duration_thresholds"` // 持续时间阈值
}

// FlowRouteResponse 表示路由流程响应
type FlowRouteResponse struct {
	NextVertices []NextVertices `json:"next_vertices"` // 下一个节点列表
}

// NextVertices 表示下一个节点
type NextVertices struct {
	NextVerticesID int `json:"id"` // 下一个节点ID
}

// UpdateJourneyStatusRequest 表示更新流程任务状态请求（第一次和第二次请求共用）
type UpdateJourneyStatusRequest struct {
	Assignment UpdateAssignment `json:"assignment"`      // 任务更新信息
	UserID     int              `json:"user_id"`         // 用户ID（操作人ID）
	Token      string           `header:"Authorization"` // 认证令牌
}

// UpdateAssignment 表示任务更新分配信息
type UpdateAssignment struct {
	ResponseAttributes map[string]any `json:"response_attributes,omitempty"`  // 响应属性（包含 entries_attributes，仅第一次请求）
	Comment            string         `json:"comment,omitempty"`              // 处理意见（仅第二次请求）
	Operation          string         `json:"operation"`                      // 操作类型: route(第一次) 或 approve/refuse/transfer/cancel(第二次)
	NextVertexID       int            `json:"next_vertex_id,omitempty"`       // 下一个节点ID（仅第二次请求）
	CarbonCopyUserIDs  []int          `json:"carbon_copy_user_ids,omitempty"` // 抄送者ID列表（仅第二次请求）
}

// VertexDetailResponse Skylark API 返回的节点详情结构体
// 职责：处理 GET /api/v4/yaw/flows/:flow_id/vertices/:id 响应的 JSON 反序列化
// 说明：包含节点的基础信息和字段列表
type VertexDetailResponse struct {
	ID        int64                  `json:"id"`         // 节点ID
	Name      string                 `json:"name"`       // 节点名称
	Type      string                 `json:"type"`       // 节点类型（如 YetAnotherWorkflow::Vertex::Normal）
	AliasName string                 `json:"alias_name"` // 节点别名
	Fields    []VertexFieldResponse  `json:"fields"`     // 节点字段列表
}

// VertexFieldResponse 节点字段响应结构体
// 职责：处理节点详情 API 中的 fields 数组元素
// 说明：包含字段的基础信息、权限信息和选项列表
type VertexFieldResponse struct {
	ID          int64                       `json:"id"`                // 字段ID
	IdentityKey string                      `json:"identity_key"`      // 字段唯一标识键
	Title       string                      `json:"title"`             // 字段标题
	Type        string                      `json:"type"`              // 字段类型（如 Field::RadioButton）
	Validations []string                    `json:"validations"`       // 验证规则（包含 "presence" 表示必填）
	Visibility  string                      `json:"visibility"`        // 可见性（public_visibility/protected_visibility/private_visibility）
	Settings    VertexFieldSettings         `json:"settings"`          // 字段设置（包含字数限制等）
	Options     []VertexFieldOptionResponse `json:"options,omitempty"` // 字段选项列表（仅选项字段有值）
}

// VertexFieldSettings 节点字段设置结构体
type VertexFieldSettings struct {
	CharSizeLimitSettings CharSizeLimitSettings `json:"char_size_limit_settings"` // 字数限制设置
}

// CharSizeLimitSettings 字数限制设置
type CharSizeLimitSettings struct {
	Lteq int `json:"lteq"` // 最大字数限制（小于等于）
}

// VertexFieldOptionResponse 节点字段选项响应结构体
// 职责：处理节点字段中的 options 数组元素
// 说明：表示选项类型字段的可选值
type VertexFieldOptionResponse struct {
	ID       int                    `json:"id"`                 // 选项ID
	Value    string                 `json:"value"`              // 选项值
	Settings map[string]interface{} `json:"settings,omitempty"` // 选项设置
	Position int                    `json:"position"`           // 选项位置
}

// ToDomain 将 VertexDetailResponse 转换为字段列表
// 说明：只提取字段信息（节点基础信息已在 FlowVertex 中）
func (v *VertexDetailResponse) ToDomain() []*core.VertexField {
	fields := make([]*core.VertexField, len(v.Fields))
	for i, field := range v.Fields {
		fields[i] = field.ToDomain()
	}
	return fields
}

// ToDomain 将 VertexFieldResponse 转换为领域模型
func (vf *VertexFieldResponse) ToDomain() *core.VertexField {
	// 转换选项列表
	options := make([]core.FieldOption, len(vf.Options))
	for i, opt := range vf.Options {
		options[i] = core.FieldOption{
			ID:       opt.ID,
			Value:    opt.Value,
			Settings: opt.Settings,
			Position: opt.Position,
		}
	}

	// 从 validations 推断是否必填（包含 "presence" 表示必填）
	required := false
	for _, v := range vf.Validations {
		if v == "presence" {
			required = true
			break
		}
	}

	// 从 visibility 推断是否可编辑
	// public_visibility: 所有人能填 → true
	// protected_visibility: 普通用户只能查看不能填写 → false
	// private_visibility: 仅管理员可查看 → false
	editable := vf.Visibility == "public_visibility"

	return &core.VertexField{
		ID:          vf.ID,
		IdentityKey: vf.IdentityKey,
		Title:       vf.Title,
		Type:        vf.Type,
		Required:    required,
		Editable:    editable,
		MaxLength:   vf.Settings.CharSizeLimitSettings.Lteq,
		Options:     options,
	}
}
