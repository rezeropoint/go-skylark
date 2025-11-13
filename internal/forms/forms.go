package forms

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// SkylarkRegistry 定义流程注册表接口
// 提供创建流程的方法
type SkylarkFormRegistry interface {
	CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error
}

// NewSkylarkFormRegistry 创建新的Skylark表单注册表实例
// 参数:
//   - config: 配置信息
//   - cache: 缓存接口
func NewSkylarkFormRegistry(config *Config, cache core.CacheInterface) (SkylarkFormRegistry, error) {
	return newSkylarkFormRegistry(config, cache)
}
