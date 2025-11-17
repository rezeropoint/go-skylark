package user

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 用户管理器接口
//
// 职责：
//  1. 调用 Skylark REST API 管理远程用户（创建、查询）
//  2. 管理本地用户ID与远程用户ID的双向映射
//  3. 提供缓存支持（Redis 30天 TTL）
//  4. 提供批量查询优化（减少数据库访问）
//
// 设计说明：
//   - 映射管理：本地用户ID（string）↔ Skylark用户ID（int）
//   - 缓存策略：30天TTL，批量查询优化
//   - 错误处理：使用预定义错误（core.ErrUserNotFound等）
//   - 依赖注入：通过GetPlatformConfigFunc获取API配置
type Manager interface {
	// CreateUser 创建Skylark用户
	//
	// 流程：
	//   1. 获取平台配置（APIBaseURL、APIToken）
	//   2. 调用 Skylark API: POST /api/v4/users
	//   3. 保存映射关系到数据库
	//   4. 更新缓存（正向 + 反向）
	//
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID（由调用方生成和管理）
	//   - name: 用户姓名
	//   - identifier: 用户标识符（可选，空字符串表示不提供）
	//   - phone: 手机号（可选，空字符串表示不提供）
	//   - openid: 微信OpenID（可选，空字符串表示不提供）
	//
	// 返回：
	//   - error: 错误信息
	//
	// 错误：
	//   - core.ErrUserCreateFailed: 调用Skylark API失败
	//   - core.ErrUserMappingExists: 映射已存在
	//
	// 说明：
	//   - 成功后映射关系已保存，使用者无需关心远程用户ID
	//   - 远程用户ID由SDK内部管理，对使用者透明
	//   - 可选参数传入空字符串时不会发送给Skylark API
	CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid string) error

	// GetRemoteUserID 查询远程用户ID（单个，内部使用）
	//
	// 流程：
	//   1. 优先从缓存获取
	//   2. 缓存未命中时查询数据库
	//   3. 回写缓存（TTL 30天）
	//
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserID: 本地用户ID
	//
	// 返回：
	//   - int: 远程用户ID
	//   - error: 错误信息
	GetRemoteUserID(ctx context.Context, tenantID, localUserID string) (int, error)

	// GetRemoteUserIDs 批量查询远程用户ID（内部使用，性能优化）
	//
	// 流程：
	//   1. 遍历localUserIDs，优先从缓存获取
	//   2. 收集缓存未命中的localUserIDs
	//   3. 批量查询数据库（使用IN子句）
	//   4. 回写缓存
	//   5. 返回合并结果
	//
	// 性能优势：
	//   - 减少数据库查询次数（批量查询）
	//   - 提高缓存命中率（30天TTL）
	//
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - localUserIDs: 本地用户ID列表
	//
	// 返回：
	//   - []int: 远程用户ID列表（顺序与输入一致）
	//   - error: 错误信息
	GetRemoteUserIDs(ctx context.Context, tenantID string, localUserIDs []string) ([]int, error)

	// GetLocalUserID 反向查询本地用户ID（内部使用）
	//
	// 流程：
	//   1. 优先从反向缓存获取
	//   2. 缓存未命中时查询数据库
	//   3. 回写反向缓存（TTL 30天）
	//
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - remoteUserID: 远程用户ID
	//
	// 返回：
	//   - string: 本地用户ID
	//   - error: 错误信息
	GetLocalUserID(ctx context.Context, tenantID string, remoteUserID int) (string, error)

	// FillLocalUserIDMap 批量反向转换远程用户ID为本地用户ID（填充映射）
	//
	// 流程：
	//   1. 从 userIDMapping 的 keys 提取需要查询的远程用户ID列表
	//   2. 遍历ID列表，优先从反向缓存获取
	//   3. 收集缓存未命中的remoteUserIDs
	//   4. 批量查询数据库（使用IN子句）
	//   5. 检查是否所有ID都有映射（无映射时返回错误）
	//   6. 异步回写反向缓存
	//   7. 填充传入的 userIDMapping
	//
	// 性能优势：
	//   - 减少数据库查询次数（批量查询）
	//   - 提高缓存命中率（30天TTL）
	//   - 直接从 map keys 获取查询列表，减少参数传递
	//
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - userIDMapping: 用户ID映射（指针，keys 为需要查询的远程ID，函数会填充 value 为本地ID）
	//
	// 返回：
	//   - error: 如果任一远程ID无映射，返回 core.ErrUserMappingNotFound
	FillLocalUserIDMap(ctx context.Context, tenantID string, userIDMapping *map[int]string) error
}

// NewManager 创建用户管理器
//
// 参数：
//   - config: 用户管理器配置
//   - db: 本地数据库连接（sqlx.SqlConn，用于存储用户ID映射）
//   - cache: 缓存接口（用于缓存用户ID映射，TTL 30天）
//   - getPlatformConfig: 获取平台配置的函数（用于获取 API BaseURL 和 Token）
//
// 返回：
//   - Manager: 用户管理器实例
//   - error: 错误信息（如数据库表初始化失败）
func NewManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc) (Manager, error) {
	return newUserManager(config, db, cache, getPlatformConfig)
}
