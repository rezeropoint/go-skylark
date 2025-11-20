package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/rezeropoint/go-skylark/v2/core"
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

	if val == "" {
		return "", fmt.Errorf("缓存值为空")
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

// ========== 组织成员缓存 ==========

// GetOrgMembers 从缓存获取组织成员列表
func (c *SkylarkCache) GetOrgMembers(ctx context.Context, tenantID string, remoteOrgID int, withDescendants bool) ([]*core.OrganizationMember, error) {
	key := fmt.Sprintf("%s:%s:%d:%t", core.CacheOrgMembersKeyPrefix, tenantID, remoteOrgID, withDescendants)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var members []*core.OrganizationMember
	if err := json.Unmarshal([]byte(val), &members); err != nil {
		return nil, fmt.Errorf("反序列化组织成员列表缓存失败: %w", err)
	}

	return members, nil
}

// SetOrgMembers 缓存组织成员列表
func (c *SkylarkCache) SetOrgMembers(ctx context.Context, tenantID string, remoteOrgID int, withDescendants bool, members []*core.OrganizationMember, ttl int) error {
	key := fmt.Sprintf("%s:%s:%d:%t", core.CacheOrgMembersKeyPrefix, tenantID, remoteOrgID, withDescendants)
	data, err := json.Marshal(members)
	if err != nil {
		return fmt.Errorf("序列化组织成员列表失败: %w", err)
	}

	return c.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// DeleteOrgMembers 删除组织成员列表缓存
func (c *SkylarkCache) DeleteOrgMembers(ctx context.Context, tenantID string, remoteOrgID int, withDescendants bool) error {
	key := fmt.Sprintf("%s:%s:%d:%t", core.CacheOrgMembersKeyPrefix, tenantID, remoteOrgID, withDescendants)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}

// ========== 组织管理员缓存 ==========

// GetOrgAdministrators 从缓存获取组织管理员列表
func (c *SkylarkCache) GetOrgAdministrators(ctx context.Context, tenantID string, remoteOrgID int) ([]*core.OrganizationAdministrator, error) {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgAdminsKeyPrefix, tenantID, remoteOrgID)
	val, err := c.redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var admins []*core.OrganizationAdministrator
	if err := json.Unmarshal([]byte(val), &admins); err != nil {
		return nil, fmt.Errorf("反序列化组织管理员列表缓存失败: %w", err)
	}

	return admins, nil
}

// SetOrgAdministrators 缓存组织管理员列表
func (c *SkylarkCache) SetOrgAdministrators(ctx context.Context, tenantID string, remoteOrgID int, admins []*core.OrganizationAdministrator, ttl int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgAdminsKeyPrefix, tenantID, remoteOrgID)
	data, err := json.Marshal(admins)
	if err != nil {
		return fmt.Errorf("序列化组织管理员列表失败: %w", err)
	}

	return c.redisClient.SetexCtx(ctx, key, string(data), ttl)
}

// DeleteOrgAdministrators 删除组织管理员列表缓存
func (c *SkylarkCache) DeleteOrgAdministrators(ctx context.Context, tenantID string, remoteOrgID int) error {
	key := fmt.Sprintf("%s:%s:%d", core.CacheOrgAdminsKeyPrefix, tenantID, remoteOrgID)
	_, err := c.redisClient.DelCtx(ctx, key)
	return err
}
