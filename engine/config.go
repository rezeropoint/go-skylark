package engine

import (
	"github.com/rezeropoint/go-skylark/internal/cache"
	"github.com/rezeropoint/go-skylark/internal/query"
	"github.com/rezeropoint/go-skylark/internal/stats"
)

type Config struct {
	Cache *cache.Config // 缓存配置
	Query *query.Config // 查询配置
	Stats *stats.Config // 统计配置
}
