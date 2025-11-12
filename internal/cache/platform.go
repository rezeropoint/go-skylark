package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"
)

// GetPlatformConfig 从缓存获取平台配置
func (f *SkylarkCache) GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	key := core.CachePlatformConfigKeyPrefix + tenantID
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var config core.PlatformConfig
	if err := json.Unmarshal([]byte(val), &config); err != nil {
		return nil, fmt.Errorf("反序列化平台配置缓存失败: %w", err)
	}

	return &config, nil
}

// SetPlatformConfig 缓存平台配置
func (f *SkylarkCache) SetPlatformConfig(ctx context.Context, config *core.PlatformConfig, ttl int) error {
	key := core.CachePlatformConfigKeyPrefix + config.TenantID
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化平台配置失败: %w", err)
	}

	return f.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// DeletePlatformConfig 删除平台配置缓存
func (f *SkylarkCache) DeletePlatformConfig(ctx context.Context, tenantID string) error {
	key := core.CachePlatformConfigKeyPrefix + tenantID
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}
