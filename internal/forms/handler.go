package forms

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/httputils"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// skylarkFormRegistry 流程注册表结构
type skylarkFormRegistry struct {
	cache core.CacheInterface
}

// newSkylarkFormRegistry 创建新的流程注册表
func newSkylarkFormRegistry(config *Config, cache core.CacheInterface) (*skylarkFormRegistry, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}

	return &skylarkFormRegistry{
		cache: cache,
	}, nil
}

func (f *skylarkFormRegistry) CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	// 构建流程地址信息
	skylarkFormAddress := core.SkylarkAPIContext{
		App:        app,
		UserID:     strconv.FormatInt(userID, 10),
		AuthHeader: authHeader,
	}

	// 获取字段映射
	FormFieldMappings, err := f.getFormFieldMappings(ctx, skylarkFormAddress, formID)
	if err != nil {
		return err
	}

	// 构建 Skylark 流程路由请求体
	formCreateRequest, err := f.buildFormCreateRequest(ctx, skylarkFormAddress, data, FormFieldMappings)
	if err != nil {
		return err
	}
	// 构建API请求URL
	apiURL := core.BuildFormAPIURL(skylarkFormAddress, formID, "responses")

	// 发送 Skylark 流程路由请求
	formCreateRowResult, err := httpc.Do(ctx, http.MethodPost, apiURL, formCreateRequest)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer formCreateRowResult.Body.Close()

	// 使用 httputils 统一处理响应（创建表单行无需解析响应体）
	if err := httputils.ReadJSONResponse(formCreateRowResult, nil); err != nil {
		return err
	}

	return nil
}
