package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"

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
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	//   - assignmentID: 任务ID
	//   - localUserID: 本地用户ID（操作人，SDK自动转换为远程用户ID）
	//   - operation: 操作类型（使用 core.OperationApprove 等常量）
	//   - options: 可选参数（评论、下一个节点ID、抄送者、字段数据等）
	UpdateFlowJourneyStatus(ctx context.Context, tenantID string, flowID int64, journeyID int64, assignmentID int64, localUserID string, operation core.JourneyOperation, options core.UpdateJourneyStatusOptions) error
	// GetFlowJourneyBySN 根据流程编号查询流程记录
	GetFlowJourneyBySN(ctx context.Context, tenantID string, flowID int64, sn string) (*core.Journey, error)
	// GetFlowJourneyAssignments 获取流程节点处理信息列表
	GetFlowJourneyAssignments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Assignment, error)
	// GetFlowJourneyDetail 获取流程记录详情（包含字段值和附件）
	GetFlowJourneyDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyDetail, error)
	// GetFlowDetail 获取流程详情（包含字段、节点、边信息）
	GetFlowDetail(ctx context.Context, tenantID string, flowID int64) (*core.FlowDetail, error)
	// GetUserAssignments 获取用户处理的任务列表
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - localUserID: 本地用户ID（SDK自动转换为远程用户ID）
	//   - category: 任务类别（使用 core.AssignmentCategoryXXX 常量）
	//   - page: 页码（从1开始）
	//   - pageSize: 每页数量
	// 返回:
	//   - []*core.Assignment: 任务列表（已补充 flow_id 和 flow_title）
	//   - int: 总数
	//   - error: 错误信息
	GetUserAssignments(ctx context.Context, tenantID string, localUserID string, category string, page, pageSize int) ([]*core.Assignment, int, error)
	// GetProposedJourneys 获取用户发起的流程列表
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - localUserID: 本地用户ID（SDK自动转换为远程用户ID）
	//   - page: 页码（从1开始）
	//   - pageSize: 每页数量
	// 返回:
	//   - []*core.Journey: 流程列表
	//   - int: 总数
	//   - error: 错误信息
	GetProposedJourneys(ctx context.Context, tenantID string, localUserID string, page, pageSize int) ([]*core.Journey, int, error)
	// SearchJourneys 搜索流程记录
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - localUserID: 本地用户ID（可选，SDK自动转换为远程用户ID作为发起人筛选条件）
	//   - req: 搜索请求（如果包含 InitiatorID，会被 SDK 覆盖）
	// 返回:
	//   - []*core.Journey: 流程列表
	//   - int: 总数
	//   - error: 错误信息
	// 示例:
	//   status := core.StatusProcessing
	//   req := &core.JourneySearchRequest{
	//       FlowID:   123,
	//       Status:   &status, // 使用状态常量
	//       Page:     1,
	//       PageSize: 20,
	//   }
	//   journeys, total, err := engine.SearchJourneys(ctx, "tenant-001", nil, req)
	SearchJourneys(ctx context.Context, tenantID string, localUserID *string, req *core.JourneySearchRequest) ([]*core.Journey, int, error)
	// GetJourneyMoments 获取流程审批历史
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - journeyID: 流程记录ID
	// 返回:
	//   - []*core.Moment: 审批历史列表（按时间正序排列）
	//   - error: 错误信息
	// 示例:
	//   moments, err := engine.GetJourneyMoments(ctx, "tenant-001", 12345)
	//   for _, moment := range moments {
	//       fmt.Printf("%s %s %s\n", moment.CreatedAt, moment.OperatorName, core.TranslateStatus(moment.Status))
	//   }
	GetJourneyMoments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Moment, error)
	// GetCurrentProcessingUsers 获取当前流程任务的处理者
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - []*core.ProcessingUser: 当前处理人列表
	//   - error: 错误信息
	// 示例:
	//   users, err := engine.GetCurrentProcessingUsers(ctx, "tenant-001", 123, 456)
	//   for _, user := range users {
	//       fmt.Printf("处理人: %s (ID: %d)\n", user.Name, user.ID)
	//   }
	GetCurrentProcessingUsers(ctx context.Context, tenantID string, flowID int64, journeyID int64) ([]*core.ProcessingUser, error)
	// AbortJourney 终止流程任务
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - error: 错误信息
	// 说明:
	//   - 终止操作不可逆，请谨慎使用
	//   - 建议在业务层添加权限校验（只允许发起人或管理员终止）
	// 示例:
	//   err := engine.AbortJourney(ctx, "tenant-001", 123, 456)
	//   if err != nil {
	//       log.Printf("终止流程失败: %v", err)
	//   }
	AbortJourney(ctx context.Context, tenantID string, flowID int64, journeyID int64) error

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
	CreateEventWithFields(ctx context.Context, creation *core.EventCreation) (string, error)
	// UpdateEventWithFields 更新事件配置（包含字段，完整替换策略）
	UpdateEventWithFields(ctx context.Context, update *core.EventUpdate) error
	// GetEventWithFields 查询事件配置（包含字段）
	GetEventWithFields(ctx context.Context, id, tenantID string) (*core.EventAggregate, error)
	// ListEventWithFields 查询事件配置列表（包含字段）
	// enabled 参数：nil-全部，true-已启用，false-已禁用
	ListEventWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error)
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

	// 统计分析

	// GetDurationStats 获取事件处理时长统计（平均/最短/最长）
	GetDurationStats(ctx context.Context, criteria *core.StatsCriteria) (*core.DurationStats, error)
	// GetStatusStats 获取事件状态统计（各状态数量和占比）
	GetStatusStats(ctx context.Context, criteria *core.StatsCriteria) (*core.StatusStats, error)
	// GetTrendStats 获取事件趋势统计（按日/周/月聚合）
	GetTrendStats(ctx context.Context, criteria *core.StatsCriteria) (*core.TrendStats, error)
	// GetNodeStats 获取节点统计（各节点事件数和平均时长）
	GetNodeStats(ctx context.Context, criteria *core.StatsCriteria) (*core.NodeStats, error)
	// GetUserStats 获取处理人统计（Top N处理人排名）
	GetUserStats(ctx context.Context, criteria *core.StatsCriteria) (*core.UserStats, error)
	// GetOrgStats 获取组织统计（各组织事件数和平均时长）
	GetOrgStats(ctx context.Context, criteria *core.StatsCriteria) (*core.OrgStats, error)
	// GetPendingStats 获取待处理事件统计（实时查询，无缓存）
	GetPendingStats(ctx context.Context, criteria *core.StatsCriteria) (*core.PendingStats, error)

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
