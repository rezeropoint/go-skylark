package mapping

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"
)

// getCachedMapping 从缓存获取单个组织映射（根据ID）
func (m *mappingManager) getCachedMapping(ctx context.Context, id string) (*core.OrgMapping, error) {
	key := core.CacheMappingKeyPrefix + id
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err // 缓存未命中或错误
	}

	var mapping core.OrgMapping
	if err := json.Unmarshal([]byte(val), &mapping); err != nil {
		return nil, fmt.Errorf("反序列化组织映射缓存失败: %w", err)
	}

	return &mapping, nil
}

// setCachedMapping 缓存单个组织映射
func (m *mappingManager) setCachedMapping(ctx context.Context, mapping *core.OrgMapping) error {
	key := core.CacheMappingKeyPrefix + mapping.ID
	data, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("序列化组织映射失败: %w", err)
	}

	// 缓存30天（单个映射变更不频繁）
	return m.rdb.SetexCtx(ctx, key, string(data), 30*24*60*60)
}

// deleteCachedMapping 删除单个组织映射缓存
func (m *mappingManager) deleteCachedMapping(ctx context.Context, id string) error {
	key := core.CacheMappingKeyPrefix + id
	_, err := m.rdb.DelCtx(ctx, key)
	return err
}

// getCachedMappingList 从缓存获取组织映射列表（根据租户ID）
func (m *mappingManager) getCachedMappingList(ctx context.Context, tenantID string) ([]*core.OrgMapping, error) {
	key := core.CacheMappingListKeyPrefix + tenantID
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err // 缓存未命中或错误
	}

	var mappings []*core.OrgMapping
	if err := json.Unmarshal([]byte(val), &mappings); err != nil {
		return nil, fmt.Errorf("反序列化组织映射列表缓存失败: %w", err)
	}

	return mappings, nil
}

// setCachedMappingList 缓存组织映射列表
func (m *mappingManager) setCachedMappingList(ctx context.Context, tenantID string, mappings []*core.OrgMapping) error {
	key := core.CacheMappingListKeyPrefix + tenantID
	data, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("序列化组织映射列表失败: %w", err)
	}

	// 缓存1天（列表查询较频繁）
	return m.rdb.SetexCtx(ctx, key, string(data), 24*60*60)
}

// deleteCachedMappingList 删除租户的组织映射列表缓存
func (m *mappingManager) deleteCachedMappingList(ctx context.Context, tenantID string) error {
	key := core.CacheMappingListKeyPrefix + tenantID
	_, err := m.rdb.DelCtx(ctx, key)
	return err
}
