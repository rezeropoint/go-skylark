// Package engine 提供 Skylark 低代码平台 SDK 的核心业务引擎。
//
// go-skylark 是一个用于对接 Skylark 低代码平台的 Go SDK，
// 提供流程管理、事件查询、统计分析等功能。
//
// # 快速开始
//
// 1. 初始化引擎：
//
//	config := &engine.Config{
//	    Cache: cache.Config{},
//	    Query: &query.Config{MaxPageSize: 100},
//	    Stats: &stats.Config{},
//	}
//	db := sqlx.NewMysql("your-db-dsn")
//	redis := redis.New("localhost:6379")
//
//	eng, err := engine.NewSkylarkEngine(config, db, redis)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer eng.Close()
//
// 2. 创建流程：
//
//	err = eng.CreateFlow(ctx, "app", flowID, userID, authHeader, data)
//
// 3. 查询事件数据：
//
//	result, err := eng.QueryEventData(ctx, &core.QueryRequest{
//	    TenantID:      "tenant-001",
//	    EventConfigID: "event-123",
//	    UserOrgIDs:    []string{"org-1", "org-2"},
//	    Page:          1,
//	    PageSize:      20,
//	})
//
// 4. 获取统计分析：
//
//	stats, err := eng.GetStatusStats(ctx, &core.StatsCriteria{
//	    TenantID:       "tenant-001",
//	    EventConfigIDs: []string{"event-123"},
//	})
//
// # 核心功能模块
//
//   - 流程管理（14个方法）：CreateFlow、UpdateFlowJourneyStatus、GetUserAssignments、GetJourneyFullDetail 等
//   - 平台配置（5个方法）：CreatePlatformConfig、GetPlatformConfig、ValidatePlatformConfig 等
//   - 事件配置（5个方法）：CreateEventWithFields、UpdateEventWithFields、ListEventWithFields 等
//   - 组织映射（5个方法）：CreateOrgMapping、GetOrgMapping、UpdateOrgMapping 等
//   - 远程查询（4个方法）：QueryEventData、GetEventDetail、GetFlowList、GetFlowFields
//   - 统计分析（7个方法）：GetDurationStats、GetStatusStats、GetTrendStats、GetNodeStats 等
//
// 详细文档：https://github.com/rezeropoint/go-skylark
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
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - flowID: 流程ID
	//   - userID: 用户ID（发起人）
	//   - authHeader: 认证头（用于调用 Skylark API）
	//   - data: 流程字段数据（键为字段名，值为 TypedValue）
	// 返回:
	//   - error: 错误信息
	CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error
	// CreateFormRow 创建表单行
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - formID: 表单ID
	//   - userID: 用户ID（创建人）
	//   - authHeader: 认证头（用于调用 Skylark API）
	//   - data: 表单字段数据（键为字段名，值为 TypedValue）
	// 返回:
	//   - error: 错误信息
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
	// GetFlowJourneyAssignments 获取流程节点处理信息列表
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - journeyID: 流程记录ID
	// 返回:
	//   - []*core.Assignment: 节点处理信息列表（包含处理人、状态、时间等）
	//   - error: 错误信息
	GetFlowJourneyAssignments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Assignment, error)
	// GetFlowJourneyDetail 获取流程记录详情（包含字段值和附件）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - *core.JourneyDetail: 流程记录详情（包含所有字段值和附件信息）
	//   - error: 错误信息
	GetFlowJourneyDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyDetail, error)
	// GetFlowDetail 获取流程详情（包含字段、节点、边信息）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	// 返回:
	//   - *core.FlowDetail: 流程详情（包含字段定义、节点配置、边关系等）
	//   - error: 错误信息
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
	//   - flowID: 流程ID
	//   - localUserID: 本地用户ID（SDK自动转换为远程用户ID）
	//   - page: 页码（从1开始）
	//   - pageSize: 每页数量
	// 返回:
	//   - []*core.Journey: 流程列表
	//   - int: 总数
	//   - error: 错误信息
	GetProposedJourneys(ctx context.Context, tenantID string, flowID int64, localUserID string, page, pageSize int) ([]*core.Journey, int, error)
	// SearchJourneys 搜索流程记录
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - req: 搜索请求（req.InitiatorID 为本地用户ID，SDK自动转换为远程用户ID）
	// 返回:
	//   - []*core.Journey: 流程列表
	//   - int: 总数
	//   - error: 错误信息
	// 示例:
	//   req := &core.JourneySearchRequest{
	//       FlowID:      123,
	//       Status:      core.StatusProcessing, // 使用状态常量
	//       InitiatorID: "local-user-123",      // 本地用户ID（可选）
	//       Page:        1,
	//       PageSize:    20,
	//   }
	//   journeys, total, err := engine.SearchJourneys(ctx, "tenant-001", req)
	SearchJourneys(ctx context.Context, tenantID string, req *core.JourneySearchRequest) ([]*core.Journey, int, error)
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
	// GetJourneyFullDetail 获取流程完整详情（一站式接口）
	// 功能：
	//   - 聚合基础信息、业务数据、审批历史、待处理节点、节点信息
	//   - 减少前端调用次数（4次 → 1次）
	//   - 自动补充节点名称、处理人姓名
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - *core.JourneyFullDetail: 流程完整详情（包含所有维度信息）
	//   - error: 错误信息
	// 示例:
	//   fullDetail, err := engine.GetJourneyFullDetail(ctx, "tenant-001", 123, 456)
	//   if err != nil {
	//       log.Fatal(err)
	//   }
	//   fmt.Printf("流程编号: %s\n", fullDetail.BasicInfo.SN)
	//   fmt.Printf("审批历史: %d 条\n", len(fullDetail.History))
	//   fmt.Printf("待处理节点: %d 个\n", len(fullDetail.PendingNodes))
	GetJourneyFullDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyFullDetail, error)
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
	// 参数:
	//   - ctx: 上下文
	//   - cfg: 平台配置信息（包含数据库连接、API配置等）
	// 返回:
	//   - string: 配置ID
	//   - error: 错误信息（连接验证失败时返回）
	CreatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) (string, error)
	// GetPlatformConfig 获取平台配置（根据租户ID）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	// 返回:
	//   - *core.PlatformConfig: 平台配置信息
	//   - error: 错误信息（配置不存在时返回）
	GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error)
	// UpdatePlatformConfig 更新平台配置（会验证连接）
	// 参数:
	//   - ctx: 上下文
	//   - cfg: 平台配置信息（必须包含有效的ID）
	// 返回:
	//   - error: 错误信息（连接验证失败或配置不存在时返回）
	UpdatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error
	// DeletePlatformConfig 删除平台配置（软删除）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	// 返回:
	//   - error: 错误信息（配置不存在时返回）
	// 说明:
	//   - 软删除后配置仍保留在数据库中，但标记为已删除
	//   - 删除后相关的远程数据库连接池会被关闭
	DeletePlatformConfig(ctx context.Context, tenantID string) error
	// ValidatePlatformConfig 验证平台连接是否可用
	// 参数:
	//   - ctx: 上下文
	//   - cfg: 平台配置信息（包含数据库连接、API配置等）
	// 返回:
	//   - error: 错误信息（连接失败时返回）
	// 说明:
	//   - 验证数据库连接和 API 连接是否可用
	//   - 不会保存配置，仅用于验证
	ValidatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error

	// 事件配置管理

	// CreateEventWithFields 创建事件配置（包含字段，事务保证原子性）
	// 参数:
	//   - ctx: 上下文
	//   - creation: 事件创建信息（包含事件配置和字段配置列表）
	// 返回:
	//   - string: 事件配置ID
	//   - error: 错误信息
	// 说明:
	//   - 事件配置和字段配置在同一事务中创建，保证原子性
	//   - 会验证远程 flow_id 是否存在
	CreateEventWithFields(ctx context.Context, creation *core.EventCreation) (string, error)
	// UpdateEventWithFields 更新事件配置（包含字段，完整替换策略）
	// 参数:
	//   - ctx: 上下文
	//   - update: 事件更新信息（包含事件配置和字段配置列表）
	// 返回:
	//   - error: 错误信息
	// 说明:
	//   - 字段配置采用完整替换策略：先删除全部字段，再插入新字段
	//   - 事件配置和字段配置在同一事务中更新，保证原子性
	UpdateEventWithFields(ctx context.Context, update *core.EventUpdate) error
	// GetEventWithFields 查询事件配置（包含字段）
	// 参数:
	//   - ctx: 上下文
	//   - id: 事件配置ID
	//   - tenantID: 租户ID（用于权限校验）
	// 返回:
	//   - *core.EventAggregate: 事件聚合对象（包含事件配置和字段配置列表）
	//   - error: 错误信息（事件不存在时返回）
	GetEventWithFields(ctx context.Context, id, tenantID string) (*core.EventAggregate, error)
	// ListEventWithFields 查询事件配置列表（包含字段）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - enabled: 启用状态过滤（nil-全部，true-已启用，false-已禁用）
	// 返回:
	//   - []*core.EventAggregate: 事件配置列表（每个包含事件配置和字段配置）
	//   - error: 错误信息
	ListEventWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error)
	// DeleteEvent 删除事件配置（软删除，字段级联删除）
	// 参数:
	//   - ctx: 上下文
	//   - id: 事件配置ID
	//   - tenantID: 租户ID（用于权限校验）
	// 返回:
	//   - error: 错误信息（事件不存在时返回）
	// 说明:
	//   - 软删除后事件配置仍保留在数据库中，但标记为已删除
	//   - 字段配置会级联删除（数据库外键约束）
	DeleteEvent(ctx context.Context, id, tenantID string) error

	// 组织映射管理

	// CreateOrgMapping 创建组织映射
	// 参数:
	//   - ctx: 上下文
	//   - mapping: 组织映射信息（包含租户ID、远程组织值、本地组织ID等）
	// 返回:
	//   - string: 映射ID
	//   - error: 错误信息（本地组织ID不存在时返回）
	// 说明:
	//   - 组织映射用于将远程数据库中的组织值映射到本地组织ID
	//   - 创建时会验证 local_org_id 是否存在于 organizations 表
	CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) (string, error)
	// GetOrgMapping 获取组织映射（根据ID）
	// 参数:
	//   - ctx: 上下文
	//   - id: 映射ID
	// 返回:
	//   - *core.OrgMapping: 组织映射信息
	//   - error: 错误信息（映射不存在时返回）
	GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error)
	// ListOrgMappings 查询组织映射列表（根据租户ID）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	// 返回:
	//   - []*core.OrgMapping: 组织映射列表
	//   - error: 错误信息
	ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error)
	// UpdateOrgMapping 更新组织映射
	// 参数:
	//   - ctx: 上下文
	//   - mapping: 组织映射信息（必须包含有效的ID）
	// 返回:
	//   - error: 错误信息（映射不存在或本地组织ID不存在时返回）
	UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error
	// DeleteOrgMapping 删除组织映射（硬删除，检查是否被使用）
	// 参数:
	//   - ctx: 上下文
	//   - id: 映射ID
	// 返回:
	//   - error: 错误信息（映射不存在或被事件配置使用时返回）
	// 说明:
	//   - 硬删除会从数据库中彻底删除映射记录
	//   - 删除前会检查是否被事件配置使用，如果被使用则不允许删除
	DeleteOrgMapping(ctx context.Context, id string) error

	// 远程查询

	// QueryEventData 查询事件数据列表（Journey聚合 + 权限过滤）
	// 参数:
	//   - ctx: 上下文
	//   - req: 查询请求（包含租户ID、事件配置ID、组织权限、分页参数、查询条件等）
	// 返回:
	//   - *core.QueryResponse: 查询结果（包含数据列表、总数、分页信息）
	//   - error: 错误信息
	// 说明:
	//   - 使用 DISTINCT ON (slp_journey_id) 进行 Journey 聚合，获取最新 Assignment
	//   - 根据用户组织权限自动过滤数据（通过 OrgMapping 计算可见的远程组织值）
	//   - 支持虚拟状态：pending（只有1个节点）、processing（多个节点）
	QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error)
	// GetEventDetail 获取事件详情（完整流转历史 + 用户名转换）
	// 参数:
	//   - ctx: 上下文
	//   - req: 详情请求（包含租户ID、事件配置ID、journey_id等）
	// 返回:
	//   - *core.DetailResponse: 事件详情（包含完整流转历史和用户名信息）
	//   - error: 错误信息
	// 说明:
	//   - 返回事件的完整流转历史（所有 Assignment 记录）
	//   - 自动将远程用户ID转换为用户名（Redis 缓存优化）
	GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error)
	// GetFlowList 获取远程flows列表（供前端配置）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	// 返回:
	//   - []*core.FlowInfo: 流程列表（包含流程ID、名称、命名空间等）
	//   - error: 错误信息
	// 说明:
	//   - 从远程 Skylark 平台获取流程列表
	//   - 结果会缓存到 Redis（TTL 1小时）
	GetFlowList(ctx context.Context, tenantID string) ([]*core.FlowInfo, error)
	// GetFlowFields 获取远程flow字段列表（供前端配置）
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	// 返回:
	//   - []*core.FieldMetadata: 字段元数据列表（包含字段名、类型、选项等）
	//   - error: 错误信息
	// 说明:
	//   - 从远程 Skylark 平台获取流程的字段定义
	//   - 结果会缓存到 Redis（TTL 1小时）
	GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error)

	// 统计分析

	// GetDurationStats 获取事件处理时长统计（平均/最短/最长）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限等）
	// 返回:
	//   - *core.DurationStats: 时长统计结果（平均时长、最短时长、最长时长）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetDurationStats(ctx context.Context, criteria *core.StatsCriteria) (*core.DurationStats, error)
	// GetStatusStats 获取事件状态统计（各状态数量和占比）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限等）
	// 返回:
	//   - *core.StatusStats: 状态统计结果（各状态的数量和占比）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetStatusStats(ctx context.Context, criteria *core.StatsCriteria) (*core.StatusStats, error)
	// GetTrendStats 获取事件趋势统计（按日/周/月聚合）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限、聚合粒度等）
	// 返回:
	//   - *core.TrendStats: 趋势统计结果（按时间粒度聚合的事件数量）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 支持按日/周/月三种粒度聚合
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetTrendStats(ctx context.Context, criteria *core.StatsCriteria) (*core.TrendStats, error)
	// GetNodeStats 获取节点统计（各节点事件数和平均时长）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限等）
	// 返回:
	//   - *core.NodeStats: 节点统计结果（各节点的事件数量和平均处理时长）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetNodeStats(ctx context.Context, criteria *core.StatsCriteria) (*core.NodeStats, error)
	// GetUserStats 获取处理人统计（Top N处理人排名）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限、Top N数量等）
	// 返回:
	//   - *core.UserStats: 处理人统计结果（Top N处理人的事件数量和平均处理时长）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetUserStats(ctx context.Context, criteria *core.StatsCriteria) (*core.UserStats, error)
	// GetOrgStats 获取组织统计（各组织事件数和平均时长）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、时间范围、组织权限等）
	// 返回:
	//   - *core.OrgStats: 组织统计结果（各组织的事件数量和平均处理时长）
	//   - error: 错误信息
	// 说明:
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
	//   - 结果会缓存到 Redis（TTL 5分钟）
	GetOrgStats(ctx context.Context, criteria *core.StatsCriteria) (*core.OrgStats, error)
	// GetPendingStats 获取待处理事件统计（实时查询，无缓存）
	// 参数:
	//   - ctx: 上下文
	//   - criteria: 统计条件（包含租户ID、事件配置ID列表、组织权限等）
	// 返回:
	//   - *core.PendingStats: 待处理事件统计结果（待处理事件总数和分布）
	//   - error: 错误信息
	// 说明:
	//   - 实时查询，不缓存结果（保证数据实时性）
	//   - 支持多事件ID聚合统计（合并多个事件的统计结果）
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
func NewSkylarkEngine(config Config, db sqlx.SqlConn, redisClient *redis.Redis) (SkylarkEngine, error) {
	return newSkylarkEngine(config, db, redisClient)
}
