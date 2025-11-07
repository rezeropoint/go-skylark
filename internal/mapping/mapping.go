package mapping

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 组织映射管理器接口
// 职责：管理组织映射配置（将远程业务字段值映射到本地组织ID）
// 特性：平台级别共享，多个事件类型可共享同一映射
// 映射关系：(TenantID + RemoteOrgValue) → LocalOrgID
type Manager interface {
	// CreateOrgMapping 创建组织映射
	// 流程：
	//   1. 验证必填字段（tenant_id, remote_org_value, local_org_id）
	//   2. 查询 organizations 表验证 local_org_id 是否存在（必须是 active 状态且未软删除）
	//   3. 验证唯一约束（tenant_id + remote_org_value）
	//   4. 插入数据库
	// 注意：映射表不存储组织名称
	CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) (string, error)

	// GetOrgMapping 获取组织映射（根据ID查询）
	// 返回 ErrOrgMappingNotFound 如果映射不存在
	// 注意：通过 LEFT JOIN organizations 表获取组织名称
	GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error)

	// ListOrgMappings 查询组织映射列表（根据租户ID）
	// 返回该租户的所有组织映射，按 remote_org_value 排序
	// 注意：通过 LEFT JOIN organizations 表获取组织名称
	ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error)

	// UpdateOrgMapping 更新组织映射
	// 流程：
	//   1. 验证映射存在
	//   2. 不允许修改 tenant_id
	//   3. 如果修改了 local_org_id，重新验证组织是否存在
	//   4. 如果修改了 remote_org_value，验证新的唯一约束
	//   5. 更新数据库
	// 注意：映射表不存储组织名称
	UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error

	// DeleteOrgMapping 删除组织映射（硬删除）
	// 流程：
	//   1. 验证映射存在
	//   2. 检查是否被事件配置使用（查询 event_configs 表）
	//   3. 如果被使用，返回 ErrOrgMappingInUse 并列出使用该映射的事件
	//   4. 硬删除数据库记录
	DeleteOrgMapping(ctx context.Context, id string) error
}

// NewManager 创建组织映射管理器
// 参数：
//   - db: 本地数据库连接（sqlx.SqlConn）
//   - rdb: Redis客户端（go-zero版本，用于缓存）
func NewManager(db sqlx.SqlConn, rdb *redis.Redis) (Manager, error) {
	return newMappingManager(db, rdb)
}
