package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"
)

// GetFlowList 从缓存获取 flows 列表
func (f *SkylarkCache) GetFlowList(ctx context.Context, tenantID string, namespaceID int) ([]*core.FlowInfo, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowListKeyPrefix, tenantID, namespaceID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var flows []*core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flows); err != nil {
		return nil, fmt.Errorf("反序列化 flows 缓存失败: %w", err)
	}

	return flows, nil
}

// SetFlowList 缓存 flows 列表
func (f *SkylarkCache) SetFlowList(ctx context.Context, tenantID string, namespaceID int, flows []*core.FlowInfo, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowListKeyPrefix, tenantID, namespaceID)
	data, err := json.Marshal(flows)
	if err != nil {
		return fmt.Errorf("序列化 flows 列表失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// GetFlowFields 从缓存获取 flow 字段列表
func (f *SkylarkCache) GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var fields []*core.FieldMetadata
	if err := json.Unmarshal([]byte(val), &fields); err != nil {
		return nil, fmt.Errorf("反序列化 flow fields 缓存失败: %w", err)
	}

	return fields, nil
}

// SetFlowFields 缓存 flow 字段列表
func (f *SkylarkCache) SetFlowFields(ctx context.Context, tenantID string, flowID int, fields []*core.FieldMetadata, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	data, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("序列化 flow fields 列表失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}
