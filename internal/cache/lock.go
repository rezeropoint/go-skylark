package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/core"
)

// AcquireLock 获取分布式锁
func (f *SkylarkCache) AcquireLock(ctx context.Context, key string, value string, expiry int) (bool, error) {
	// 使用 SET key value NX PX milliseconds 命令原子性地获取锁
	result, err := f.redisClient.SetnxExCtx(ctx, key, value, expiry)
	if err != nil {
		return false, fmt.Errorf("%w: %v", core.ErrFlowLockAcquireFailed, err)
	}

	return result, nil
}

// ReleaseLock 释放分布式锁
func (f *SkylarkCache) ReleaseLock(ctx context.Context, key string, value string) error {
	// 使用 Lua 脚本确保原子性，只有锁的持有者才能释放锁
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

	if deleted, ok := result.(int64); !ok || deleted == 0 {
		return fmt.Errorf("%w: 锁已被其他进程持有或已过期", core.ErrFlowLockReleaseFailed)
	}

	return nil
}

// ExtendLock 延长分布式锁的过期时间
func (f *SkylarkCache) ExtendLock(ctx context.Context, key string, value string, expiry int) error {
	// 使用 Lua 脚本确保原子性，只有锁的持有者才能延长锁
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

	if extended, ok := result.(int64); !ok || extended == 0 {
		return fmt.Errorf("%w: 锁已被其他进程持有或已过期", core.ErrFlowLockAcquireFailed)
	}

	return nil
}

// AcquireLockWithRetry 获取分布式锁，支持重试
func (f *SkylarkCache) AcquireLockWithRetry(ctx context.Context, key string, value string, expiry int) error {
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

		if i == maxRetries {
			return core.ErrFlowAlreadyProcessing
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			// 继续重试
		}
	}

	return core.ErrFlowAlreadyProcessing
}
