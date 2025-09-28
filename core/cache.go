package core

import "context"

// CacheInterface 定义缓存操作接口
// 用于字段映射的缓存管理
type CacheInterface interface {
	// ClearFieldMappingsCache 清除字段映射缓存
	ClearFieldMappingsCache(ctx context.Context, cacheKey string) error
	// GetFieldMappingsFromCache 从缓存获取字段映射
	GetFieldMappingsFromCache(ctx context.Context, cacheKey string) (map[string]FieldMapping, bool, error)
	// SaveFieldMappingsToCache 保存字段映射到缓存
	SaveFieldMappingsToCache(ctx context.Context, cacheKey string, fieldMappings map[string]FieldMapping) error

	// AcquireLock 获取分布式锁
	// 参数:
	//   - ctx: 上下文
	//   - key: 锁的键
	//   - value: 锁的值，用于确保只有锁的持有者才能释放锁
	//   - expiry: 锁的过期时间（秒）
	// 返回:
	//   - bool: 是否成功获取锁
	//   - error: 错误信息
	AcquireLock(ctx context.Context, key string, value string, expiry int) (bool, error)

	// AcquireLockWithRetry 获取分布式锁，支持重试
	// 参数:
	//   - ctx: 上下文
	//   - key: 锁的键
	//   - value: 锁的值，用于确保只有锁的持有者才能释放锁
	//   - expiry: 锁的过期时间（秒）
	// 返回:
	//   - error: 错误信息，如果获取失败返回 ErrFlowAlreadyProcessing
	AcquireLockWithRetry(ctx context.Context, key string, value string, expiry int) error

	// ReleaseLock 释放分布式锁
	// 参数:
	//   - ctx: 上下文
	//   - key: 锁的键
	//   - value: 锁的值，用于确保只有锁的持有者才能释放锁
	// 返回:
	//   - error: 错误信息
	ReleaseLock(ctx context.Context, key string, value string) error

	// ExtendLock 延长分布式锁的过期时间
	// 参数:
	//   - ctx: 上下文
	//   - key: 锁的键
	//   - value: 锁的值，用于确保只有锁的持有者才能延长锁
	//   - expiry: 新的过期时间（秒）
	// 返回:
	//   - error: 错误信息
	ExtendLock(ctx context.Context, key string, value string, expiry int) error
}
