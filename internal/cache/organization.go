package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rezeropoint/go-skylark/core"
)

// GetOrgIDMapping 从缓存获取组织ID映射（正向：local_org_id -> remote_org_id）
func (c *SkylarkCache) GetOrgIDMapping(ctx context.Context, tenantID, localOrgID string) (int, error) {
	key := fmt.Sprintf("%s:%s:%s", core.CacheOrgIDMappingKeyPrefix, tenantID, localOrgID)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return 0, err
	}

	remoteOrgID, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("解析远程组织ID失败: %w", err)
	}

	return remoteOrgID, nil
}

// SetOrgIDMapping 缓存组织ID映射（正向：local_org_id -> remote_org_id）
func (c *SkylarkCache) SetOrgIDMapping(ctx context.Context, tenantID, localOrgID string, remoteOrgID int, ttl int) error {
	key := fmt.Sprintf("%s:%s:%s", core.CacheOrgIDMappingKeyPrefix, tenantID, localOrgID)
	return c.redisClient.SetexCtx(ctx, key, strconv.Itoa(remoteOrgID), ttl)
}

// DeleteOrgIDMapping 删除组织ID映射缓存（正向）
func (c *SkylarkCache) DeleteOrgIDMapping(ctx context.Context, tenantID, localOrgID string) error {
	key := fmt.Sprintf("%s:%s:%s", core.CacheOrgIDMappingKeyPrefix, tenantID, localOrgID)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}

// GetOrgIDMappingReverse 从缓存获取组织ID映射（反向：remote_org_id -> local_org_id）
func (c *SkylarkCache) GetOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int) (string, error) {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgIDMappingReverseKeyPrefix, tenantID, remoteOrgID)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	return val, nil
}

// SetOrgIDMappingReverse 缓存组织ID映射（反向：remote_org_id -> local_org_id）
func (c *SkylarkCache) SetOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int, localOrgID string, ttl int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgIDMappingReverseKeyPrefix, tenantID, remoteOrgID)
	return c.redisClient.SetexCtx(ctx, key, localOrgID, ttl)
}

// DeleteOrgIDMappingReverse 删除组织ID映射缓存（反向）
func (c *SkylarkCache) DeleteOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgIDMappingReverseKeyPrefix, tenantID, remoteOrgID)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}
