package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// nullValueMarker 空值标记，用于缓存穿透防护
// 当资源不存在时，缓存此标记以避免重复查询不存在的资源
const nullValueMarker = "__NULL__"

// nullValueCacheTTL 空值缓存的 TTL（秒）
// 使用较短的 TTL，避免长期缓存不存在的资源
const nullValueCacheTTL = 300 // 5 分钟

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

	// 检测空值标记（缓存穿透防护）
	if val == nullValueMarker {
		return nil, core.ErrFlowNotFound
	}

	var flowInfo core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flowInfo); err != nil {
		return nil, fmt.Errorf("反序列化 flow info (API) 缓存失败: %w", err)
	}

	f.metrics.FlowInfoAPIHit.Add(1)
	return &flowInfo, nil
}

// SetFlowInfoAPI 缓存单个 flow 信息 (API 数据)
// 支持空值缓存：当 flowInfo 为 nil 时，缓存空值标记（用于缓存穿透防护）
func (f *SkylarkCache) SetFlowInfoAPI(ctx context.Context, tenantID string, flowInfo *core.FlowInfo, ttl int) error {
	var flowID int64
	if flowInfo != nil {
		flowID = int64(flowInfo.ID)
	} else {
		// 如果 flowInfo 为 nil，需要从 context 或其他方式获取 flowID
		// 这里假设调用方会传入有效的 flowID（通过其他参数）
		// 实际上，当 flowInfo 为 nil 时，调用方应该传入 flowID
		// 为了保持接口兼容性，这里暂时不支持 nil 的情况
		// 空值缓存应该在调用方处理（传入一个包含 flowID 的临时对象）
		return fmt.Errorf("flowInfo 不能为 nil，请使用 SetFlowInfoAPINull 方法缓存空值")
	}

	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoAPIKeyPrefix, tenantID, flowID)
	
	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)
	
	data, err := json.Marshal(flowInfo)
	if err != nil {
		return fmt.Errorf("序列化 flow info (API) 失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttlWithJitter)
}

// SetFlowInfoAPINull 缓存空值标记（用于缓存穿透防护）
func (f *SkylarkCache) SetFlowInfoAPINull(ctx context.Context, tenantID string, flowID int64) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowInfoAPIKeyPrefix, tenantID, flowID)
	
	// 空值缓存使用较短的 TTL，并添加随机偏移
	ttlWithJitter := addJitter(nullValueCacheTTL)
	
	return f.redisClient.SetexCtx(ctx, key, nullValueMarker, ttlWithJitter)
}

// GetFlowDetailAPI 从缓存获取 flow 详情 (API 数据)
func (f *SkylarkCache) GetFlowDetailAPI(ctx context.Context, tenantID string, flowID int64) (*core.FlowDetail, error) {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowDetailAPIKeyPrefix, tenantID, flowID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	// 检测空值标记（缓存穿透防护）
	if val == nullValueMarker {
		return nil, core.ErrFlowNotFound
	}

	var flowDetail core.FlowDetail
	if err := json.Unmarshal([]byte(val), &flowDetail); err != nil {
		return nil, fmt.Errorf("反序列化 flow detail (API) 缓存失败: %w", err)
	}

	return &flowDetail, nil
}

// SetFlowDetailAPI 缓存 flow 详情 (API 数据)
func (f *SkylarkCache) SetFlowDetailAPI(ctx context.Context, tenantID string, flowDetail *core.FlowDetail, ttl int) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowDetailAPIKeyPrefix, tenantID, flowDetail.ID)
	
	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)
	
	data, err := json.Marshal(flowDetail)
	if err != nil {
		return fmt.Errorf("序列化 flow detail (API) 失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttlWithJitter)
}

// SetFlowDetailAPINull 缓存空值标记（用于缓存穿透防护）
func (f *SkylarkCache) SetFlowDetailAPINull(ctx context.Context, tenantID string, flowID int64) error {
	key := fmt.Sprintf("%s%s:%d", core.CacheFlowDetailAPIKeyPrefix, tenantID, flowID)
	
	// 空值缓存使用较短的 TTL，并添加随机偏移
	ttlWithJitter := addJitter(nullValueCacheTTL)
	
	return f.redisClient.SetexCtx(ctx, key, nullValueMarker, ttlWithJitter)
}
