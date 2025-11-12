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

	// 组织映射缓存（mapping 模块 - 业务字段值映射）
	GetOrgMapping(ctx context.Context, id string) (*OrgMapping, error)
	SetOrgMapping(ctx context.Context, mapping *OrgMapping, ttl int) error
	DeleteOrgMapping(ctx context.Context, id string) error
	GetOrgMappingList(ctx context.Context, tenantID string) ([]*OrgMapping, error)
	SetOrgMappingList(ctx context.Context, tenantID string, mappings []*OrgMapping, ttl int) error
	DeleteOrgMappingList(ctx context.Context, tenantID string) error

	// 组织ID映射缓存（organization 模块 - 组织ID双向映射）
	GetOrgIDMapping(ctx context.Context, tenantID, localOrgID string) (int, error)
	SetOrgIDMapping(ctx context.Context, tenantID, localOrgID string, remoteOrgID int, ttl int) error
	DeleteOrgIDMapping(ctx context.Context, tenantID, localOrgID string) error
	GetOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int) (string, error)
	SetOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int, localOrgID string, ttl int) error
	DeleteOrgIDMappingReverse(ctx context.Context, tenantID string, remoteOrgID int) error

	// 用户ID映射缓存（user 模块 - 用户ID双向映射）
	GetUserIDMapping(ctx context.Context, tenantID, localUserID string) (int, error)
	SetUserIDMapping(ctx context.Context, tenantID, localUserID string, remoteUserID int, ttl int) error
	DeleteUserIDMapping(ctx context.Context, tenantID, localUserID string) error
	GetUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int) (string, error)
	SetUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int, localUserID string, ttl int) error
	DeleteUserIDMappingReverse(ctx context.Context, tenantID string, remoteUserID int) error

	// 统计结果缓存（stats 模块）
	GetStats(ctx context.Context, key string) (string, error)
	SetStats(ctx context.Context, key string, jsonData string, ttl int) error

	// 平台配置缓存（platform 模块）
	GetPlatformConfig(ctx context.Context, tenantID string) (*PlatformConfig, error)
	SetPlatformConfig(ctx context.Context, config *PlatformConfig, ttl int) error
	DeletePlatformConfig(ctx context.Context, tenantID string) error

	// 事件配置缓存（event 模块）
	GetEventConfig(ctx context.Context, tenantID, id string) (*EventAggregate, error)
	SetEventConfig(ctx context.Context, config *EventAggregate, ttl int) error
	DeleteEventConfig(ctx context.Context, tenantID, id string) error
	GetEventConfigList(ctx context.Context, tenantID string, enabled *bool) ([]*EventAggregate, error)
	SetEventConfigList(ctx context.Context, tenantID string, enabled *bool, configs []*EventAggregate, ttl int) error
	DeleteEventConfigList(ctx context.Context, tenantID string, enabled *bool) error
}
