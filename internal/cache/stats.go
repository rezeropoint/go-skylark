package cache

import (
	"context"
	"fmt"
)

// GetStats 从缓存获取统计结果（JSON 字符串）
func (f *SkylarkCache) GetStats(ctx context.Context, key string) (string, error) {
	val, err := f.redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	if val == "" {
		return "", fmt.Errorf("缓存值为空")
	}

	return val, nil
}

// SetStats 缓存统计结果（JSON 字符串）
func (f *SkylarkCache) SetStats(ctx context.Context, key string, jsonData string, ttl int) error {
	return setStringWithJitter(ctx, f.redisClient, key, jsonData, ttl)
}
