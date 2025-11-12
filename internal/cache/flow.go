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
		// 缓存未命中
		f.metrics.FlowListMiss.Add(1)
		return nil, err
	}

	var flows []*core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flows); err != nil {
		return nil, fmt.Errorf("反序列化 flows 缓存失败: %w", err)
	}

	// 缓存命中
	f.metrics.FlowListHit.Add(1)
	return flows, nil
}

// SetFlowList 缓存 flows 列表
func (f *SkylarkCache) SetFlowList(ctx context.Context, tenantID string, namespaceID int, flows []*core.FlowInfo, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowListKeyPrefix, tenantID, namespaceID)
	data, err := json.Marshal(flows)
	if err != nil {
		return fmt.Errorf("序列化 flows 列表失败: %w", err)
	}

	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)

	// 缓存列表
	return f.redisClient.SetexCtx(ctx, key, string(data), ttlWithJitter)
}

// GetFlowFields 从缓存获取 flow 字段列表
func (f *SkylarkCache) GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		f.metrics.FlowFieldsMiss.Add(1)
		return nil, err
	}

	var fields []*core.FieldMetadata
	if err := json.Unmarshal([]byte(val), &fields); err != nil {
		return nil, fmt.Errorf("反序列化 flow fields 缓存失败: %w", err)
	}

	f.metrics.FlowFieldsHit.Add(1)
	return fields, nil
}

// SetFlowFields 缓存 flow 字段列表
func (f *SkylarkCache) SetFlowFields(ctx context.Context, tenantID string, flowID int, fields []*core.FieldMetadata, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	data, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("序列化 flow fields 列表失败: %w", err)
	}

	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)

	return f.redisClient.SetexCtx(ctx, key, string(data), ttlWithJitter)
}

// GetFlowInfo 从缓存获取单个 flow 信息
func (f *SkylarkCache) GetFlowInfo(ctx context.Context, tenantID string, flowID int64) (*core.FlowInfo, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoKeyPrefix, tenantID, flowID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		f.metrics.FlowInfoMiss.Add(1)
		return nil, err
	}

	var flowInfo core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flowInfo); err != nil {
		return nil, fmt.Errorf("反序列化 flow info 缓存失败: %w", err)
	}

	f.metrics.FlowInfoHit.Add(1)
	return &flowInfo, nil
}

// SetFlowInfo 缓存单个 flow 信息 (数据库数据)
func (f *SkylarkCache) SetFlowInfo(ctx context.Context, tenantID string, flowInfo *core.FlowInfo, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoKeyPrefix, tenantID, flowInfo.ID)
	data, err := json.Marshal(flowInfo)
	if err != nil {
		return fmt.Errorf("序列化 flow info 失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// GetFlowInfoAPI 从缓存获取单个 flow 信息 (API 数据)
func (f *SkylarkCache) GetFlowInfoAPI(ctx context.Context, tenantID string, flowID int64) (*core.FlowInfo, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoAPIKeyPrefix, tenantID, flowID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		f.metrics.FlowInfoAPIMiss.Add(1)
		return nil, err
	}

	var flowInfo core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flowInfo); err != nil {
		return nil, fmt.Errorf("反序列化 flow info (API) 缓存失败: %w", err)
	}

	f.metrics.FlowInfoAPIHit.Add(1)
	return &flowInfo, nil
}

// SetFlowInfoAPI 缓存单个 flow 信息 (API 数据)
func (f *SkylarkCache) SetFlowInfoAPI(ctx context.Context, tenantID string, flowInfo *core.FlowInfo, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoAPIKeyPrefix, tenantID, flowInfo.ID)
	data, err := json.Marshal(flowInfo)
	if err != nil {
		return fmt.Errorf("序列化 flow info (API) 失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}
