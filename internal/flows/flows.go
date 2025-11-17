package flows

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// SkylarkRegistry 定义流程注册表接口
// 提供创建流程的方法
type SkylarkFlowRegistry interface {
	// CreateFlow 创建并启动一个流程
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - flowID: 流程ID
	//   - userID: 用户ID（远程用户ID）
	//   - authHeader: 认证头信息
	//   - data: 流程数据
	CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error

	// UpdateJourneyStatus 更新流程任务状态
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID（用于获取字段映射）
	//   - journeyID: 流程记录ID
	//   - assignmentID: 任务ID
	//   - localUserID: 本地用户ID（操作人，SDK自动转换为远程用户ID）
	//   - operation: 操作类型（使用 core.OperationApprove 等常量）
	//   - options: 可选参数（评论、下一个节点ID、抄送者、字段数据等）
	UpdateJourneyStatus(ctx context.Context, tenantID string, flowID int64, journeyID int64, assignmentID int64, localUserID string, operation core.JourneyOperation, options core.UpdateJourneyStatusOptions) error

	// GetJourneyBySN 根据流程编号查询流程记录
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - sn: 流程编号
	// 返回:
	//   - *core.Journey: 流程记录信息
	//   - error: 错误信息（如果不存在返回 core.ErrJourneyNotFound）
	GetJourneyBySN(ctx context.Context, tenantID string, flowID int64, sn string) (*core.Journey, error)

	// GetJourneyAssignments 获取流程节点处理信息列表
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - journeyID: 流程记录ID
	// 返回:
	//   - []*core.Assignment: 任务列表
	//   - error: 错误信息
	GetJourneyAssignments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Assignment, error)

	// GetJourneyDetail 获取流程记录详情
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - *core.JourneyDetail: 流程记录详情（包含字段值和附件）
	//   - error: 错误信息（如果不存在返回 core.ErrJourneyNotFound）
	GetJourneyDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyDetail, error)

	// GetFlowDetail 获取流程详情
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	// 返回:
	//   - *core.FlowDetail: 流程详情（包含字段、节点、边信息）
	//   - error: 错误信息（如果不存在返回 core.ErrFlowNotFound）
	GetFlowDetail(ctx context.Context, tenantID string, flowID int64) (*core.FlowDetail, error)

	// GetUserAssignments 获取用户处理的任务列表
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - localUserID: 本地用户ID（SDK内部自动转换为远程用户ID）
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
	//   - localUserID: 本地用户ID（SDK内部自动转换为远程用户ID）
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
	//   - req: 搜索请求（req.InitiatorID 为本地用户ID，SDK内部自动转换为远程用户ID）
	// 返回:
	//   - []*core.Journey: 流程列表
	//   - int: 总数
	//   - error: 错误信息
	SearchJourneys(ctx context.Context, tenantID string, req *core.JourneySearchRequest) ([]*core.Journey, int, error)

	// GetJourneyMoments 获取流程审批历史
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - journeyID: 流程记录ID
	// 返回:
	//   - []*core.Moment: 审批历史列表
	//   - error: 错误信息
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
	GetCurrentProcessingUsers(ctx context.Context, tenantID string, flowID int64, journeyID int64) ([]*core.ProcessingUser, error)

	// AbortJourney 终止流程任务
	// 参数:
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - flowID: 流程ID
	//   - journeyID: 流程记录ID
	// 返回:
	//   - error: 错误信息
	AbortJourney(ctx context.Context, tenantID string, flowID int64, journeyID int64) error
}

// NewSkylarkFlowRegistry 创建新的Skylark流程注册表实例
// 参数:
//   - config: 配置信息
//   - cache: 缓存接口
//   - getPlatformConfig: 获取平台配置的函数（通过依赖注入）
//   - getRemoteUserIDs: 获取远程用户ID的函数（通过依赖注入，用于入参转换）
//   - fillLocalUserIDMap: 批量反向转换函数（通过依赖注入，用于出参转换）
//   - getRemoteDB: 获取远程数据库连接的函数（通过依赖注入，用于性能优化）
func NewSkylarkFlowRegistry(config *Config, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, fillLocalUserIDMap core.FillLocalUserIDMapFunc, getRemoteDB core.GetRemoteDBFunc) (SkylarkFlowRegistry, error) {
	return newSkylarkFlowRegistry(config, cache, getPlatformConfig, getRemoteUserIDs, fillLocalUserIDMap, getRemoteDB)
}
