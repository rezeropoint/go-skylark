package event

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 事件+字段配置管理器接口
// 职责：统一管理事件配置和字段配置，提供聚合接口
// 说明：事件和字段作为一个整体管理，一次提交事件+字段，事务保证原子性
type Manager interface {
	// CreateWithFields 创建事件配置（包含字段）
	// 使用数据库事务保证原子性：要么全成功，要么全失败
	// 流程：
	//   1. 验证租户权限
	//   2. 验证 flow_id 是否存在（调用远程数据库）
	//   3. 插入事件配置，获取生成的ID
	//   4. 批量插入字段配置
	//   5. 可选：验证字段名是否在远程表存在
	// 返回：创建后的事件配置（包含生成的ID）
	CreateWithFields(ctx context.Context, creation *core.EventCreation) (string, error)

	// UpdateWithFields 更新事件配置（包含字段）
	// 采用完整替换策略：先删除所有旧字段，再插入新字段（避免复杂的diff逻辑）
	// 使用数据库事务保证原子性
	// 流程：
	//   1. 验证事件配置存在且有权限
	//   2. 更新事件配置基本信息
	//   3. 删除所有旧字段配置
	//   4. 批量插入新字段配置
	UpdateWithFields(ctx context.Context, update *core.EventUpdate) error

	// GetWithFields 查询事件配置（包含字段）
	// 返回事件配置和关联的所有字段（按display_order排序）
	GetWithFields(ctx context.Context, id, tenantID string) (*core.EventAggregate, error)

	// ListWithFields 查询事件配置列表（包含字段）
	// 返回所有匹配的事件配置和关联的字段
	// enabled参数：nil-全部，true-已启用，false-已禁用
	// 优化：使用批量查询，一次SQL查询所有字段
	ListWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error)

	// Delete 删除事件配置（软删除）
	// 字段配置会通过数据库外键级联删除（ON DELETE CASCADE）
	Delete(ctx context.Context, id, tenantID string) error
}

// NewManager 创建事件+字段配置管理器
// 参数：
//   - db: 本地数据库连接（sqlx.SqlConn）
//   - getRemoteDB: 获取远程数据库连接的函数（用于验证flow_id、字段名等）
func NewManager(db sqlx.SqlConn, getRemoteDB core.GetRemoteDBFunc) (Manager, error) {
	return newEventManager(db, getRemoteDB)
}
