package admin

import (
	"github.com/rezeropoint/go-skylark/v2/internal/cache"
	"github.com/rezeropoint/go-skylark/v2/internal/platform"
)

// Config 系统管理引擎配置
type Config struct {
	Cache    *cache.Config    // 缓存配置（可选，用于优化性能）
	Platform *platform.Config // 平台配置（可选，用于配置平台连接缓存 TTL 等）
}
