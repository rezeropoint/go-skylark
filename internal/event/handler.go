package event

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// eventManager 事件+字段配置管理器
type eventManager struct {
	config      Config               // 配置参数
	dbConn      sqlx.SqlConn         // 本地数据库连接
	cache       core.CacheInterface  // 缓存接口（可选）
	getRemoteDB core.GetRemoteDBFunc // 获取远程数据库连接的函数（用于验证flow_id、字段名等）
}

// newEventManager 创建事件+字段配置管理器
func newEventManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getRemoteDB core.GetRemoteDBFunc) (*eventManager, error) {
	if getRemoteDB == nil {
		return nil, fmt.Errorf("getRemoteDB 函数不能为空")
	}

	manager := &eventManager{
		config:      config,
		dbConn:      db,
		cache:       cache,
		getRemoteDB: getRemoteDB,
	}

	// 初始化数据库表（在包初始化时执行）
	ctx := context.Background()
	if err := manager.initTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化事件配置表失败: %w", err)
	}

	return manager, nil
}

// CreateWithFields 创建事件配置（包含字段）
func (m *eventManager) CreateWithFields(ctx context.Context, creation *core.EventCreation) (string, error) {
	// 1. 验证基本信息
	if creation.EventConfig.ID != "" {
		return "", fmt.Errorf("%w: 创建时不允许指定ID，系统会自动生成", core.ErrInvalidEventConfig)
	}

	if err := m.validateEventConfig(&creation.EventConfig); err != nil {
		return "", err
	}
	if len(creation.Fields) == 0 {
		return "", fmt.Errorf("%w: 至少需要配置一个字段", core.ErrInvalidEventConfig)
	}

	// 2. 验证字段配置
	for i := range creation.Fields {
		if err := m.validateFieldConfig(&creation.Fields[i]); err != nil {
			return "", fmt.Errorf("字段配置[%d]无效: %w", i, err)
		}
	}

	// 3. 生成事件配置ID
	creation.EventConfig.ID = uuid.New().String()

	var createdID string

	// 4. 使用事务创建
	err := m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 4.1 检查事件名称是否已存在（租户内唯一）
		var existingID string
		checkQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2"
		err := session.QueryRowCtx(ctx, &existingID, checkQuery, creation.EventConfig.Name, creation.EventConfig.TenantID)
		if err == nil {
			return fmt.Errorf("%w: 事件名称 '%s' 已存在", core.ErrInvalidEventConfig, creation.EventConfig.Name)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查事件名称失败: %w", err)
		}

		// 4.2 检查flow_id 是否已被使用（租户内唯一）
		checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2"
		err = session.QueryRowCtx(ctx, &existingID, checkFlowQuery, creation.EventConfig.FlowID, creation.EventConfig.TenantID)
		if err == nil {
			return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, creation.EventConfig.FlowID, existingID)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查flow_id 失败: %w", err)
		}

		// 4.3 插入事件配置
		id, err := m.insertEvent(ctx, session, &creation.EventConfig)
		if err != nil {
			return err
		}
		createdID = id

		// 4.4 为字段配置填event_config_id
		for i := range creation.Fields {
			creation.Fields[i].EventConfigID = createdID
		}

		// 4.5 批量插入字段配置
		if err := m.insertFields(ctx, session, creation.Fields); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "create_with_fields"),
		logx.Field("event_config_id", createdID),
		logx.Field("tenant_id", creation.EventConfig.TenantID),
		logx.Field("field_count", len(creation.Fields)),
	).Info("创建事件配置成功（包含字段）")

	// 5. 清除缓存（新增事件会影响列表）
	if m.cache != nil {
		trueVal := true
		falseVal := false
		// 清除事件配置列表缓存
		_ = m.cache.DeleteEventConfigList(ctx, creation.EventConfig.TenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteEventConfigList(ctx, creation.EventConfig.TenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteEventConfigList(ctx, creation.EventConfig.TenantID, nil)       // enabled=all

		// 清除 flow_id 列表缓存
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, creation.EventConfig.TenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, creation.EventConfig.TenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, creation.EventConfig.TenantID, nil)       // enabled=all
	}

	// 6. 返回创建的事件配置ID
	return createdID, nil
}

// UpdateWithFields 更新事件配置（包含字段）
func (m *eventManager) UpdateWithFields(ctx context.Context, update *core.EventUpdate) error {
	// 1. 验证基本信息
	if update.EventConfig.ID == "" {
		return fmt.Errorf("%w: 事件配置ID不能为空", core.ErrInvalidEventConfig)
	}
	if err := m.validateEventConfig(&update.EventConfig); err != nil {
		return err
	}
	if len(update.Fields) == 0 {
		return fmt.Errorf("%w: 至少需要配置一个字段", core.ErrInvalidEventConfig)
	}

	// 2. 验证字段配置
	for i := range update.Fields {
		if err := m.validateFieldConfig(&update.Fields[i]); err != nil {
			return fmt.Errorf("字段配置[%d]无效: %w", i, err)
		}
	}

	// 3. 使用事务更新（完整替换策略）
	err := m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 3.1 验证事件配置存在且有权限
		var existingTenantID string
		checkQuery := "SELECT tenant_id FROM event_configs WHERE id = $1"
		err := session.QueryRowCtx(ctx, &existingTenantID, checkQuery, update.EventConfig.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return core.ErrEventConfigNotFound
			}
			return fmt.Errorf("查询事件配置失败: %w", err)
		}

		// 3.2 验证租户ID一致
		if existingTenantID != update.EventConfig.TenantID {
			return fmt.Errorf("%w: 不允许修改租户ID", core.ErrInvalidEventConfig)
		}

		// 3.3 检查事件名称是否与其他事件冲突
		var conflictID string
		checkNameQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2 AND id != $3"
		err = session.QueryRowCtx(ctx, &conflictID, checkNameQuery, update.EventConfig.Name, update.EventConfig.TenantID, update.EventConfig.ID)
		if err == nil {
			return fmt.Errorf("%w: 事件名称 '%s' 已被其他事件使用", core.ErrInvalidEventConfig, update.EventConfig.Name)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查事件名称失败: %w", err)
		}

		// 3.4 检查flow_id 是否与其他事件冲突
		checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2 AND id != $3"
		err = session.QueryRowCtx(ctx, &conflictID, checkFlowQuery, update.EventConfig.FlowID, update.EventConfig.TenantID, update.EventConfig.ID)
		if err == nil {
			return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, update.EventConfig.FlowID, conflictID)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查flow_id 失败: %w", err)
		}

		// 3.5 更新事件配置基本信息
		if err := m.updateEvent(ctx, session, &update.EventConfig); err != nil {
			return err
		}

		// 3.6 删除所有旧字段配置（完整替换策略）
		if err := m.deleteFieldsByEventID(ctx, session, update.EventConfig.ID); err != nil {
			return err
		}

		// 3.7 为字段配置填event_config_id
		for i := range update.Fields {
			update.Fields[i].EventConfigID = update.EventConfig.ID
		}

		// 3.8 批量插入新字段配置
		if err := m.insertFields(ctx, session, update.Fields); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除缓存
	if m.cache != nil {
		// 清除单个事件配置缓存
		_ = m.cache.DeleteEventConfig(ctx, update.EventConfig.TenantID, update.EventConfig.ID)

		// 清除所有相关的列表缓存
		trueVal := true
		falseVal := false
		_ = m.cache.DeleteEventConfigList(ctx, update.EventConfig.TenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteEventConfigList(ctx, update.EventConfig.TenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteEventConfigList(ctx, update.EventConfig.TenantID, nil)       // enabled=all

		// 清除 flow_id 列表缓存
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, update.EventConfig.TenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, update.EventConfig.TenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, update.EventConfig.TenantID, nil)       // enabled=all
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "update_with_fields"),
		logx.Field("event_config_id", update.EventConfig.ID),
		logx.Field("tenant_id", update.EventConfig.TenantID),
		logx.Field("field_count", len(update.Fields)),
	).Info("更新事件配置成功（包含字段）")

	return nil
}

// GetWithFields 查询事件配置（包含字段）
func (m *eventManager) GetWithFields(ctx context.Context, id, tenantID string) (*core.EventAggregate, error) {
	// 1. 尝试从缓存获取
	if m.cache != nil {
		cached, err := m.cache.GetEventConfig(ctx, tenantID, id)
		if err == nil {
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "event_manager"),
				logx.Field("operation", "get_with_fields"),
				logx.Field("event_id", id),
				logx.Field("tenant_id", tenantID),
				logx.Field("cache_hit", true),
			).Info("事件配置缓存命中")
			return cached, nil
		}
	}

	// 2. 缓存未命中，查询事件配置
	event, err := m.get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. 查询字段配置列表
	fields, err := m.listFieldsByEventID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 4. 组装返回
	result := &core.EventAggregate{
		EventConfig: *event,
		Fields:      fields,
	}

	// 5. 写入缓存
	if m.cache != nil {
		ttl := int(m.config.EventConfigCacheTTL.Seconds())
		_ = m.cache.SetEventConfig(ctx, result, ttl)
	}

	return result, nil
}

// ListWithFields 查询事件配置列表（包含字段）
func (m *eventManager) ListWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventAggregate, error) {
	// 1. 尝试从缓存获取
	if m.cache != nil {
		cached, err := m.cache.GetEventConfigList(ctx, tenantID, enabled)
		if err == nil {
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "event_manager"),
				logx.Field("operation", "list_with_fields"),
				logx.Field("tenant_id", tenantID),
				logx.Field("cache_hit", true),
			).Info("事件配置列表缓存命中")
			return cached, nil
		}
	}

	// 2. 缓存未命中，查询事件配置列表
	events, err := m.list(ctx, tenantID, enabled)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return []*core.EventAggregate{}, nil
	}

	// 3. 提取所有event_config_id
	eventIDs := make([]string, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	// 4. 批量查询所有字段配置（一次SQL查询，性能优化）
	fieldsMap, err := m.listFieldsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, err
	}

	// 5. 组装返回结果
	result := make([]*core.EventAggregate, len(events))
	for i, event := range events {
		fields, exists := fieldsMap[event.ID]
		if !exists {
			fields = []*core.FieldConfig{}
		}
		result[i] = &core.EventAggregate{
			EventConfig: *event,
			Fields:      fields,
		}
	}

	// 6. 写入缓存
	if m.cache != nil {
		ttl := int(m.config.EventConfigListCacheTTL.Seconds())
		_ = m.cache.SetEventConfigList(ctx, tenantID, enabled, result, ttl)
	}

	return result, nil
}

// Delete 删除事件配置（硬删除）
func (m *eventManager) Delete(ctx context.Context, id, tenantID string) error {
	// 硬删除事件配置
	deleteQuery := `
		DELETE FROM event_configs
		WHERE id = $1 AND tenant_id = $2
	`
	result, err := m.dbConn.ExecCtx(ctx, deleteQuery, id, tenantID)
	if err != nil {
		return fmt.Errorf("删除事件配置失败: %w", err)
	}

	// 检查是否有记录被删除
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrEventConfigNotFound
	}

	// 注意：字段配置会通过数据库外键级联删除（ON DELETE CASCADE）

	// 清除缓存
	if m.cache != nil {
		// 清除单个事件配置缓存
		_ = m.cache.DeleteEventConfig(ctx, tenantID, id)

		// 清除所有相关的列表缓存
		trueVal := true
		falseVal := false
		_ = m.cache.DeleteEventConfigList(ctx, tenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteEventConfigList(ctx, tenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteEventConfigList(ctx, tenantID, nil)       // enabled=all

		// 清除 flow_id 列表缓存
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, tenantID, &trueVal)  // enabled=true
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, tenantID, &falseVal) // enabled=false
		_ = m.cache.DeleteConfiguredFlowIDsList(ctx, tenantID, nil)       // enabled=all
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "delete"),
		logx.Field("event_config_id", id),
		logx.Field("tenant_id", tenantID),
	).Info("删除事件配置成功")

	return nil
}

// ListConfiguredFlowIDs 获取已配置的 flow_id 列表
func (m *eventManager) ListConfiguredFlowIDs(ctx context.Context, tenantID string, enabled *bool) ([]int, error) {
	// 1. 尝试从缓存获取
	if m.cache != nil {
		flowIDs, err := m.cache.GetConfiguredFlowIDsList(ctx, tenantID, enabled)
		if err == nil {
			return flowIDs, nil
		}
		// 缓存未命中或出错，继续查询数据库
	}

	// 2. 查询数据库（使用 DISTINCT 去重）
	query := `
		SELECT DISTINCT flow_id
		FROM event_configs
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}

	// 根据 enabled 参数过滤
	if enabled != nil {
		query += " AND enabled = $2"
		args = append(args, *enabled)
	}

	query += " ORDER BY flow_id"

	var flowIDs []int
	err := m.dbConn.QueryRowsCtx(ctx, &flowIDs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询已配置的 flow_id 列表失败: %w", err)
	}

	// 3. 写入缓存（TTL: 5分钟，与事件配置列表缓存一致）
	if m.cache != nil {
		_ = m.cache.SetConfiguredFlowIDsList(ctx, tenantID, enabled, flowIDs, 300) // 5分钟
	}

	return flowIDs, nil
}

// Update 更新事件配置（不含字段）
func (m *eventManager) Update(ctx context.Context, config *core.EventConfig) error {
	// 验证配置
	if config.ID == "" || config.TenantID == "" {
		return fmt.Errorf("%w: ID和租户ID不能为空", core.ErrInvalidEventConfig)
	}
	if err := m.validateEventConfig(config); err != nil {
		return err
	}

	// 查询旧配置验证权限
	var oldTenantID string
	checkQuery := "SELECT tenant_id FROM event_configs WHERE id = $1"
	err := m.dbConn.QueryRowCtx(ctx, &oldTenantID, checkQuery, config.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return core.ErrEventConfigNotFound
		}
		return fmt.Errorf("查询事件配置失败: %w", err)
	}

	// 验证租户ID一致
	if oldTenantID != config.TenantID {
		return fmt.Errorf("%w: 不允许修改租户ID", core.ErrInvalidEventConfig)
	}

	// 检查事件名称是否与其他事件冲突
	var conflictID string
	checkNameQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2 AND id != $3"
	err = m.dbConn.QueryRowCtx(ctx, &conflictID, checkNameQuery, config.Name, config.TenantID, config.ID)
	if err == nil {
		return fmt.Errorf("%w: 事件名称 '%s' 已被其他事件使用", core.ErrInvalidEventConfig, config.Name)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("检查事件名称失败: %w", err)
	}

	// 检查flow_id 是否与其他事件冲突
	checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2 AND id != $3"
	err = m.dbConn.QueryRowCtx(ctx, &conflictID, checkFlowQuery, config.FlowID, config.TenantID, config.ID)
	if err == nil {
		return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, config.FlowID, conflictID)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("检查flow_id 失败: %w", err)
	}

	// 更新数据
	updateQuery := `
		UPDATE event_configs
		SET name = $1, flow_id = $2, flow_title = $3, org_field_name = $4,
		    description = $5, enabled = $6, updated_by = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND tenant_id = $9
	`
	result, err := m.dbConn.ExecCtx(ctx, updateQuery,
		config.Name, config.FlowID, config.FlowTitle, config.OrgFieldName,
		config.Description, config.Enabled, config.UpdatedBy,
		config.ID, config.TenantID,
	)
	if err != nil {
		return fmt.Errorf("更新事件配置失败: %w", err)
	}

	// 检查是否有记录被更
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrEventConfigNotFound
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "update"),
		logx.Field("event_config_id", config.ID),
		logx.Field("tenant_id", config.TenantID),
	).Info("更新事件配置成功")

	return nil
}

// initTable 初始化事件配置相关表（event_configs + event_field_configs）（私有方法，在包初始化时调用）
func (m *eventManager) initTable(ctx context.Context) error {
	// 1. 检查并创建事件配置表
	var eventTableCount int
	err := m.dbConn.QueryRowCtx(ctx, &eventTableCount, CheckEventConfigTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查事件配置表存在性失败: %w", err)
	}

	if eventTableCount == 0 {
		_, err = m.dbConn.ExecCtx(ctx, CreateEventConfigTableSQL)
		if err != nil {
			return fmt.Errorf("创建事件配置表失败: %w", err)
		}
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "event_manager"),
			logx.Field("operation", "init_table"),
			logx.Field("table", EventConfigTableName),
		).Info("事件配置表初始化成功")
	}

	// 2. 检查并创建字段配置表
	var fieldTableCount int
	err = m.dbConn.QueryRowCtx(ctx, &fieldTableCount, CheckFieldConfigTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查字段配置表存在性失败: %w", err)
	}

	if fieldTableCount == 0 {
		_, err = m.dbConn.ExecCtx(ctx, CreateFieldConfigTableSQL)
		if err != nil {
			return fmt.Errorf("创建字段配置表失败: %w", err)
		}
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "event_manager"),
			logx.Field("operation", "init_table"),
			logx.Field("table", FieldConfigTableName),
		).Info("字段配置表初始化成功")
	}

	return nil
}
