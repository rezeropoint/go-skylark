package flows

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
)

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
}

// NewSkylarkFlowRegistry 创建新的Skylark流程注册表实例
// 参数:
//   - config: 配置信息
//   - cache: 缓存接口
func NewSkylarkFlowRegistry(config *Config, cache core.CacheInterface) (SkylarkFlowRegistry, error) {
	return newSkylarkFlowRegistry(config, cache)
}
