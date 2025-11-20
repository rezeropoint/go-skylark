package cache

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// GetUserName 从缓存获取用户名
func (f *SkylarkCache) GetUserName(ctx context.Context, tenantID string, userID string) (string, error) {
	key := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, userID)
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	if val == "" {
		return "", fmt.Errorf("缓存值为空")
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

	if val == "" {
		return "", fmt.Errorf("缓存值为空")
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

// BatchGetUserNames 批量查询用户名（优先从缓存获取）
// 说明：
//   - 先从 Redis 缓存获取
//   - 缓存未命中时查询远程数据库 users 表
//   - 自动回写缓存
//
// 参数：
//   - ctx: 上下文
//   - remoteDB: 远程数据库连接（Skylark PostgreSQL）
//   - tenantID: 租户ID
//   - userIDs: 用户ID列表
//   - ttl: 缓存TTL（秒）
//
// 返回：
//   - map[userID]userName: 用户名映射表
//   - error: 错误信息
func (f *SkylarkCache) BatchGetUserNames(
	ctx context.Context,
	remoteDB sqlx.SqlConn,
	tenantID string,
	userIDs []string,
	ttl int,
) (map[string]string, error) {
	if len(userIDs) == 0 {
		return make(map[string]string), nil
	}

	userNames := make(map[string]string, len(userIDs))
	uncachedIDs := []string{}

	// 1. 尝试从缓存获取
	for _, userID := range userIDs {
		val, err := f.GetUserName(ctx, tenantID, userID)
		if err == nil && val != "" {
			userNames[userID] = val
			f.metrics.UserNameHit.Add(1) // 缓存命中
		} else {
			uncachedIDs = append(uncachedIDs, userID)
			f.metrics.UserNameMiss.Add(1) // 缓存未命中
		}
	}

	// 2. 查询未命中的用户名（从远程 users 表）
	if len(uncachedIDs) > 0 {
		type userRow struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}

		query := "SELECT id, name FROM users WHERE id = ANY($1)"
		var users []*userRow
		err := remoteDB.QueryRowsCtx(ctx, &users, query, pq.Array(uncachedIDs))
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("批量查询用户名失败: %w", err)
		}

		// 添加随机偏移防止缓存雪崩
		ttlWithJitter := addJitter(ttl)

		for _, user := range users {
			userNames[user.ID] = user.Name

			// 3. 写入缓存
			_ = f.SetUserName(ctx, tenantID, user.ID, user.Name, ttlWithJitter)
		}
	}

	return userNames, nil
}
