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
