package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/httputils"
)

// skylarkHTTPClient Skylark用户API客户端
//
// 说明：
//  - 封装 Skylark 用户相关 REST API 调用
//  - 使用原生 net/http（30秒超时）
//  - 使用 httputils 统一处理错误和响应
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

// createUser 调用Skylark API创建用户
//
// API端点: POST https://{apiBaseURL}/api/v4/users
// 请求头:
//   - Authorization: {apiToken}
//   - Content-Type: application/json
//
// 参数：
//   - ctx: 上下文
//   - apiBaseURL: Skylark API地址（如: skylark.example.com）
//   - apiToken: API认证Token
//   - req: 创建用户请求
//
// 返回：
//   - *UserResponse: 用户响应（包含远程用户ID）
//   - error: 错误信息
//
// 错误：
//   - core.ErrSkylarkAPIUnauthorized: 401 认证失败
//   - core.ErrSkylarkAPIBadRequest: 400 请求参数错误
//   - core.ErrSkylarkAPIServerError: 500 服务器错误
func (c *skylarkHTTPClient) createUser(ctx context.Context, apiBaseURL, apiToken string, req *CreateUserRequest) (*UserResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/users", apiBaseURL)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（自动处理错误和JSON解析）
	var userResp UserResponse
	if err := httputils.ReadJSONResponse(resp, &userResp); err != nil {
		return nil, err
	}

	return &userResp, nil
}
