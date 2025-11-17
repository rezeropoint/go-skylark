package organization

import (
	"net/http"
	"time"
)

// skylarkHTTPClient Skylark HTTP 客户端封装
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

// CreateOrganizationRequest 创建组织请求
type CreateOrganizationRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	ApplicationStrategy string `json:"application_strategy"` // "must_approved", "auto_approved", "closed"
	FounderID           int    `json:"founder_id"`
	ParentID            *int   `json:"parent_id,omitempty"` // 可选，创建子组织时使用
}

// OrganizationResponse Skylark 组织响应
type OrganizationResponse struct {
	ID                  int     `json:"id"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	DescriptionText     string  `json:"description_text"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	ChildrenCount       int     `json:"children_count"`
	ParentID            *int    `json:"parent_id"`
	Ancestry            *string `json:"ancestry"` // 祖先路径，如 "14516/15113"
	ApplicationStrategy string  `json:"application_strategy"`
}
