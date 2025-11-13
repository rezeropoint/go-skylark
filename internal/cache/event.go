package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// GetEventConfig 从缓存获取事件配置（含字段）
func (f *SkylarkCache) GetEventConfig(ctx context.Context, tenantID, id string) (*core.EventAggregate, error) {
	key := fmt.Sprintf("%s%s:%s", core.CacheEventConfigKeyPrefix, tenantID, id)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var config core.EventAggregate
	if err := json.Unmarshal([]byte(val), &config); err != nil {
		return nil, fmt.Errorf("反序列化事件配置缓存失败: %w", err)
	}

	return &config, nil
}

// SetEventConfig 缓存事件配置（含字段）
func (f *SkylarkCache) SetEventConfig(ctx context.Context, config *core.EventAggregate, ttl int) error {
	key := fmt.Sprintf("%s%s:%s", core.CacheEventConfigKeyPrefix, config.EventConfig.TenantID, config.EventConfig.ID)
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化事件配置失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// DeleteEventConfig 删除事件配置缓存
func (f *SkylarkCache) DeleteEventConfig(ctx context.Context, tenantID, id string) error {
	key := fmt.Sprintf("%s%s:%s", core.CacheEventConfigKeyPrefix, tenantID, id)
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}

// GetEventConfigList 从缓存获取事件配置列表
func (f *SkylarkCache) GetEventConfigList(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error) {
	enabledStr := "all"
	if enabled != nil {
		if *enabled {
			enabledStr = "true"
		} else {
			enabledStr = "false"
		}
	}
	key := fmt.Sprintf("%s%s:enabled=%s", core.CacheEventConfigListKeyPrefix, tenantID, enabledStr)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var configs []*core.EventAggregate
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		return nil, fmt.Errorf("反序列化事件配置列表缓存失败: %w", err)
	}

	return configs, nil
}

// SetEventConfigList 缓存事件配置列表
func (f *SkylarkCache) SetEventConfigList(ctx context.Context, tenantID string, enabled *bool, configs []*core.EventAggregate, ttl int) error {
	enabledStr := "all"
	if enabled != nil {
		if *enabled {
			enabledStr = "true"
		} else {
			enabledStr = "false"
		}
	}
	key := fmt.Sprintf("%s%s:enabled=%s", core.CacheEventConfigListKeyPrefix, tenantID, enabledStr)
	data, err := json.Marshal(configs)
	if err != nil {
		return fmt.Errorf("序列化事件配置列表失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// DeleteEventConfigList 删除事件配置列表缓存
func (f *SkylarkCache) DeleteEventConfigList(ctx context.Context, tenantID string, enabled *bool) error {
	enabledStr := "all"
	if enabled != nil {
		if *enabled {
			enabledStr = "true"
		} else {
			enabledStr = "false"
		}
	}
	key := fmt.Sprintf("%s%s:enabled=%s", core.CacheEventConfigListKeyPrefix, tenantID, enabledStr)
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}
