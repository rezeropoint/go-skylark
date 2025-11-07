package query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// queryManager 远程查询管理器实现
type queryManager struct {
	config          Config                            // 配置参数
	dbConn          sqlx.SqlConn                      // 本地数据库连接（查询事件配置、字段配置、组织映射）
	getRemoteDB     core.GetRemoteDBFunc              // 获取远程数据库连接的函数（由 Platform Manager 提供）
	getEventConfig  core.GetEventConfigWithFieldsFunc // 获取事件配置（含字段）的函数（由 Event Manager 提供）
	listOrgMappings core.ListOrgMappingsFunc          // 获取组织映射列表的函数（由 Mapping Manager 提供）
	cache           core.CacheInterface               // 缓存接口（统一缓存管理）
}

// newQueryManager 创建远程查询管理器
func newQueryManager(
	config Config,
	db sqlx.SqlConn,
	getRemoteDB core.GetRemoteDBFunc,
	getEventConfig core.GetEventConfigWithFieldsFunc,
	listOrgMappings core.ListOrgMappingsFunc,
	cache core.CacheInterface,
) (*queryManager, error) {
	// 验证必填参数
	if getRemoteDB == nil {
		return nil, fmt.Errorf("getRemoteDB 函数不能为空")
	}
	if getEventConfig == nil {
		return nil, fmt.Errorf("getEventConfig 函数不能为空")
	}
	if listOrgMappings == nil {
		return nil, fmt.Errorf("listOrgMappings 函数不能为空")
	}

	// 缓存接口必须提供
	if cache == nil {
		return nil, fmt.Errorf("缓存接口不能为空")
	}

	// 验证和设置默认配置
	if config.FlowListCacheTTL <= 0 {
		config.FlowListCacheTTL = time.Hour
	}
	if config.FlowFieldsCacheTTL <= 0 {
		config.FlowFieldsCacheTTL = time.Hour
	}
	if config.UserNameCacheTTL <= 0 {
		config.UserNameCacheTTL = 24 * time.Hour
	}

	if config.MaxPageSize <= 0 {
		config.MaxPageSize = 1000
	}
	if config.DefaultPageSize <= 0 {
		config.DefaultPageSize = 20
	}
	if config.QueryTimeout <= 0 {
		config.QueryTimeout = 30 * time.Second
	}

	manager := &queryManager{
		config:          config,
		dbConn:          db,
		getRemoteDB:     getRemoteDB,
		getEventConfig:  getEventConfig,
		listOrgMappings: listOrgMappings,
		cache:           cache,
	}

	return manager, nil
}

// ========== 远程流程查询 ==========

// GetFlowList 获取远程flows列表（供前端配置使用）
func (m *queryManager) GetFlowList(ctx context.Context, tenantID string) ([]*core.FlowInfo, error) {
	// 1. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 2. 读取平台配置获取 namespace_id
	var namespaceID int
	configQuery := "SELECT namespace_id FROM skylark_platform_configs WHERE tenant_id = $1"
	err = m.dbConn.QueryRowCtx(ctx, &namespaceID, configQuery, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrPlatformConfigNotFound
		}
		return nil, fmt.Errorf("读取平台配置失败: %w", err)
	}

	// 3. 尝试从缓存获取
	var flows []*core.FlowInfo
	flows, err = m.cache.GetFlowList(ctx, tenantID, namespaceID)
	if err == nil && flows != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_flow_list"),
			logx.Field("tenant_id", tenantID),
			logx.Field("namespace_id", namespaceID),
			logx.Field("source", "cache"),
			logx.Field("count", len(flows)),
		).Info("从缓存获取 flows 列表成功")
		return flows, nil
	}

	// 4. 查询远程数据库（移除 flow_version 字段，排除 flow_version = '0' 的记录）
	flowQuery := `
        SELECT id, title, namespace_id
        FROM flows
        WHERE namespace_id = $1
          AND (flow_version IS NULL OR flow_version <> '0')
        ORDER BY id DESC
    `
	err = remoteDB.QueryRowsCtx(ctx, &flows, flowQuery, namespaceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询远程 flows 表失败: %w", err)
	}

	// 如果没有记录，返回空数组
	if flows == nil {
		flows = []*core.FlowInfo{}
	}

	// 5. 写入缓存
	if len(flows) > 0 {
		_ = m.cache.SetFlowList(ctx, tenantID, namespaceID, flows, int(m.config.FlowListCacheTTL.Seconds()))
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_flow_list"),
		logx.Field("tenant_id", tenantID),
		logx.Field("namespace_id", namespaceID),
		logx.Field("source", "database"),
		logx.Field("count", len(flows)),
	).Info("查询远程 flows 列表成功")

	return flows, nil
}

// GetFlowFields 获取远程flow字段列表（供前端配置使用）
func (m *queryManager) GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error) {
	// 1. 验证 flow_id
	if flowID <= 0 {
		return nil, core.ErrInvalidFlowID
	}

	// 2. 尝试从缓存获取
	fields, err := m.cache.GetFlowFields(ctx, tenantID, flowID)
	if err == nil && fields != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "query_manager"),
			logx.Field("operation", "get_flow_fields"),
			logx.Field("tenant_id", tenantID),
			logx.Field("flow_id", flowID),
			logx.Field("source", "cache"),
			logx.Field("count", len(fields)),
		).Info("从缓存获取 flow 字段列表成功")
		return fields, nil
	}

	// 3. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 4. 构建表名并查询字段元数据
	tableName := fmt.Sprintf("assignments_%d", flowID)
	fieldQuery := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		ORDER BY ordinal_position
	`
	var allFields []*core.FieldMetadata
	err = remoteDB.QueryRowsCtx(ctx, &allFields, fieldQuery, tableName)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询远程表字段失败: %w", err)
	}

	// 5. 过滤并标记系统字段
	fields = make([]*core.FieldMetadata, 0, len(allFields))
	for _, field := range allFields {
		// 标记是否为系统字段
		field.IsSystem = core.IsSystemField(field.FieldName)
		fields = append(fields, field)
	}

	// 如果没有字段，返回空数组
	if fields == nil {
		fields = []*core.FieldMetadata{}
	}

	// 6. 写入缓存
	if len(fields) > 0 {
		_ = m.cache.SetFlowFields(ctx, tenantID, flowID, fields, int(m.config.FlowFieldsCacheTTL.Seconds()))
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_flow_fields"),
		logx.Field("tenant_id", tenantID),
		logx.Field("flow_id", flowID),
		logx.Field("source", "database"),
		logx.Field("count", len(fields)),
	).Info("查询远程 flow 字段列表成功")

	return fields, nil
}

// ========== 远程事件数据查询 ==========

// QueryEventData 查询事件数据列表（Journey聚合 + 权限过滤）
func (m *queryManager) QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error) {
	// 1. 验证基本请求参数
	if req.EventConfigID == "" {
		return nil, fmt.Errorf("%w: 事件配置ID不能为空", core.ErrInvalidQueryParam)
	}
	if req.Page < 1 {
		return nil, core.ErrInvalidPageParam
	}
	if req.PageSize <= 0 {
		req.PageSize = m.config.DefaultPageSize
	}
	if req.PageSize > m.config.MaxPageSize {
		return nil, fmt.Errorf("每页数量不能超过%d", m.config.MaxPageSize)
	}

	// 2. 加载事件配置（含字段）
	eventConfigWithFields, err := m.getEventConfig(ctx, req.EventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 加载组织映射
	orgMappings, err := m.listOrgMappings(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载组织映射失败: %w", err)
	}

	// 4. 过滤可见字段
	visibleFields := make([]*core.FieldConfig, 0)
	for _, field := range eventConfigWithFields.Fields {
		if field.IsVisible {
			visibleFields = append(visibleFields, field)
		}
	}
	if len(visibleFields) == 0 {
		return nil, core.ErrNoVisibleFields
	}

	// 5. 验证字段名（防SQL注入）
	for _, field := range visibleFields {
		if err := validateFieldName(field.FieldName); err != nil {
			return nil, fmt.Errorf("字段名 '%s' 无效: %w", field.FieldName, err)
		}
	}
	if req.SortField != "" {
		if err := validateFieldName(req.SortField); err != nil {
			return nil, fmt.Errorf("排序字段名 '%s' 无效: %w", req.SortField, err)
		}
	}

	// 6. 计算组织权限（UserOrgIDs → RemoteOrgValues）
	allowedOrgValues, err := m.calculateAllowedOrgValues(req.UserOrgIDs, orgMappings)
	if err != nil {
		return nil, err
	}

	// 6.1 如果配置了组织字段但没有任何映射，返回空结果（防止查询所有数据）
	if eventConfigWithFields.EventConfig.OrgFieldName != nil && len(orgMappings) > 0 && len(allowedOrgValues) == 0 {
		// 有映射配置，但用户组织不在映射中，返回空结果
		return &core.QueryResponse{
			Columns: buildColumnInfo(visibleFields),
			Records: []map[string]interface{}{},
			Total:   0,
		}, nil
	}

	// 7. 构建动态 SQL（DISTINCT ON Journey聚合）
	querySQL, queryArgs, err := m.buildQuerySQLWithConfig(req, &eventConfigWithFields.EventConfig, visibleFields, allowedOrgValues)
	if err != nil {
		return nil, err
	}

	// 8. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 9. 执行查询并解析结果
	records, err := m.executeQueryAndParse(ctx, remoteDB, querySQL, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("查询远程事件数据失败: %w", err)
	}

	// 10. 查询总记录数（Journey数量）
	total, err := m.countJourneysWithConfig(ctx, remoteDB, req, &eventConfigWithFields.EventConfig, allowedOrgValues)
	if err != nil {
		return nil, fmt.Errorf("查询总记录数失败: %w", err)
	}

	// 11. 构建列定义
	columns := buildColumnInfo(visibleFields)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "query_event_data"),
		logx.Field("event_config_id", req.EventConfigID),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("page", req.Page),
		logx.Field("page_size", req.PageSize),
		logx.Field("total", total),
		logx.Field("records", len(records)),
	).Info("查询事件数据成功")

	return &core.QueryResponse{
		Columns: columns,
		Records: records,
		Total:   total,
	}, nil
}

// GetEventDetail 获取事件详情（完整流转历史 + 用户名转换）
func (m *queryManager) GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error) {
	// 1. 验证请求参数
	if req.EventConfigID == "" {
		return nil, fmt.Errorf("%w: 事件配置ID不能为空", core.ErrInvalidQueryParam)
	}
	if req.JourneyID <= 0 {
		return nil, fmt.Errorf("%w: Journey ID 必须大于0", core.ErrInvalidQueryParam)
	}

	// 2. 加载事件配置
	eventConfigWithFields, err := m.getEventConfig(ctx, req.EventConfigID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("加载事件配置失败: %w", err)
	}

	// 3. 获取远程数据库连接
	remoteDB, err := m.getRemoteDB(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取远程数据库连接失败: %w", err)
	}

	// 4. 查询 Journey 的所有 Assignment（完整流转历史）
	assignments, err := m.queryJourneyAssignments(ctx, remoteDB, &eventConfigWithFields.EventConfig, req.JourneyID)
	if err != nil {
		return nil, err
	}

	if len(assignments) == 0 {
		return nil, core.ErrJourneyNotFound
	}

	// 5. 提取用户 ID 列表
	userIDs := extractUserIDs(assignments)

	// 6. 批量查询用户名（优先从 Redis 缓存获取）
	userNames, err := m.batchGetUserNames(ctx, remoteDB, req.TenantID, userIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询用户名失败: %w", err)
	}

	// 7. 组装 DetailResponse（传入可见字段列表，用于过滤业务数据）
	response := m.buildDetailResponse(assignments, userNames, eventConfigWithFields.Fields)

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "query_manager"),
		logx.Field("operation", "get_event_detail"),
		logx.Field("event_config_id", req.EventConfigID),
		logx.Field("journey_id", req.JourneyID),
		logx.Field("tenant_id", req.TenantID),
		logx.Field("assignment_count", len(assignments)),
	).Info("查询事件详情成功")

	return response, nil
}

// ========== 辅助方法 ==========

// batchGetUserNames 批量查询用户名（优先从缓存获取）
func (m *queryManager) batchGetUserNames(ctx context.Context, remoteDB sqlx.SqlConn, tenantID string, userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return make(map[string]string), nil
	}

	userNames := make(map[string]string, len(userIDs))
	uncachedIDs := []string{}

	// 1. 尝试从缓存获取
	for _, userID := range userIDs {
		val, err := m.cache.GetUserName(ctx, tenantID, userID)
		if err == nil && val != "" {
			userNames[userID] = val
		} else {
			uncachedIDs = append(uncachedIDs, userID)
		}
	}

	// 2. 查询未命中的用户名（从远程 users 表）
	if len(uncachedIDs) > 0 {
		type userRow struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}

		query := "SELECT id, name FROM users WHERE id = ANY($1)"
		var users []*userRow
		err := remoteDB.QueryRowsCtx(ctx, &users, query, pq.Array(uncachedIDs))
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("批量查询用户名失败: %w", err)
		}

		for _, user := range users {
			userNames[user.ID] = user.Name

			// 3. 写入缓存
			_ = m.cache.SetUserName(ctx, tenantID, user.ID, user.Name, int(m.config.UserNameCacheTTL.Seconds()))
		}
	}

	return userNames, nil
}
