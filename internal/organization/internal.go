package organization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"
)

// createOrganization 调用 Skylark API 创建组织
func (c *skylarkHTTPClient) createOrganization(ctx context.Context, apiBaseURL, apiToken string, req *CreateOrganizationRequest) (*OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations", apiBaseURL)

	// 序列化请求体
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
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
	var orgResp OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgResp); err != nil {
		return nil, err
	}

	return &orgResp, nil
}

// deleteOrganization 调用 Skylark API 删除组织
func (c *skylarkHTTPClient) deleteOrganization(ctx context.Context, apiBaseURL, apiToken string, remoteOrgID int) error {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d", apiBaseURL, remoteOrgID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应（DELETE请求无响应体，传入nil）
	return httputils.ReadJSONResponse(resp, nil)
}

// listOrganizations 调用 Skylark API 列出所有组织
func (c *skylarkHTTPClient) listOrganizations(ctx context.Context, apiBaseURL, apiToken string) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations", apiBaseURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// listRootOrganizations 调用 Skylark API 列出根组织
func (c *skylarkHTTPClient) listRootOrganizations(ctx context.Context, apiBaseURL, apiToken string) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/roots", apiBaseURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// listChildOrganizations 调用 Skylark API 列出子组织
func (c *skylarkHTTPClient) listChildOrganizations(ctx context.Context, apiBaseURL, apiToken string, parentRemoteOrgID int) ([]OrganizationResponse, error) {
	// 构建 URL
	url := fmt.Sprintf("https://%s/api/v4/organizations/%d/children", apiBaseURL, parentRemoteOrgID)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", apiToken)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrSkylarkAPIServerError, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 处理响应
	var orgs []OrganizationResponse
	if err := httputils.ReadJSONResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}
