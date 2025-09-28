package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"

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
}

// NewSkylarkEngine 创建新的Skylark引擎实例
// 参数:
//   - config: 配置信息
//   - redisClient: Redis客户端
func NewSkylarkEngine(config *Config, redisClient *redis.Redis) (SkylarkEngine, error) {
	return newSkylarkEngine(config, redisClient)
}
