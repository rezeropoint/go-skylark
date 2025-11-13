package forms

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"
	"github.com/rezeropoint/go-skylark/v2/internal/images"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// getFormFieldMappings 获取表单字段映射
// 先尝试从缓存获取，缓存未命中则从API获取
func (f *skylarkFormRegistry) getFormFieldMappings(ctx context.Context, skylarkFormAddress core.SkylarkAPIContext, formID int64) (map[string]core.FieldMapping, error) {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%s%s:%d", core.CacheFieldMappingKeyPrefix, skylarkFormAddress.App, formID)

	// 尝试从缓存中获取
	fieldMappings, found, err := f.cache.GetFieldMappingsFromCache(ctx, cacheKey)
	if err != nil {
		// 缓存查询出错，记录错误但继续执行，不影响正常流程
		fmt.Printf("从缓存获取字段映射时发生错误: %v\n", err)
	} else if found {
		// 缓存命中，直接返回
		return fieldMappings, nil
	}

	// 缓存未命中或已过期，发起请求
	formAPIURL := core.BuildFormAPIURL(skylarkFormAddress, formID)
	resp, err := httpc.Do(ctx, http.MethodGet, formAPIURL, core.AuthHeader{Token: skylarkFormAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 统一处理响应
	var respField core.Field
	if err := httputils.ReadJSONResponse(resp, &respField); err != nil {
		return nil, err
	}

	fieldMappings = make(map[string]core.FieldMapping, len(respField.Fields))
	for _, field := range respField.Fields {
		fieldMappings[field.IdentityKey] = field
	}

	// 将结果存入缓存
	if cacheErr := f.cache.SaveFieldMappingsToCache(ctx, cacheKey, fieldMappings); cacheErr != nil {
		// 缓存保存失败只记录错误，不影响正常流程
		fmt.Printf("保存字段映射到缓存时发生错误: %v\n", cacheErr)
	}

	return fieldMappings, nil
}

// buildFormCreateRequest 构建表单创建请求
// 根据原始数据和字段映射构建请求体
func (f *skylarkFormRegistry) buildFormCreateRequest(ctx context.Context, skylarkFormAddress core.SkylarkAPIContext, originalData map[string]core.TypedValue, fieldMappings map[string]core.FieldMapping) (FormCreateRequest, error) {
	entries := make([]map[string]any, 0, len(originalData))

	for key, TypedValue := range originalData {
		// 先判断 key 是否存在于 fieldMappings 中
		if fieldMapping, ok := fieldMappings[key]; ok {
			fieldId := fieldMapping.ID

			switch {
			case TypedValue.Type == string(core.FieldImage):
				// 提取图片URL
				imageURL, ok := TypedValue.Value.(string)
				if !ok {
					return FormCreateRequest{}, fmt.Errorf("%w: 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 创建图片字段的 entry
				id, name, err := images.CreateImageEntryFromURL(ctx, skylarkFormAddress, imageURL)
				if err != nil {
					return FormCreateRequest{}, err
				}
				// 创建 entry 并添加到 entries
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    name,
						"value_id": id,
					})
			case TypedValue.Type == string(core.FieldImageBase64):
				// 提取 base64 数据
				base64Data, ok := TypedValue.Value.(string)
				if !ok {
					return FormCreateRequest{}, fmt.Errorf("%w: Base64 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 上传 base64 图片
				id, name, err := images.CreateImageEntryFromBase64(ctx, skylarkFormAddress, base64Data)
				if err != nil {
					return FormCreateRequest{}, err
				}
				// 创建 entry 并添加到 entries
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    name,
						"value_id": id,
					})
			case TypedValue.Type == string(core.FieldString) && core.IsOptionField(fieldMapping.Type):
				// 处理选项字段(仅限字符串类型)
				valueStr, ok := TypedValue.Value.(string)
				if !ok {
					return FormCreateRequest{}, fmt.Errorf("%w: 选项字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 查找匹配的选项
				optionId := 0
				for _, option := range fieldMapping.Options {
					if option.Value == valueStr {
						optionId = option.ID
						break
					}
				}

				if optionId == 0 {
					return FormCreateRequest{}, fmt.Errorf("%w: 选项字段 %s 的值 %s 不存在", core.ErrOptionNotFound, key, valueStr)
				}

				// 找到匹配的选项，添加option_id
				entries = append(
					entries,
					map[string]any{
						"field_id":  fieldId,
						"value":     valueStr,
						"option_id": optionId,
					})
			default:
				// 不需要特殊处理，直接添加。因为any类型可以包含所有类型
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    TypedValue.Value,
					})
			}
		}
	}
	if len(entries) == 0 {
		return FormCreateRequest{}, fmt.Errorf("%w: 字段映射为: %+v", core.ErrFieldMappingEmpty, fieldMappings)
	}
	userID, err := strconv.Atoi(skylarkFormAddress.UserID)
	if err != nil {
		return FormCreateRequest{}, fmt.Errorf("%w: %v", core.ErrUserIDConversionFailed, err)
	}
	// 创建符合 Request 结构体的数据
	return FormCreateRequest{
		Response: Response{
			EntriesAttributes: entries,
		},
		UserID: userID,
		Token:  skylarkFormAddress.AuthHeader,
	}, nil
}
