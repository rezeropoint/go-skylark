// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：缓存接口定义
// 说明：统一的缓存抽象层，支持字段映射、分布式锁、查询结果等多种缓存场景
package core

import "context"

// CacheInterface 缓存操作接口
type CacheInterface interface {
	// 字段映射缓存（flows/forms 模块）
	ClearFieldMappingsCache(ctx context.Context, cacheKey string) error
	GetFieldMappingsFromCache(ctx context.Context, cacheKey string) (map[string]FieldMapping, bool, error)
	SaveFieldMappingsToCache(ctx context.Context, cacheKey string, fieldMappings map[string]FieldMapping) error

	// 分布式锁
	AcquireLock(ctx context.Context, key string, value string, expiry int) (bool, error)
	AcquireLockWithRetry(ctx context.Context, key string, value string, expiry int) error
	ReleaseLock(ctx context.Context, key string, value string) error
	ExtendLock(ctx context.Context, key string, value string, expiry int) error

	// Flow 缓存（query 模块）
	GetFlowList(ctx context.Context, tenantID string, namespaceID int) ([]*FlowInfo, error)
	SetFlowList(ctx context.Context, tenantID string, namespaceID int, flows []*FlowInfo, ttl int) error
	GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*FieldMetadata, error)
	SetFlowFields(ctx context.Context, tenantID string, flowID int, fields []*FieldMetadata, ttl int) error

	// 用户名缓存（query/stats 模块）
	GetUserName(ctx context.Context, tenantID string, userID string) (string, error)
	SetUserName(ctx context.Context, tenantID string, userID string, name string, ttl int) error

	// 组织映射缓存（mapping 模块）
	GetOrgMapping(ctx context.Context, id string) (*OrgMapping, error)
	SetOrgMapping(ctx context.Context, mapping *OrgMapping, ttl int) error
	DeleteOrgMapping(ctx context.Context, id string) error
	GetOrgMappingList(ctx context.Context, tenantID string) ([]*OrgMapping, error)
	SetOrgMappingList(ctx context.Context, tenantID string, mappings []*OrgMapping, ttl int) error
	DeleteOrgMappingList(ctx context.Context, tenantID string) error

	// 统计结果缓存（stats 模块）
	GetStats(ctx context.Context, key string) (string, error)
	SetStats(ctx context.Context, key string, jsonData string, ttl int) error
}
