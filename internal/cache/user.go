package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rezeropoint/go-skylark/core"
)

// GetUserName 从缓存获取用户名
func (f *SkylarkCache) GetUserName(ctx context.Context, tenantID string, userID string) (string, error) {
	key := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, userID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	return val, nil
}

// SetUserName 缓存用户名
func (f *SkylarkCache) SetUserName(ctx context.Context, tenantID string, userID string, name string, ttl int) error {
	key := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, userID)
	return f.redisClient.SetexCtx(ctx, key, name, ttl)
}

// GetUserIDMapping 从缓存获取用户ID映射（正向：local_user_id -> remote_user_id）
func (c *SkylarkCache) GetUserIDMapping(ctx context.Context, tenantID, localUserID string) (int, error) {
	key := fmt.Sprintf("%s:%s:%s", core.CacheUserIDMappingKeyPrefix, tenantID, localUserID)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return 0, err
	}

	remoteUserID, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("解析远程用户ID失败: %w", err)
	}

	return remoteUserID, nil
}

// SetUserIDMapping 缓存用户ID映射（正向：local_user_id -> remote_user_id）
func (c *SkylarkCache) SetUserIDMapping(ctx context.Context, tenantID, localUserID string, remoteUserID int, ttl int) error {
	key := fmt.Sprintf("%s:%s:%s", core.CacheUserIDMappingKeyPrefix, tenantID, localUserID)
	return c.redisClient.SetexCtx(ctx, key, strconv.Itoa(remoteUserID), ttl)
}

// DeleteUserIDMapping 删除用户ID映射缓存（正向）
func (c *SkylarkCache) DeleteUserIDMapping(ctx context.Context, tenantID, localUserID string) error {
	key := fmt.Sprintf("%s:%s:%s", core.CacheUserIDMappingKeyPrefix, tenantID, localUserID)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}

// GetUserIDMappingReverse 从缓存获取用户ID映射（反向：remote_user_id -> local_user_id）
func (c *SkylarkCache) GetUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int) (string, error) {
	key := fmt.Sprintf("%s:%s:%d", core.CacheUserIDMappingReverseKeyPrefix, tenantID, remoteUserID)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	return val, nil
}

// SetUserIDMappingReverse 缓存用户ID映射（反向：remote_user_id -> local_user_id）
func (c *SkylarkCache) SetUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int, localUserID string, ttl int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheUserIDMappingReverseKeyPrefix, tenantID, remoteUserID)
	return c.redisClient.SetexCtx(ctx, key, localUserID, ttl)
}

// DeleteUserIDMappingReverse 删除用户ID映射缓存（反向）
func (c *SkylarkCache) DeleteUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheUserIDMappingReverseKeyPrefix, tenantID, remoteUserID)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}
