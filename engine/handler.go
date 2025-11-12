package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/cache"
	"github.com/rezeropoint/go-skylark/internal/event"
	"github.com/rezeropoint/go-skylark/internal/flows"
	"github.com/rezeropoint/go-skylark/internal/forms"
	"github.com/rezeropoint/go-skylark/internal/mapping"
	"github.com/rezeropoint/go-skylark/internal/organization"
	"github.com/rezeropoint/go-skylark/internal/platform"
	"github.com/rezeropoint/go-skylark/internal/query"
	"github.com/rezeropoint/go-skylark/internal/stats"
	"github.com/rezeropoint/go-skylark/internal/user"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type skylarkEngine struct {
	cache        *cache.SkylarkCache
	flows        flows.SkylarkFlowRegistry
	forms        forms.SkylarkFormRegistry
	platform     platform.Manager
	organization organization.Manager
	user         user.Manager
	event        event.Manager
	mapping      mapping.Manager
	query        query.Manager
	stats        stats.Manager
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

	// 1. 初始化平台管理器（核心依赖，最先初始化）
	platformMgr, err := platform.NewManager(platform.Config{}, db)
	if err != nil {
		return nil, err
	}

	// 2. 初始化组织ID映射管理器（依赖 Platform）
	orgMgr, err := organization.NewManager(organization.Config{}, db, cache, platformMgr.Get)
	if err != nil {
		return nil, err
	}

	// 3. 初始化用户ID映射管理器（依赖 Platform）
	userMgr, err := user.NewManager(user.Config{}, db, cache, platformMgr.Get)
	if err != nil {
		return nil, err
	}

	// 4. 初始化流程和表单管理器（依赖 Platform + User）
	flows, err := flows.NewSkylarkFlowRegistry(&flows.Config{}, cache, platformMgr.Get, userMgr.GetRemoteUserIDs)
	if err != nil {
		return nil, err
	}

	forms, err := forms.NewSkylarkFormRegistry(&forms.Config{}, cache)
	if err != nil {
		return nil, err
	}

	// 5. 初始化组织映射管理器（业务字段值映射）
	mappingMgr, err := mapping.NewManager(db, cache)
	if err != nil {
		return nil, err
	}

	// 6. 初始化事件配置管理器（注入 platform.GetRemoteDB）
	eventMgr, err := event.NewManager(db, platformMgr.GetRemoteDB)
	if err != nil {
		return nil, err
	}

	// 7. 初始化查询管理器（注入多个依赖函数）
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

	// 8. 初始化统计管理器（注入多个依赖函数）
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
		cache:        cache,
		flows:        flows,
		forms:        forms,
		platform:     platformMgr,
		organization: orgMgr,
		user:         userMgr,
		event:        eventMgr,
		mapping:      mappingMgr,
		query:        queryMgr,
		stats:        statsMgr,
	}, nil
}

func (e *skylarkEngine) CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.flows.CreateFlow(ctx, app, flowID, userID, authHeader, data)
}

func (e *skylarkEngine) CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.forms.CreateFormRow(ctx, app, formID, userID, authHeader, data)
}

func (e *skylarkEngine) UpdateFlowJourneyStatus(ctx context.Context, tenantID string, flowID int64, journeyID int64, assignmentID int64, localUserID string, operation core.JourneyOperation, options flows.UpdateJourneyStatusOptions) error {
	return e.flows.UpdateJourneyStatus(ctx, tenantID, flowID, journeyID, assignmentID, localUserID, operation, options)
}

// GetFlowJourneyBySN 根据流程编号查询流程记录
func (e *skylarkEngine) GetFlowJourneyBySN(ctx context.Context, tenantID string, flowID int64, sn string) (*core.Journey, error) {
	return e.flows.GetJourneyBySN(ctx, tenantID, flowID, sn)
}

// GetFlowJourneyAssignments 获取流程节点处理信息列表
func (e *skylarkEngine) GetFlowJourneyAssignments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Assignment, error) {
	return e.flows.GetJourneyAssignments(ctx, tenantID, journeyID)
}

// Close 关闭引擎，释放资源（尤其是 platform 管理的远程数据库连接池）
func (e *skylarkEngine) Close() error {
	if e.platform != nil {
		return e.platform.Close()
	}
	return nil
}

// 平台配置管理方法

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

// 事件配置管理方法

func (e *skylarkEngine) CreateEventWithFields(ctx context.Context, creation *core.EventCreation) (string, error) {
	return e.event.CreateWithFields(ctx, creation)
}

func (e *skylarkEngine) UpdateEventWithFields(ctx context.Context, update *core.EventUpdate) error {
	return e.event.UpdateWithFields(ctx, update)
}

func (e *skylarkEngine) GetEventWithFields(ctx context.Context, id, tenantID string) (*core.EventAggregate, error) {
	return e.event.GetWithFields(ctx, id, tenantID)
}

func (e *skylarkEngine) ListEventWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error) {
	return e.event.ListWithFields(ctx, tenantID, enabled)
}

func (e *skylarkEngine) DeleteEvent(ctx context.Context, id, tenantID string) error {
	return e.event.Delete(ctx, id, tenantID)
}

// 组织映射管理方法

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

// 远程查询方法

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

// 统计分析方法

func (e *skylarkEngine) GetDurationStats(ctx context.Context, criteria *core.StatsCriteria) (*core.DurationStats, error) {
	return e.stats.GetDurationStats(ctx, criteria)
}

func (e *skylarkEngine) GetStatusStats(ctx context.Context, criteria *core.StatsCriteria) (*core.StatusStats, error) {
	return e.stats.GetStatusStats(ctx, criteria)
}

func (e *skylarkEngine) GetTrendStats(ctx context.Context, criteria *core.StatsCriteria) (*core.TrendStats, error) {
	return e.stats.GetTrendStats(ctx, criteria)
}

func (e *skylarkEngine) GetNodeStats(ctx context.Context, criteria *core.StatsCriteria) (*core.NodeStats, error) {
	return e.stats.GetNodeStats(ctx, criteria)
}

func (e *skylarkEngine) GetUserStats(ctx context.Context, criteria *core.StatsCriteria) (*core.UserStats, error) {
	return e.stats.GetUserStats(ctx, criteria)
}

func (e *skylarkEngine) GetOrgStats(ctx context.Context, criteria *core.StatsCriteria) (*core.OrgStats, error) {
	return e.stats.GetOrgStats(ctx, criteria)
}

func (e *skylarkEngine) GetPendingStats(ctx context.Context, criteria *core.StatsCriteria) (*core.PendingStats, error) {
	return e.stats.GetPendingStats(ctx, criteria)
}

// 组织管理方法

func (e *skylarkEngine) CreateOrganization(ctx context.Context, tenantID, localOrgID, name, description string, founderID int) (*core.Organization, error) {
	return e.organization.CreateOrganization(ctx, tenantID, localOrgID, name, description, founderID)
}

func (e *skylarkEngine) CreateSubOrganization(ctx context.Context, tenantID, localOrgID, parentLocalOrgID, name, description string, founderID int) (*core.Organization, error) {
	return e.organization.CreateSubOrganization(ctx, tenantID, localOrgID, parentLocalOrgID, name, description, founderID)
}

func (e *skylarkEngine) DeleteOrganization(ctx context.Context, tenantID, localOrgID string) error {
	return e.organization.DeleteOrganization(ctx, tenantID, localOrgID)
}

// 用户管理接口

func (e *skylarkEngine) CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid *string) (*core.User, error) {
	return e.user.CreateUser(ctx, tenantID, localUserID, name, identifier, phone, openid)
}

func (e *skylarkEngine) GetUser(ctx context.Context, tenantID, localUserID string) (*core.User, error) {
	return e.user.GetUser(ctx, tenantID, localUserID)
}

// 资源管理方法
