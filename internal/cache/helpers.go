package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// 缓存防护常量

// nullValueMarker 空值标记，用于缓存穿透防护
// 当资源不存在时，缓存此标记以避免重复查询不存在的资源
const nullValueMarker = "__NULL__"

// nullValueCacheTTL 空值缓存的 TTL（秒）
// 使用较短的 TTL，避免长期缓存不存在的资源
const nullValueCacheTTL = 300 // 5 分钟

// addJitter 为 TTL 添加随机偏移，防止缓存雪崩
// 说明：在原始 TTL 基础上添加 ±5% 的随机偏移
// 参数：
//   - ttl: 原始 TTL（秒）
//
// 返回：
//   - 添加随机偏移后的 TTL（秒）
//
// 示例：
//   - ttl = 3600  → 返回 3420 ~ 3780（±180秒）
//   - ttl = 300   → 返回 285 ~ 315（±15秒）
func addJitter(ttl int) int {
	if ttl <= 0 {
		return ttl
	}

	// 计算 ±5% 的偏移量
	jitter := ttl / 20 // 5% = 1/20

	// 生成 [-jitter, +jitter] 范围的随机数
	offset := rand.Intn(2*jitter+1) - jitter

	return ttl + offset
}

// getJSONWithNullCheck 通用的 JSON 缓存获取函数，支持空值检测
// 参数：
//   - ctx: 上下文
//   - redisClient: Redis 客户端
//   - key: 缓存键
//   - nullValueMarker: 空值标记（用于检测）
//   - notFoundErr: 当检测到空值标记时返回的错误
// 返回：
//   - *T: 反序列化后的结果
//   - error: 错误信息（缓存未命中返回原始错误，空值标记返回 notFoundErr）
func getJSONWithNullCheck[T any](
	ctx context.Context,
	redisClient *redis.Redis,
	key string,
	nullValueMarker string,
	notFoundErr error,
) (*T, error) {
	val, err := redisClient.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	// 检测空值标记（缓存穿透防护）
	if val == nullValueMarker {
		return nil, notFoundErr
	}

	var result T
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &result, nil
}

// setJSONWithJitter 通用的 JSON 缓存设置函数，自动添加 TTL 抖动
// 参数：
//   - ctx: 上下文
//   - redisClient: Redis 客户端
//   - key: 缓存键
//   - data: 要缓存的数据（JSON 序列化）
//   - ttl: TTL（秒），会自动添加随机偏移
func setJSONWithJitter(
	ctx context.Context,
	redisClient *redis.Redis,
	key string,
	data []byte,
	ttl int,
) error {
	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)
	return redisClient.SetexCtx(ctx, key, string(data), ttlWithJitter)
}

// setNullValue 通用的空值缓存函数
// 参数：
//   - ctx: 上下文
//   - redisClient: Redis 客户端
//   - key: 缓存键
//   - nullValueMarker: 空值标记
//   - ttl: TTL（秒），会自动添加随机偏移
func setNullValue(
	ctx context.Context,
	redisClient *redis.Redis,
	key string,
	nullValueMarker string,
	ttl int,
) error {
	// 空值缓存使用较短的 TTL，并添加随机偏移
	ttlWithJitter := addJitter(ttl)
	return redisClient.SetexCtx(ctx, key, nullValueMarker, ttlWithJitter)
}

// getStringWithNullCheck 通用的字符串缓存获取函数，支持空值检测
// 参数：
//   - ctx: 上下文
//   - redisClient: Redis 客户端
//   - key: 缓存键
//   - nullValueMarker: 空值标记（用于检测）
//   - notFoundErr: 当检测到空值标记时返回的错误
// 返回：
//   - string: 缓存值
//   - error: 错误信息（缓存未命中返回原始错误，空值标记返回 notFoundErr）
func getStringWithNullCheck(
	ctx context.Context,
	redisClient *redis.Redis,
	key string,
	nullValueMarker string,
	notFoundErr error,
) (string, error) {
	val, err := redisClient.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}

	// 检测空值标记（缓存穿透防护）
	if val == nullValueMarker {
		return "", notFoundErr
	}

	return val, nil
}

// setStringWithJitter 通用的字符串缓存设置函数，自动添加 TTL 抖动
// 参数：
//   - ctx: 上下文
//   - redisClient: Redis 客户端
//   - key: 缓存键
//   - value: 要缓存的字符串值
//   - ttl: TTL（秒），会自动添加随机偏移
func setStringWithJitter(
	ctx context.Context,
	redisClient *redis.Redis,
	key string,
	value string,
	ttl int,
) error {
	// 添加随机偏移防止缓存雪崩
	ttlWithJitter := addJitter(ttl)
	return redisClient.SetexCtx(ctx, key, value, ttlWithJitter)
}
