package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type SkylarkCache struct {
	redisClient *redis.Redis
	config      *Config
}

func NewSkylarkCache(redisClient *redis.Redis, config *Config) *SkylarkCache {
	return &SkylarkCache{
		redisClient: redisClient,
		config:      config,
	}
}

// SaveFieldMappingsToCache 保存字段映射到缓存
func (f *SkylarkCache) SaveFieldMappingsToCache(ctx context.Context, cacheKey string, fieldMappings map[string]core.FieldMapping) error {
	// 序列化字段映射
	data, err := json.Marshal(fieldMappings)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	// 将数据保存到Redis，设置过期时间
	err = f.redisClient.SetexCtx(ctx, cacheKey, string(data), f.config.FieldMappingsCacheExpiry)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	return nil
}

// GetFieldMappingsFromCache 从缓存获取字段映射
// 返回字段映射、是否找到以及可能的错误
func (f *SkylarkCache) GetFieldMappingsFromCache(ctx context.Context, cacheKey string) (map[string]core.FieldMapping, bool, error) {
	// 从Redis获取缓存数据
	data, err := f.redisClient.GetCtx(ctx, cacheKey)
	if err != nil {
		// 如果是key不存在的错误，返回缓存未命中
		if err.Error() == "redis: nil" {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}

	// 如果数据为空，表示缓存未命中
	if data == "" {
		return nil, false, nil
	}

	// 反序列化数据
	var fieldMappings map[string]core.FieldMapping
	err = json.Unmarshal([]byte(data), &fieldMappings)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	return fieldMappings, true, nil
}

// ClearFieldMappingsCache 清除指定的字段映射缓存
func (f *SkylarkCache) ClearFieldMappingsCache(ctx context.Context, cacheKey string) error {
	_, err := f.redisClient.DelCtx(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrCacheOperationFailed, err)
	}
	return nil
}

// AcquireLock 获取分布式锁
func (f *SkylarkCache) AcquireLock(ctx context.Context, key string, value string, expiry int) (bool, error) {
	// 使用 SET key value NX PX milliseconds 命令原子性地获取锁
	// NX: 只在key不存在时设置
	// PX: 设置过期时间（毫秒）
	result, err := f.redisClient.SetnxExCtx(ctx, key, value, expiry)
	if err != nil {
		return false, fmt.Errorf("%w: %v", core.ErrFlowLockAcquireFailed, err)
	}

	return result, nil
}

// ReleaseLock 释放分布式锁
func (f *SkylarkCache) ReleaseLock(ctx context.Context, key string, value string) error {
	// 使用 Lua 脚本确保原子性，只有锁的持有者才能释放锁
	// 脚本逻辑：检查 key 的值是否等于传入的 value，如果是则删除 key
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := f.redisClient.EvalCtx(ctx, luaScript, []string{key}, value)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrFlowLockReleaseFailed, err)
	}

	// 检查是否成功释放锁
	if deleted, ok := result.(int64); !ok || deleted == 0 {
		return fmt.Errorf("%w: 锁已被其他进程持有或已过期", core.ErrFlowLockReleaseFailed)
	}

	return nil
}

// ExtendLock 延长分布式锁的过期时间
func (f *SkylarkCache) ExtendLock(ctx context.Context, key string, value string, expiry int) error {
	// 使用 Lua 脚本确保原子性，只有锁的持有者才能延长锁
	// 脚本逻辑：检查 key 的值是否等于传入的 value，如果是则设置新的过期时间
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := f.redisClient.EvalCtx(ctx, luaScript, []string{key}, value, fmt.Sprintf("%d", expiry*1000))
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrFlowLockAcquireFailed, err)
	}

	// 检查是否成功延长锁
	if extended, ok := result.(int64); !ok || extended == 0 {
		return fmt.Errorf("%w: 锁已被其他进程持有或已过期", core.ErrFlowLockAcquireFailed)
	}

	return nil
}

// AcquireLockWithRetry 获取分布式锁，支持重试
func (f *SkylarkCache) AcquireLockWithRetry(ctx context.Context, key string, value string, expiry int) error {
	// 从配置获取重试参数，如果未配置则使用默认值
	maxRetries := f.config.FlowLockRetryCount
	if maxRetries <= 0 {
		maxRetries = 3 // 默认重试3次
	}

	retryInterval := time.Duration(f.config.FlowLockRetryInterval) * time.Millisecond
	if retryInterval <= 0 {
		retryInterval = 500 * time.Millisecond // 默认500ms间隔
	}

	for i := 0; i <= maxRetries; i++ {
		acquired, err := f.AcquireLock(ctx, key, value, expiry)
		if err != nil {
			return fmt.Errorf("%w: %v", core.ErrFlowLockAcquireFailed, err)
		}
		if acquired {
			return nil // 成功获取锁
		}

		// 如果是最后一次重试，直接返回错误
		if i == maxRetries {
			return core.ErrFlowAlreadyProcessing
		}

		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			// 继续重试
		}
	}

	return core.ErrFlowAlreadyProcessing
}
