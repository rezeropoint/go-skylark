package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/flows"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// SkylarkRegistry 定义流程注册表接口
// 提供创建流程的方法
type SkylarkEngine interface {
	// CreateFlow 创建并启动一个流程
	// 参数:
	//   - ctx: 上下文
	//   - app: 应用名称
	//   - flowID: 流程ID
	//   - userID: 用户ID
	//   - authHeader: 认证头信息
	//   - data: 流程数据
	CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error
	CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error

	// UpdateFlowJourneyStatus 更新流程任务状态
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
	UpdateFlowJourneyStatus(ctx context.Context, app string, flowID int64, journeyID int64, assignmentID int64, userID int64, authHeader string, operation string, options flows.UpdateJourneyStatusOptions) error
}

// NewSkylarkEngine 创建新的Skylark引擎实例
// 参数:
//   - config: 配置信息
//   - redisClient: Redis客户端
func NewSkylarkEngine(config *Config, redisClient *redis.Redis) (SkylarkEngine, error) {
	return newSkylarkEngine(config, redisClient)
}
