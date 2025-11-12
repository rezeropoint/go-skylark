package platform

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 平台配置管理器接口
// 职责：管理 Skylark 平台对接配置（PostgreSQL存储）
// 约束：一个租户只能配置一个 Skylark 平台（tenant_id UNIQUE）
type Manager interface {
	// Create 创建平台配置
	// 注意：创建前会验证连接是否可用，验证失败则创建失败
	Create(ctx context.Context, cfg *core.PlatformConfig) (string, error)

	// Get 获取平台配置（根据租户ID查询）
	// 返回 ErrPlatformConfigNotFound 如果配置不存在
	Get(ctx context.Context, tenantID string) (*core.PlatformConfig, error)

	// GetAPIConfig 获取Skylark API调用配置（供flows、forms等Manager使用）
	// 该方法实现了 core.GetPlatformConfigFunc 函数签名，用于依赖注入
	// 流程：
	//   1. 根据 tenantID 查询平台配置
	//   2. 验证 EnableAPI 是否开启
	//   3. 验证 APIBaseURL 和 APIToken 是否配置
	//   4. 返回轻量级的 SkylarkAPIConfig（不包含敏感数据库信息）
	// 返回：
	//   - *core.SkylarkAPIConfig: API调用配置（只包含App和Token）
	//   - error: ErrPlatformConfigNotFound / ErrAPINotEnabled / ErrInvalidPlatformConfig
	GetAPIConfig(ctx context.Context, tenantID string) (*core.SkylarkAPIConfig, error)

	// Update 更新平台配置
	// 注意：更新前会验证连接是否可用，验证失败则更新失败
	Update(ctx context.Context, cfg *core.PlatformConfig) error

	// Delete 删除平台配置（软删除）
	// 注意：如果平台配置被事件配置使用，则删除失败
	Delete(ctx context.Context, tenantID string) error

	// Validate 验证平台连接是否可用
	// 测试方式：连接远程数据库并查询 flows 表
	Validate(ctx context.Context, cfg *core.PlatformConfig) error

	// GetRemoteDB 获取远程 Skylark 数据库连接（供其他 Manager 使用）
	// 该方法实现了 core.GetRemoteDBFunc 函数签名，用于依赖注入
	// 流程：
	//   1. 根据 tenantID 查询平台配置（本地数据库）
	//   2. 使用配置信息创建远程连接（如果连接池中不存在）
	//   3. 返回远程连接（复用连接池中的连接）
	// 返回：
	//   - sqlx.SqlConn: 远程数据库连接
	//   - error: 如果平台配置不存在或连接失败
	GetRemoteDB(ctx context.Context, tenantID string) (sqlx.SqlConn, error)

	// Close 关闭管理器（释放资源，关闭所有远程连接）
	Close() error
}

// NewManager 创建平台配置管理器
// 参数：
//   - db: 本地数据库连接（sqlx.SqlConn）
func NewManager(config Config, db sqlx.SqlConn) (Manager, error) {
	return newPlatformManager(config, db)
}
