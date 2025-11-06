package flows

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

// UpdateJourneyStatusRequest 表示更新流程任务状态请求
type UpdateJourneyStatusRequest struct {
	Assignment UpdateAssignment `json:"assignment"`      // 任务更新信息
	UserID     int              `json:"user_id"`         // 用户ID
	Token      string           `header:"Authorization"` // 认证令牌
}

// UpdateAssignment 表示任务更新分配信息
type UpdateAssignment struct {
	ResponseAttributes map[string]any      `json:"response_attributes"`   // 响应属性（包含 entries_attributes）
	Comment            string              `json:"comment,omitempty"`     // 处理意见
	Operation          string              `json:"operation"`             // 操作类型: approve/refuse/transfer/cancel
	NextVertexID       int                 `json:"next_vertex_id"`        // 下一个节点ID
	CarbonCopyUserIDs  []int               `json:"carbon_copy_user_ids"`  // 抄送者ID列表
	DurationThresholds []DurationThreshold `json:"duration_thresholds"`   // 持续时间阈值
}

// DurationThreshold 表示持续时间阈值
type DurationThreshold struct {
	GID   string `json:"gid"`   // 全局标识符
	Value string `json:"value"` // 处理时限
}
