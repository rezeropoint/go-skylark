package forms

// FormCreateRequest 表示表单创建请求
type FormCreateRequest struct {
	Response Response `json:"response"`        // 分配信息
	UserID   int      `json:"user_id"`         // 用户ID
	Token    string   `header:"Authorization"` // 认证令牌
}

// Response 表示路由分配信息
type Response struct {
	EntriesAttributes []map[string]any `json:"entries_attributes"` // 响应属性
}
