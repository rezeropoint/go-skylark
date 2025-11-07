package cache

import (
	"context"
)

// GetStats 从缓存获取统计结果（JSON 字符串）
func (f *SkylarkCache) GetStats(ctx context.Context, key string) (string, error) {
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	return val, nil
}

// SetStats 缓存统计结果（JSON 字符串）
func (f *SkylarkCache) SetStats(ctx context.Context, key string, jsonData string, ttl int) error {
	return f.redisClient.SetexCtx(ctx, key, jsonData, ttl)
}
