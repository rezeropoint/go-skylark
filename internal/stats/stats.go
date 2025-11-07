package stats

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 统计分析管理器接口
// 职责：查询远程Skylark数据库 + 组织权限过滤 + SQL构建 + 用户名转换 + 统计分析
type Manager interface {
	GetDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error) // GetDurationStats 获取事件处理时长统计（平均/最短/最长时长）
	GetStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error)     // GetStatusStats 获取事件状态统计（各状态数量和占比）
	GetTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error)       // GetTrendStats 获取事件趋势统计（按日/周/月聚合）
	GetNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error)         // GetNodeStats 获取节点统计（各节点事件数和平均时长）
	GetUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error)         // GetUserStats 获取处理人统计（Top N处理人排名）
	GetOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error)           // GetOrgStats 获取组织统计（各组织事件数和平均时长）
	GetPendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error)   // GetPendingStats 获取待处理事件统计（实时查询，无缓存）
}

// NewManager 创建远程查询管理器
// 参数：
//   - config: 查询管理器配置
//   - db: 本地数据库连接
//   - getRemoteDB: 获取远程数据库连接的函数（由 Platform Manager 提供）
//   - getEventConfig: 获取事件配置（含字段）的函数（由 Event Manager 提供）
//   - listOrgMappings: 获取组织映射列表的函数（由 Mapping Manager 提供）
//   - cache: 缓存接口（必须提供，用于缓存）
func NewManager(
	config Config,
	db sqlx.SqlConn,
	getRemoteDB core.GetRemoteDBFunc,
	getEventConfig core.GetEventConfigWithFieldsFunc,
	listOrgMappings core.ListOrgMappingsFunc,
	cache core.CacheInterface,
) (Manager, error) {
	return newStatsManager(config, db, getRemoteDB, getEventConfig, listOrgMappings, cache)
}
