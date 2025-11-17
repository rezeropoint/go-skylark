package user

import (
	"net/http"
	"time"
)

// skylarkHTTPClient Skylark用户API客户端
//
// 说明：
//   - 封装 Skylark 用户相关 REST API 调用
//   - 使用原生 net/http（30秒超时）
//   - 使用 httputils 统一处理错误和响应
type skylarkHTTPClient struct {
	httpClient *http.Client
}

// newSkylarkHTTPClient 创建 HTTP 客户端
func newSkylarkHTTPClient() *skylarkHTTPClient {
	return &skylarkHTTPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateUserRequest 创建用户请求
//
// API端点: POST /api/v4/users
// 请求体格式: JSON
type CreateUserRequest struct {
	Name       string  `json:"name"`                 // 用户姓名（必填）
	Identifier *string `json:"identifier,omitempty"` // 用户标识符（可选）
	Phone      *string `json:"phone,omitempty"`      // 手机号（可选）
	Openid     *string `json:"openid,omitempty"`     // 微信OpenID（可选）
}

// UserResponse Skylark用户响应
//
// API返回格式: JSON
// 说明：包含Skylark创建用户后返回的字段
type UserResponse struct {
	ID         int     `json:"id"`         // Skylark用户ID（整数）
	Name       string  `json:"name"`       // 用户姓名
	Identifier *string `json:"identifier"` // 用户标识符（可空）
	Phone      *string `json:"phone"`      // 手机号（可空）
	Openid     *string `json:"openid"`     // 微信OpenID（可空）
	CreatedAt  string  `json:"created_at"` // 创建时间（ISO8601格式）
	UpdatedAt  string  `json:"updated_at"` // 更新时间（ISO8601格式）
}
