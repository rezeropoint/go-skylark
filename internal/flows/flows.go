package flows

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
)

// UpdateJourneyStatusOptions 更新流程任务状态的可选参数
type UpdateJourneyStatusOptions struct {
	Comment           string                     // 处理意见
	NextVertexID      int                        // 下一个节点ID
	CarbonCopyUserIDs []int                      // 抄送者ID列表
	Data              map[string]core.TypedValue // 字段数据
}

// SkylarkRegistry 定义流程注册表接口
// 提供创建流程的方法
type SkylarkFlowRegistry interface {
	// CreateFlow 创建并启动一个流程
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - flowID: 流程ID
	//   - userID: 用户ID
	//   - authHeader: 认证头信息
	//   - data: 流程数据
	CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error

	// UpdateJourneyStatus 更新流程任务状态
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - flowID: 流程ID（用于获取字段映射）
	//   - journeyID: 流程记录ID
	//   - assignmentID: 任务ID
	//   - userID: 用户ID
	//   - authHeader: 认证头信息
	//   - operation: 操作类型 (approve/refuse/transfer/cancel)
	//   - options: 可选参数（评论、下一个节点ID、抄送者、时限、字段数据等）
	UpdateJourneyStatus(ctx context.Context, app string, flowID int64, journeyID int64, assignmentID int64, userID int64, authHeader string, operation string, options UpdateJourneyStatusOptions) error
}

// NewSkylarkFlowRegistry 创建新的Skylark流程注册表实例
// 参数:
//   - config: 配置信息
//   - cache: 缓存接口
func NewSkylarkFlowRegistry(config *Config, cache core.CacheInterface) (SkylarkFlowRegistry, error) {
	return newSkylarkFlowRegistry(config, cache)
}
