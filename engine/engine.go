package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/flows"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// SkylarkEngine 定义 Skylark 低代码平台 SDK 的统一引擎接口
// 提供完整的 CRUD 能力：流程创建、表单操作、配置管理、数据查询、统计分析
type SkylarkEngine interface {
	// 流程管理（原有功能）

	// CreateFlow 创建并启动一个流程
	CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error
	// CreateFormRow 创建表单行
	CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error
	// UpdateFlowJourneyStatus 更新流程任务状态
	// operation 支持: approve/refuse/transfer/cancel
	UpdateFlowJourneyStatus(ctx context.Context, app string, flowID int64, journeyID int64, assignmentID int64, userID int64, authHeader string, operation string, options flows.UpdateJourneyStatusOptions) error

	// 平台配置管理

	// CreatePlatformConfig 创建平台配置（会验证连接）
	CreatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) (string, error)
	// GetPlatformConfig 获取平台配置（根据租户ID）
	GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error)
	// UpdatePlatformConfig 更新平台配置（会验证连接）
	UpdatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error
	// DeletePlatformConfig 删除平台配置（软删除）
	DeletePlatformConfig(ctx context.Context, tenantID string) error
	// ValidatePlatformConfig 验证平台连接是否可用
	ValidatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error

	// 事件配置管理

	// CreateEventWithFields 创建事件配置（包含字段，事务保证原子性）
	CreateEventWithFields(ctx context.Context, req *core.CreateEventRequest) (string, error)
	// UpdateEventWithFields 更新事件配置（包含字段，完整替换策略）
	UpdateEventWithFields(ctx context.Context, req *core.UpdateEventRequest) error
	// GetEventWithFields 查询事件配置（包含字段）
	GetEventWithFields(ctx context.Context, id, tenantID string) (*core.EventConfigWithFields, error)
	// ListEventWithFields 查询事件配置列表（包含字段）
	// enabled 参数：nil-全部，true-已启用，false-已禁用
	ListEventWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventConfigWithFields, error)
	// DeleteEvent 删除事件配置（软删除，字段级联删除）
	DeleteEvent(ctx context.Context, id, tenantID string) error

	// 组织映射管理

	// CreateOrgMapping 创建组织映射
	CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) (string, error)
	// GetOrgMapping 获取组织映射（根据ID）
	GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error)
	// ListOrgMappings 查询组织映射列表（根据租户ID）
	ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error)
	// UpdateOrgMapping 更新组织映射
	UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error
	// DeleteOrgMapping 删除组织映射（硬删除，检查是否被使用）
	DeleteOrgMapping(ctx context.Context, id string) error

	// 远程查询

	// QueryEventData 查询事件数据列表（Journey聚合 + 权限过滤）
	QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error)
	// GetEventDetail 获取事件详情（完整流转历史 + 用户名转换）
	GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error)
	// GetFlowList 获取远程flows列表（供前端配置）
	GetFlowList(ctx context.Context, tenantID string) ([]*core.FlowInfo, error)
	// GetFlowFields 获取远程flow字段列表（供前端配置）
	GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error)

	// 组织管理

	// CreateOrganization 创建根组织（无父组织）
	// 流程：调用 Skylark API 创建组织 → 保存映射 → 更新缓存
	CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description string, founderID int) (*core.Organization, error)
	// CreateSubOrganization 创建子组织
	// 流程：查询父组织映射 → 调用 Skylark API → 保存映射 → 更新缓存
	CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description string, founderID int) (*core.Organization, error)
	// DeleteOrganization 删除组织
	// 流程：查询组织映射 → 调用 Skylark API → 删除映射 → 清理缓存
	DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error

	// 用户管理

	// CreateUser 创建Skylark用户
	// 流程：调用 Skylark API 创建用户 → 保存映射 → 更新缓存
	CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid *string) (*core.User, error)

	// GetUser 查询用户（通过本地用户ID）
	// 流程：查询映射关系（缓存优先 → 数据库）→ 返回User对象
	GetUser(ctx context.Context, tenantID, localUserID string) (*core.User, error)

	// 统计分析

	// GetDurationStats 获取事件处理时长统计（平均/最短/最长）
	GetDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error)
	// GetStatusStats 获取事件状态统计（各状态数量和占比）
	GetStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error)
	// GetTrendStats 获取事件趋势统计（按日/周/月聚合）
	GetTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error)
	// GetNodeStats 获取节点统计（各节点事件数和平均时长）
	GetNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error)
	// GetUserStats 获取处理人统计（Top N处理人排名）
	GetUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error)
	// GetOrgStats 获取组织统计（各组织事件数和平均时长）
	GetOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error)
	// GetPendingStats 获取待处理事件统计（实时查询，无缓存）
	GetPendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error)

	// 资源管理

	// Close 关闭引擎，释放资源（尤其是远程数据库连接池）
	Close() error
}

// NewSkylarkEngine 创建新的 Skylark 引擎实例
// 参数:
//   - config: 配置信息（包含 Cache、Query、Stats 配置）
//   - db: 本地数据库连接（用于存储配置数据）
//   - redisClient: Redis 客户端（用于缓存）
func NewSkylarkEngine(config *Config, db sqlx.SqlConn, redisClient *redis.Redis) (SkylarkEngine, error) {
	return newSkylarkEngine(config, db, redisClient)
}
