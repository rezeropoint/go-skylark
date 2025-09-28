package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/images"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// getFlowFieldMappings 获取流程字段映射
// 先尝试从缓存获取，缓存未命中则从API获取
func (f *skylarkFlowRegistry) getFlowFieldMappings(ctx context.Context, skylarkFlowAddress core.BasicSkylarkAddress, flowID int64) (map[string]core.FieldMapping, error) {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%s:%d", skylarkFlowAddress.App, flowID)

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
	flowAPIURL := core.BuildFlowAPIURL(skylarkFlowAddress, flowID)
	resp, err := httpc.Do(ctx, http.MethodGet, flowAPIURL, core.AuthHeader{Token: skylarkFlowAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 从响应体中读取数据
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: 状态码: %d, 响应体: %s", core.ErrHTTPRequestFailed, resp.StatusCode, string(respBody))
	}

	// 反序列化 JSON 数据到结构体
	var respField core.Field
	err = json.Unmarshal(respBody, &respField)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
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

// buildFlowRouteRequest 构建流程路由请求
// 根据原始数据和字段映射构建请求体
func (f *skylarkFlowRegistry) buildFlowRouteRequest(ctx context.Context, skylarkFlowAddress core.BasicSkylarkAddress, flowID int64, originalData map[string]core.TypedValue, fieldMappings map[string]core.FieldMapping) (FlowRouteRequest, error) {
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
					return FlowRouteRequest{}, fmt.Errorf("%w: 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 创建图片字段的 entry
				id, name, err := images.CreateImageEntryFromURL(ctx, skylarkFlowAddress, imageURL)
				if err != nil {
					return FlowRouteRequest{}, err
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
					return FlowRouteRequest{}, fmt.Errorf("%w: Base64 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 上传 base64 图片
				id, name, err := images.CreateImageEntryFromBase64(ctx, skylarkFlowAddress, base64Data)
				if err != nil {
					return FlowRouteRequest{}, err
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
					return FlowRouteRequest{}, fmt.Errorf("%w: 选项字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
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
					return FlowRouteRequest{}, fmt.Errorf("%w: 选项字段 %s 的值 %s 不存在", core.ErrOptionNotFound, key, valueStr)
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
		// 字段映射为空，可能是缓存没有更新，清除字段映射缓存
		cacheKey := fmt.Sprintf("%s:%d", skylarkFlowAddress.App, flowID)
		if cacheErr := f.cache.ClearFieldMappingsCache(ctx, cacheKey); cacheErr != nil {
			// 缓存清除失败，返回包含缓存清除失败信息的错误
			return FlowRouteRequest{}, fmt.Errorf("%w: 字段映射为: %+v, 缓存清除错误为: %v", core.ErrFieldMappingEmptyAndCacheClearFailed, fieldMappings, cacheErr)
		}
		// 缓存清除成功，返回正常的字段映射为空错误
		return FlowRouteRequest{}, fmt.Errorf("%w: 字段映射为: %+v", core.ErrFieldMappingEmpty, fieldMappings)
	}
	userID, err := strconv.Atoi(skylarkFlowAddress.UserID)
	if err != nil {
		return FlowRouteRequest{}, fmt.Errorf("%w: %v", core.ErrUserIDConversionFailed, err)
	}
	// 创建符合 Request 结构体的数据
	return FlowRouteRequest{
		Assignment: RouteAssignment{
			Operation: core.OperationRoute,
			ResponseAttributes: map[string]any{
				"entries_attributes": entries,
			},
		},
		UserID: userID,
		Webhook: Webhook{
			PayloadURL:       "",
			SubscribedEvents: []string{core.EventJourneyStatus},
		},
		Token: skylarkFlowAddress.AuthHeader,
	}, nil
}

// buildFlowProposeRequest 构建流程提议请求
// 根据路由流程响应构建提议请求
func (f *skylarkFlowRegistry) buildFlowProposeRequest(skylarkFlowAddress core.BasicSkylarkAddress, routeFlowBody []byte) (FlowProposeRequest, error) {
	// 解析JSON响应
	var result FlowRouteResponse
	if err := json.Unmarshal(routeFlowBody, &result); err != nil {
		return FlowProposeRequest{}, fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	// 检查 NextVertices 是否为空
	if len(result.NextVertices) == 0 {
		var resultJSON map[string]any
		_ = json.Unmarshal(routeFlowBody, &resultJSON)
		return FlowProposeRequest{}, fmt.Errorf("%w: 响应为: %v", core.ErrNoNextVertices, resultJSON)
	}

	id := result.NextVertices[0].NextVerticesID

	userID, err := strconv.Atoi(skylarkFlowAddress.UserID)
	if err != nil {
		return FlowProposeRequest{}, fmt.Errorf("%w: %v", core.ErrUserIDConversionFailed, err)
	}
	return FlowProposeRequest{
		Assignment: ProposeAssignment{
			Operation:          core.OperationPropose,
			NextVertexID:       id,
			DurationThresholds: []map[string]string{},
		},
		UserID: userID,
		Webhook: Webhook{
			PayloadURL:       "",
			SubscribedEvents: []string{core.EventJourneyStatus},
		},
		Token: skylarkFlowAddress.AuthHeader,
	}, nil
}
