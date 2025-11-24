package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// GetPlatformConfig 从缓存获取平台配置
func (f *SkylarkCache) GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	key := core.CachePlatformConfigKeyPrefix + tenantID
	return getJSONWithNullCheck[core.PlatformConfig](
		ctx,
		f.redisClient,
		key,
		nullValueMarker,
		core.ErrPlatformConfigNotFound,
	)
}

// SetPlatformConfig 缓存平台配置
func (f *SkylarkCache) SetPlatformConfig(ctx context.Context, config *core.PlatformConfig, ttl int) error {
	key := core.CachePlatformConfigKeyPrefix + config.TenantID
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化平台配置失败: %w", err)
	}

	return setJSONWithJitter(ctx, f.redisClient, key, data, ttl)
}

// SetPlatformConfigNull 缓存空值标记（用于缓存穿透防护）
func (f *SkylarkCache) SetPlatformConfigNull(ctx context.Context, tenantID string) error {
	key := core.CachePlatformConfigKeyPrefix + tenantID
	return setNullValue(ctx, f.redisClient, key, nullValueMarker, nullValueCacheTTL)
}

// DeletePlatformConfig 删除平台配置缓存
func (f *SkylarkCache) DeletePlatformConfig(ctx context.Context, tenantID string) error {
	key := core.CachePlatformConfigKeyPrefix + tenantID
	_, err := f.redisClient.DelCtx(ctx, key)
	return err
}
