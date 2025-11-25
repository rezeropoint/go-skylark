package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// SaveFieldMappingsToCache 保存字段映射到缓存
func (f *SkylarkCache) SaveFieldMappingsToCache(ctx context.Context, cacheKey string, fieldMappings map[string]core.FieldMapping) error {
	data, err := json.Marshal(fieldMappings)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	err = f.redisClient.SetexCtx(ctx, cacheKey, string(data), f.config.FieldMappingsCacheExpiry)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	return nil
}

// GetFieldMappingsFromCache 从缓存获取字段映射
func (f *SkylarkCache) GetFieldMappingsFromCache(ctx context.Context, cacheKey string) (map[string]core.FieldMapping, bool, error) {
	data, err := f.redisClient.GetCtx(ctx, cacheKey)
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	if data == "" {
		return nil, false, nil
	}

	var fieldMappings map[string]core.FieldMapping
	err = json.Unmarshal([]byte(data), &fieldMappings)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	return fieldMappings, true, nil
}

// ClearFieldMappingsCache 清除指定的字段映射缓存
func (f *SkylarkCache) ClearFieldMappingsCache(ctx context.Context, cacheKey string) error {
	_, err := f.redisClient.DelCtx(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}
	return nil
}

// GetVertexFieldsFromCache 从缓存获取节点字段信息
// 参数：
//   - ctx: 上下文
//   - cacheKey: 缓存键（格式：skylark:vertex:{tenantID}:{flowID}:{vertexID}）
//
// 返回：
//   - []*core.VertexField: 节点字段列表
//   - bool: 是否命中缓存
//   - error: 错误信息
func (f *SkylarkCache) GetVertexFieldsFromCache(ctx context.Context, cacheKey string) ([]*core.VertexField, bool, error) {
	data, err := f.redisClient.GetCtx(ctx, cacheKey)
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	if data == "" {
		return nil, false, nil
	}

	var fields []*core.VertexField
	err = json.Unmarshal([]byte(data), &fields)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	return fields, true, nil
}

// SaveVertexFieldsToCache 保存节点字段信息到缓存
// 参数：
//   - ctx: 上下文
//   - cacheKey: 缓存键（格式：skylark:vertex:{tenantID}:{flowID}:{vertexID}）
//   - fields: 节点字段列表
//   - ttl: 缓存过期时间（秒）
//
// 返回：
//   - error: 错误信息
func (f *SkylarkCache) SaveVertexFieldsToCache(ctx context.Context, cacheKey string, fields []*core.VertexField, ttl int) error {
	data, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	err = f.redisClient.SetexCtx(ctx, cacheKey, string(data), ttl)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	return nil
}
