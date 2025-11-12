package cache

import (
	"sync/atomic"
)

// CacheMetrics 缓存指标统计
type CacheMetrics struct {
	FlowListHit        atomic.Int64 // Flow列表缓存命中次数 (DB数据)
	FlowListMiss       atomic.Int64 // Flow列表缓存未命中次数 (DB数据)
	FlowFieldsHit      atomic.Int64 // Flow字段缓存命中次数 (DB数据)
	FlowFieldsMiss     atomic.Int64 // Flow字段缓存未命中次数 (DB数据)
	FlowInfoHit        atomic.Int64 // Flow信息缓存命中次数 (DB数据)
	FlowInfoMiss       atomic.Int64 // Flow信息缓存未命中次数 (DB数据)
	FlowInfoAPIHit     atomic.Int64 // Flow信息缓存命中次数 (API数据)
	FlowInfoAPIMiss    atomic.Int64 // Flow信息缓存未命中次数 (API数据)
	UserNameHit        atomic.Int64 // 用户名缓存命中次数
	UserNameMiss       atomic.Int64 // 用户名缓存未命中次数
	OrgMappingHit      atomic.Int64 // 组织映射缓存命中次数
	OrgMappingMiss     atomic.Int64 // 组织映射缓存未命中次数
	PlatformConfigHit  atomic.Int64 // 平台配置缓存命中次数
	PlatformConfigMiss atomic.Int64 // 平台配置缓存未命中次数
	EventConfigHit     atomic.Int64 // 事件配置缓存命中次数
	EventConfigMiss    atomic.Int64 // 事件配置缓存未命中次数
	StatsHit           atomic.Int64 // 统计结果缓存命中次数
	StatsMiss          atomic.Int64 // 统计结果缓存未命中次数
}

// CacheStats 缓存统计结果
type CacheStats struct {
	Type     string  // 缓存类型
	Hit      int64   // 命中次数
	Miss     int64   // 未命中次数
	Total    int64   // 总请求次数
	HitRate  float64 // 命中率 (0-100)
	MissRate float64 // 未命中率 (0-100)
}

// GetStats 获取指定类型的缓存统计
func (m *CacheMetrics) GetStats(cacheType string) *CacheStats {
	var hit, miss int64

	switch cacheType {
	case "flow_list":
		hit = m.FlowListHit.Load()
		miss = m.FlowListMiss.Load()
	case "flow_fields":
		hit = m.FlowFieldsHit.Load()
		miss = m.FlowFieldsMiss.Load()
	case "flow_info":
		hit = m.FlowInfoHit.Load()
		miss = m.FlowInfoMiss.Load()
	case "flow_info_api":
		hit = m.FlowInfoAPIHit.Load()
		miss = m.FlowInfoAPIMiss.Load()
	case "user_name":
		hit = m.UserNameHit.Load()
		miss = m.UserNameMiss.Load()
	case "org_mapping":
		hit = m.OrgMappingHit.Load()
		miss = m.OrgMappingMiss.Load()
	case "platform_config":
		hit = m.PlatformConfigHit.Load()
		miss = m.PlatformConfigMiss.Load()
	case "event_config":
		hit = m.EventConfigHit.Load()
		miss = m.EventConfigMiss.Load()
	case "stats":
		hit = m.StatsHit.Load()
		miss = m.StatsMiss.Load()
	default:
		return nil
	}

	total := hit + miss
	hitRate := float64(0)
	missRate := float64(0)
	if total > 0 {
		hitRate = float64(hit) / float64(total) * 100
		missRate = float64(miss) / float64(total) * 100
	}

	return &CacheStats{
		Type:     cacheType,
		Hit:      hit,
		Miss:     miss,
		Total:    total,
		HitRate:  hitRate,
		MissRate: missRate,
	}
}

// GetAllStats 获取所有缓存类型的统计
func (m *CacheMetrics) GetAllStats() []*CacheStats {
	types := []string{
		"flow_list", "flow_fields", "flow_info", "flow_info_api",
		"user_name", "org_mapping",
		"platform_config", "event_config", "stats",
	}

	stats := make([]*CacheStats, 0, len(types))
	for _, t := range types {
		if s := m.GetStats(t); s != nil {
			stats = append(stats, s)
		}
	}

	return stats
}

// Reset 重置所有统计数据
func (m *CacheMetrics) Reset() {
	m.FlowListHit.Store(0)
	m.FlowListMiss.Store(0)
	m.FlowFieldsHit.Store(0)
	m.FlowFieldsMiss.Store(0)
	m.FlowInfoHit.Store(0)
	m.FlowInfoMiss.Store(0)
	m.FlowInfoAPIHit.Store(0)
	m.FlowInfoAPIMiss.Store(0)
	m.UserNameHit.Store(0)
	m.UserNameMiss.Store(0)
	m.OrgMappingHit.Store(0)
	m.OrgMappingMiss.Store(0)
	m.PlatformConfigHit.Store(0)
	m.PlatformConfigMiss.Store(0)
	m.EventConfigHit.Store(0)
	m.EventConfigMiss.Store(0)
	m.StatsHit.Store(0)
	m.StatsMiss.Store(0)
}
