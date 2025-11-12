package engine

import (
	"github.com/rezeropoint/go-skylark/internal/cache"
	"github.com/rezeropoint/go-skylark/internal/event"
	"github.com/rezeropoint/go-skylark/internal/mapping"
	"github.com/rezeropoint/go-skylark/internal/platform"
	"github.com/rezeropoint/go-skylark/internal/query"
	"github.com/rezeropoint/go-skylark/internal/stats"
)

type Config struct {
	Cache    *cache.Config    // 缓存配置
	Platform *platform.Config // 平台配置
	Event    *event.Config    // 事件配置
	Mapping  *mapping.Config  // 组织映射配置
	Query    *query.Config    // 查询配置
	Stats    *stats.Config    // 统计配置
}
