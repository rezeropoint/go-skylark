package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// GetOrgMapping 从缓存获取单个组织映射
func (f *SkylarkCache) GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error) {
	key := core.CacheMappingKeyPrefix + id
	return getJSONWithNullCheck[core.OrgMapping](
		ctx,
		f.redisClient,
		key,
		nullValueMarker,
		core.ErrOrgMappingNotFound,
	)
}

// SetOrgMapping 缓存单个组织映射
func (f *SkylarkCache) SetOrgMapping(ctx context.Context, mapping *core.OrgMapping, ttl int) error {
	key := core.CacheMappingKeyPrefix + mapping.ID
	data, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("序列化组织映射失败: %w", err)
	}

	return setJSONWithJitter(ctx, f.redisClient, key, data, ttl)
}

// SetOrgMappingNull 缓存空值标记（用于缓存穿透防护）
func (f *SkylarkCache) SetOrgMappingNull(ctx context.Context, id string) error {
	key := core.CacheMappingKeyPrefix + id
	return setNullValue(ctx, f.redisClient, key, nullValueMarker, nullValueCacheTTL)
}

// DeleteOrgMapping 删除单个组织映射缓存
func (f *SkylarkCache) DeleteOrgMapping(ctx context.Context, id string) error {
	key := core.CacheMappingKeyPrefix + id
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}

// GetOrgMappingList 从缓存获取组织映射列表
func (f *SkylarkCache) GetOrgMappingList(ctx context.Context, tenantID string) ([]*core.OrgMapping, error) {
	key := core.CacheMappingListKeyPrefix + tenantID
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var mappings []*core.OrgMapping
	if err := json.Unmarshal([]byte(val), &mappings); err != nil {
		return nil, fmt.Errorf("反序列化组织映射列表缓存失败: %w", err)
	}

	return mappings, nil
}

// SetOrgMappingList 缓存组织映射列表
func (f *SkylarkCache) SetOrgMappingList(ctx context.Context, tenantID string, mappings []*core.OrgMapping, ttl int) error {
	key := core.CacheMappingListKeyPrefix + tenantID
	data, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("序列化组织映射列表失败: %w", err)
	}

	return setJSONWithJitter(ctx, f.redisClient, key, data, ttl)
}

// DeleteOrgMappingList 删除租户的组织映射列表缓存
func (f *SkylarkCache) DeleteOrgMappingList(ctx context.Context, tenantID string) error {
	key := core.CacheMappingListKeyPrefix + tenantID
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}
