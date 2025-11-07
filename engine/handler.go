package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/cache"
	"github.com/rezeropoint/go-skylark/internal/event"
	"github.com/rezeropoint/go-skylark/internal/flows"
	"github.com/rezeropoint/go-skylark/internal/forms"
	"github.com/rezeropoint/go-skylark/internal/mapping"
	"github.com/rezeropoint/go-skylark/internal/platform"
	"github.com/rezeropoint/go-skylark/internal/query"
	"github.com/rezeropoint/go-skylark/internal/stats"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type skylarkEngine struct {
	cache    *cache.SkylarkCache
	flows    flows.SkylarkFlowRegistry
	forms    forms.SkylarkFormRegistry
	platform platform.Manager
	event    event.Manager
	mapping  mapping.Manager
	query    query.Manager
	stats    stats.Manager
}

// newSkylarkEngine 创建新的 Skylark 引擎实例
func newSkylarkEngine(config *Config, db sqlx.SqlConn, redisClient *redis.Redis) (*skylarkEngine, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}
	if db == nil {
		return nil, core.ErrLocalDBNil
	}

	// 初始化缓存
	cache := cache.NewSkylarkCache(redisClient, config.Cache)

	// 初始化流程和表单管理器
	flows, err := flows.NewSkylarkFlowRegistry(&flows.Config{}, cache)
	if err != nil {
		return nil, err
	}

	forms, err := forms.NewSkylarkFormRegistry(&forms.Config{}, cache)
	if err != nil {
		return nil, err
	}

	// 1. 初始化平台管理器（核心依赖，最先初始化）
	platformMgr, err := platform.NewManager(db)
	if err != nil {
		return nil, err
	}

	// 2. 初始化组织映射管理器
	mappingMgr, err := mapping.NewManager(db, cache)
	if err != nil {
		return nil, err
	}

	// 3. 初始化事件配置管理器（注入 platform.GetRemoteDB）
	eventMgr, err := event.NewManager(db, platformMgr.GetRemoteDB)
	if err != nil {
		return nil, err
	}

	// 4. 初始化查询管理器（注入多个依赖函数）
	queryConfig := config.Query
	if queryConfig == nil {
		queryConfig = &query.Config{} // 使用默认配置
	}
	queryMgr, err := query.NewManager(
		*queryConfig,
		db,
		platformMgr.GetRemoteDB,
		eventMgr.GetWithFields,
		mappingMgr.ListOrgMappings,
		cache,
	)
	if err != nil {
		return nil, err
	}

	// 5. 初始化统计管理器（注入多个依赖函数）
	statsConfig := config.Stats
	if statsConfig == nil {
		statsConfig = &stats.Config{} // 使用默认配置
	}
	statsMgr, err := stats.NewManager(
		*statsConfig,
		db,
		platformMgr.GetRemoteDB,
		eventMgr.GetWithFields,
		mappingMgr.ListOrgMappings,
		cache,
	)
	if err != nil {
		return nil, err
	}

	return &skylarkEngine{
		cache:    cache,
		flows:    flows,
		forms:    forms,
		platform: platformMgr,
		event:    eventMgr,
		mapping:  mappingMgr,
		query:    queryMgr,
		stats:    statsMgr,
	}, nil
}

func (e *skylarkEngine) CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.flows.CreateFlow(ctx, app, flowID, userID, authHeader, data)
}

func (e *skylarkEngine) CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.forms.CreateFormRow(ctx, app, formID, userID, authHeader, data)
}

func (e *skylarkEngine) UpdateFlowJourneyStatus(ctx context.Context, app string, flowID int64, journeyID int64, assignmentID int64, userID int64, authHeader string, operation string, options flows.UpdateJourneyStatusOptions) error {
	return e.flows.UpdateJourneyStatus(ctx, app, flowID, journeyID, assignmentID, userID, authHeader, operation, options)
}

// Close 关闭引擎，释放资源（尤其是 platform 管理的远程数据库连接池）
func (e *skylarkEngine) Close() error {
	if e.platform != nil {
		return e.platform.Close()
	}
	return nil
}

// ========== 平台配置管理方法 ==========

func (e *skylarkEngine) CreatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) (string, error) {
	return e.platform.Create(ctx, cfg)
}

func (e *skylarkEngine) GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	return e.platform.Get(ctx, tenantID)
}

func (e *skylarkEngine) UpdatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error {
	return e.platform.Update(ctx, cfg)
}

func (e *skylarkEngine) DeletePlatformConfig(ctx context.Context, tenantID string) error {
	return e.platform.Delete(ctx, tenantID)
}

func (e *skylarkEngine) ValidatePlatformConfig(ctx context.Context, cfg *core.PlatformConfig) error {
	return e.platform.Validate(ctx, cfg)
}

// ========== 事件配置管理方法 ==========

func (e *skylarkEngine) CreateEventWithFields(ctx context.Context, req *core.CreateEventRequest) (string, error) {
	return e.event.CreateWithFields(ctx, req)
}

func (e *skylarkEngine) UpdateEventWithFields(ctx context.Context, req *core.UpdateEventRequest) error {
	return e.event.UpdateWithFields(ctx, req)
}

func (e *skylarkEngine) GetEventWithFields(ctx context.Context, id, tenantID string) (*core.EventConfigWithFields, error) {
	return e.event.GetWithFields(ctx, id, tenantID)
}

func (e *skylarkEngine) ListEventWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventConfigWithFields, error) {
	return e.event.ListWithFields(ctx, tenantID, enabled)
}

func (e *skylarkEngine) DeleteEvent(ctx context.Context, id, tenantID string) error {
	return e.event.Delete(ctx, id, tenantID)
}

// ========== 组织映射管理方法 ==========

func (e *skylarkEngine) CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) (string, error) {
	return e.mapping.CreateOrgMapping(ctx, mapping)
}

func (e *skylarkEngine) GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error) {
	return e.mapping.GetOrgMapping(ctx, id)
}

func (e *skylarkEngine) ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error) {
	return e.mapping.ListOrgMappings(ctx, tenantID)
}

func (e *skylarkEngine) UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error {
	return e.mapping.UpdateOrgMapping(ctx, mapping)
}

func (e *skylarkEngine) DeleteOrgMapping(ctx context.Context, id string) error {
	return e.mapping.DeleteOrgMapping(ctx, id)
}

// ========== 远程查询方法 ==========

func (e *skylarkEngine) QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error) {
	return e.query.QueryEventData(ctx, req)
}

func (e *skylarkEngine) GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error) {
	return e.query.GetEventDetail(ctx, req)
}

func (e *skylarkEngine) GetFlowList(ctx context.Context, tenantID string) ([]*core.FlowInfo, error) {
	return e.query.GetFlowList(ctx, tenantID)
}

func (e *skylarkEngine) GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error) {
	return e.query.GetFlowFields(ctx, tenantID, flowID)
}

// ========== 统计分析方法 ==========

func (e *skylarkEngine) GetDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error) {
	return e.stats.GetDurationStats(ctx, req)
}

func (e *skylarkEngine) GetStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error) {
	return e.stats.GetStatusStats(ctx, req)
}

func (e *skylarkEngine) GetTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error) {
	return e.stats.GetTrendStats(ctx, req)
}

func (e *skylarkEngine) GetNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error) {
	return e.stats.GetNodeStats(ctx, req)
}

func (e *skylarkEngine) GetUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error) {
	return e.stats.GetUserStats(ctx, req)
}

func (e *skylarkEngine) GetOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error) {
	return e.stats.GetOrgStats(ctx, req)
}

func (e *skylarkEngine) GetPendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error) {
	return e.stats.GetPendingStats(ctx, req)
}
