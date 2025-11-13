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
