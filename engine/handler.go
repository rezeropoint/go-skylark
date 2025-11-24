package engine

import (
	"context"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/cache"
	"github.com/rezeropoint/go-skylark/v2/internal/event"
	"github.com/rezeropoint/go-skylark/v2/internal/flows"
	"github.com/rezeropoint/go-skylark/v2/internal/forms"
	"github.com/rezeropoint/go-skylark/v2/internal/mapping"
	"github.com/rezeropoint/go-skylark/v2/internal/organization"
	"github.com/rezeropoint/go-skylark/v2/internal/platform"
	"github.com/rezeropoint/go-skylark/v2/internal/query"
	"github.com/rezeropoint/go-skylark/v2/internal/stats"
	"github.com/rezeropoint/go-skylark/v2/internal/user"

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
func newSkylarkEngine(config Config, db sqlx.SqlConn, redisClient *redis.Redis) (*skylarkEngine, error) {
	if db == nil {
		return nil, core.ErrLocalDBNil
	}

	// 初始化缓存
	cache := cache.NewSkylarkCache(redisClient, config.Cache)

	// 1. 初始化平台管理器（核心依赖，最先初始化）
	// 设置默认值
	if config.Platform.PlatformConfigCacheTTL == 0 {
		config.Platform.PlatformConfigCacheTTL = 30 * time.Minute
	}
	platformMgr, err := platform.NewManager(config.Platform, db, cache)
	if err != nil {
		return nil, err
	}

	// 2. 初始化用户ID映射管理器（依赖 Platform）
	userMgr, err := user.NewManager(user.Config{}, db, cache, platformMgr.GetAPIConfig)
	if err != nil {
		return nil, err
	}

	// 3. 初始化组织ID映射管理器（依赖 Platform + User）
	orgMgr, err := organization.NewManager(organization.Config{}, db, cache, platformMgr.GetAPIConfig, userMgr.GetRemoteUserIDs, userMgr.FillLocalUserIDMap)
	if err != nil {
		return nil, err
	}

	// 4. 初始化组织映射管理器（业务字段值映射）
	// 设置默认值
	if config.Mapping.OrgMappingCacheTTL == 0 {
		config.Mapping.OrgMappingCacheTTL = 7 * 24 * time.Hour
	}
	mappingMgr, err := mapping.NewManager(config.Mapping, db, cache)
	if err != nil {
		return nil, err
	}

	// 5. 初始化事件配置管理器（依赖 Platform，注入 platform.GetRemoteDB）
	// 设置默认值
	if config.Event.EventConfigCacheTTL == 0 {
		config.Event.EventConfigCacheTTL = 10 * time.Minute
	}
	if config.Event.EventConfigListCacheTTL == 0 {
		config.Event.EventConfigListCacheTTL = 5 * time.Minute
	}
	eventMgr, err := event.NewManager(config.Event, db, cache, platformMgr.GetRemoteDB)
	if err != nil {
		return nil, err
	}

	// 6. 初始化流程和表单管理器（依赖 Platform + User + Event）
	flows, err := flows.NewSkylarkFlowRegistry(flows.Config{}, cache, platformMgr.GetAPIConfig, userMgr.GetRemoteUserIDs, userMgr.FillLocalUserIDMap, platformMgr.GetRemoteDB, eventMgr.ListConfiguredFlowIDs)
	if err != nil {
		return nil, err
	}

	forms, err := forms.NewSkylarkFormRegistry(forms.Config{}, cache)
	if err != nil {
		return nil, err
	}

	// 7. 初始化查询管理器（注入多个依赖函数）
	queryMgr, err := query.NewManager(
		config.Query,
		db,
		platformMgr.GetRemoteDB,
		eventMgr.GetWithFields,
		mappingMgr.ListOrgMappings,
		userMgr.FillLocalUserIDMap,
		eventMgr.ListConfiguredFlowIDs,
		cache,
	)
	if err != nil {
		return nil, err
	}

	// 8. 初始化统计管理器（注入多个依赖函数）
	statsMgr, err := stats.NewManager(
		config.Stats,
		db,
		platformMgr.GetRemoteDB,
		eventMgr.GetWithFields,
		mappingMgr.ListOrgMappings,
		userMgr.FillLocalUserIDMap,
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

func (e *skylarkEngine) UpdateFlowJourneyStatus(ctx context.Context, tenantID string, flowID int64, journeyID int64, assignmentID int64, localUserID string, operation core.JourneyOperation, options core.UpdateJourneyStatusOptions) error {
	return e.flows.UpdateJourneyStatus(ctx, tenantID, flowID, journeyID, assignmentID, localUserID, operation, options)
}

// GetFlowJourneyAssignments 获取流程节点处理信息列表
func (e *skylarkEngine) GetFlowJourneyAssignments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Assignment, error) {
	return e.flows.GetJourneyAssignments(ctx, tenantID, journeyID)
}

// GetFlowJourneyDetail 获取流程记录详情（包含字段值和附件）
func (e *skylarkEngine) GetFlowJourneyDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyDetail, error) {
	return e.flows.GetJourneyDetail(ctx, tenantID, flowID, journeyID)
}

// GetFlowDetail 获取流程详情（包含字段、节点、边信息）
func (e *skylarkEngine) GetFlowDetail(ctx context.Context, tenantID string, flowID int64) (*core.FlowDetail, error) {
	return e.flows.GetFlowDetail(ctx, tenantID, flowID)
}

// GetUserAssignments 获取用户处理的任务列表
func (e *skylarkEngine) GetUserAssignments(ctx context.Context, tenantID string, localUserID string, category string, page, pageSize int) ([]*core.Assignment, int, error) {
	return e.flows.GetUserAssignments(ctx, tenantID, localUserID, category, page, pageSize)
}

// GetProposedJourneys 获取用户发起的流程列表
func (e *skylarkEngine) GetProposedJourneys(ctx context.Context, tenantID string, flowID int64, localUserID string, page, pageSize int) ([]*core.Journey, int, error) {
	return e.flows.GetProposedJourneys(ctx, tenantID, flowID, localUserID, page, pageSize)
}

// SearchJourneys 搜索流程记录
func (e *skylarkEngine) SearchJourneys(ctx context.Context, tenantID string, req *core.JourneySearchRequest) ([]*core.Journey, int, error) {
	return e.flows.SearchJourneys(ctx, tenantID, req)
}

// GetJourneyMoments 获取流程审批历史
func (e *skylarkEngine) GetJourneyMoments(ctx context.Context, tenantID string, journeyID int64) ([]*core.Moment, error) {
	return e.flows.GetJourneyMoments(ctx, tenantID, journeyID)
}

// GetJourneyFullDetail 获取流程完整详情（一站式接口）
func (e *skylarkEngine) GetJourneyFullDetail(ctx context.Context, tenantID string, flowID int64, journeyID int64) (*core.JourneyFullDetail, error) {
	return e.flows.GetJourneyFullDetail(ctx, tenantID, flowID, journeyID)
}

// GetCurrentProcessingUsers 获取当前流程任务的处理者
func (e *skylarkEngine) GetCurrentProcessingUsers(ctx context.Context, tenantID string, flowID int64, journeyID int64) ([]*core.ProcessingUser, error) {
	return e.flows.GetCurrentProcessingUsers(ctx, tenantID, flowID, journeyID)
}

// AbortJourney 终止流程任务
func (e *skylarkEngine) AbortJourney(ctx context.Context, tenantID string, flowID int64, journeyID int64) error {
	return e.flows.AbortJourney(ctx, tenantID, flowID, journeyID)
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

// 资源管理方法
