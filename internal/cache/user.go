package cache

import (
	"context"
	"fmt"

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
