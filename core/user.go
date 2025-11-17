// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：用户管理
// 说明：本文件定义的类型用于管理 Skylark 用户及其映射关系
package core

import (
	"context"
	"time"
)

// 用户ID映射缓存键定义
const (
	// CacheUserIDMappingKeyPrefix 用户ID映射缓存键前缀（正向映射：local → remote）
	// 格式：skylark:user_mapping:{tenantID}:{localUserID}
	// 值：remoteUserID (int)
	// TTL: 30天
	CacheUserIDMappingKeyPrefix = "skylark:user_mapping"

	// CacheUserIDMappingReverseKeyPrefix 用户ID映射反向缓存键前缀（反向映射：remote → local）
	// 格式：skylark:user_mapping_rev:{tenantID}:{remoteUserID}
	// 值：localUserID (string)
	// TTL: 30天
	CacheUserIDMappingReverseKeyPrefix = "skylark:user_mapping_rev"
)

// User Skylark 用户领域模型
// 说明：只存储 Skylark 用户ID（纯 Go 类型，无框架依赖）
// 注意：
//   - 不存储 name、phone、identifier 等本地已有的信息
//   - 不存储组织关系（本地通过 system_user_org_relations 表管理）
//   - 这个结构体只用于返回 Skylark 的用户ID
type User struct {
	ID int // Skylark 用户ID（整数）
}

// UserIDMapping 用户ID映射领域模型
// 说明：管理本地用户ID与 Skylark 用户ID的映射关系
// 用途：支持双向查询（local_user_id ↔ remote_user_id）
type UserIDMapping struct {
	ID           string    // UUID
	TenantID     string    // 租户ID
	LocalUserID  string    // 本地用户ID（由调用方的 GO 项目生成和管理）
	RemoteUserID int       // Skylark 用户ID（整数，调用 Skylark API 创建后获得）
	CreatedAt    time.Time // 创建时间
	UpdatedAt    time.Time // 更新时间
}

// GetRemoteUserIDsFunc 批量查询远程用户ID的函数类型
// 用途：供其他 Manager 根据本地用户ID批量查询 Skylark 用户ID
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - localUserIDs: 本地用户ID列表
//
// 返回：
//   - []int: Skylark 用户ID列表
//   - error: 错误信息（如映射不存在）
//
// 说明：
//   - 该函数会优先查询缓存，缓存未命中时查询数据库
//   - 用于批量操作场景
//   - 由 User Manager 的 GetRemoteUserIDs 方法实现
type GetRemoteUserIDsFunc func(ctx context.Context, tenantID string, localUserIDs []string) ([]int, error)

// FillLocalUserIDMapFunc 批量反向转换函数类型（填充本地用户ID映射）
// 用途：供其他 Manager 根据远程用户ID批量查询本地用户ID
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - userIDMapping: 用户ID映射（指针，keys 为需要查询的远程ID，函数会填充 value 为本地ID）
//
// 返回：
//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
//
// 说明：
//   - 该函数会优先查询缓存（反向缓存），缓存未命中时批量查询数据库
//   - 直接从 map keys 获取需要查询的远程ID列表，填充 values
//   - 用于出参用户ID转换场景（远程ID → 本地ID）
//   - 由 User Manager 的 FillLocalUserIDMap 方法实现
type FillLocalUserIDMapFunc func(ctx context.Context, tenantID string, userIDMapping *map[int]string) error

// ExtractUserIDsToMap 通用函数：从结构体列表中提取用户ID到映射（用于去重和占位）
// 用途：统一处理各种结构体列表的用户ID提取逻辑，避免重复代码
//
// 参数：
//   - items: 任意类型的切片
//   - extractor: 提取函数，从单个元素中提取用户ID（返回0表示无有效ID）
//
// 返回：
//   - map[int]string: 用户ID映射（keys为远程ID，values为空字符串占位，用于后续填充本地ID）
//
// 说明：
//   - 自动去重（同一个远程ID只会出现一次）
//   - 跳过无效ID（extractor返回0的元素）
//   - values为空字符串占位，后续通过 GetLocalUserIDsWithMapFunc 填充本地ID
//
// 使用示例：
//
//	// 示例1：Journey列表
//	userIDMapping := core.ExtractUserIDsToMap(journeys, func(j *core.Journey) int {
//	    if j.User != nil {
//	        return int(j.User.ID)
//	    }
//	    return 0
//	})
//
//	// 示例2：Assignment列表
//	userIDMapping := core.ExtractUserIDsToMap(assignments, func(a *core.Assignment) int {
//	    return int(a.AssigneeID)
//	})
func ExtractUserIDsToMap[T any](items []T, extractor func(T) int) map[int]string {
	mapping := make(map[int]string)
	for _, item := range items {
		if id := extractor(item); id != 0 {
			mapping[id] = ""
		}
	}
	return mapping
}
